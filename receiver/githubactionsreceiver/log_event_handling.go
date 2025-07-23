// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubactionsreceiver

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v62/github"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

const (
	// Logs larger than this will be truncated
	maxLogEntryBytes = 1 * 1024 * 1024 // 1 MB
)

type logEntryBuilder struct {
	currentBody       strings.Builder
	currentParsedTime time.Time
	currentStepNumber int64
	hasCurrentEntry   bool
}

func (b *logEntryBuilder) reset() {
	b.currentBody.Reset()
	b.currentParsedTime = time.Time{}
	b.currentStepNumber = 0
	b.hasCurrentEntry = false
}

func eventToLogs(event interface{}, config *Config, ghClient *github.Client, logger *zap.Logger, withTraceInfo bool) (*plog.Logs, error) {
	e, ok := event.(*github.WorkflowRunEvent)
	if !ok {
		return nil, nil
	}

	log := enrichLogger(logger, e)
	log.Debug("Processing WorkflowRunEvent for logs",
		zap.String("status", e.GetWorkflowRun().GetStatus()),
		zap.String("conclusion", e.GetWorkflowRun().GetConclusion()),
		zap.Int64("run_id", e.GetWorkflowRun().GetID()),
		zap.Int("run_attempt", e.GetWorkflowRun().GetRunAttempt()))

	if e.GetWorkflowRun().GetStatus() != "completed" {
		log.Debug("Run not completed, skipping")
		return nil, nil
	}

	zipReader, cleanup, err := getWorkflowRunLogsZip(context.Background(), ghClient, e, log)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	setWorkflowRunEventAttributes(resourceLogs.Resource().Attributes(), e, config)

	traceID, _ := generateTraceID(e.GetWorkflowRun().GetID(), e.GetWorkflowRun().GetRunAttempt())
	jobs, files := extractJobsAndFilesFromZip(zipReader, log)

	log.Debug("Extracted jobs and files from zip", zap.Int("job_count", len(jobs)), zap.Int("file_count", len(files)))
	log.Debug("Job names", zap.Any("job_names", jobs))

	for i, jobName := range jobs {
		log.Debug("Processing job", zap.Int("job_index", i+1), zap.String("job_name", jobName))
		processJobLogs(jobName, files, resourceLogs, traceID, e, withTraceInfo, log)
		log.Debug("Completed job", zap.Int("job_index", i+1), zap.String("job_name", jobName))
	}

	log.Debug("All jobs processed", zap.Int("total_resource_logs", logs.ResourceLogs().Len()))
	return &logs, nil
}

func enrichLogger(logger *zap.Logger, e *github.WorkflowRunEvent) *zap.Logger {
	log := logger.With(zap.Int64("workflow_run_id", e.GetWorkflowRun().GetID()))
	if repo := e.GetRepo(); repo != nil {
		log = log.With(
			zap.String("repo_owner", repo.GetOwner().GetLogin()),
			zap.String("repo_name", repo.GetName()),
		)
	}
	if workflow := e.GetWorkflow(); workflow != nil {
		log = log.With(zap.String("workflow_path", workflow.GetPath()))
	}
	return log.With(
		zap.Int("workflow_run_attempt", e.GetWorkflowRun().GetRunAttempt()),
		zap.String("workflow_run_name", e.GetWorkflowRun().GetName()),
		zap.String("workflow_url", e.GetWorkflowRun().GetWorkflowURL()),
	)
}

func getWorkflowRunLogsZip(ctx context.Context, ghClient *github.Client, e *github.WorkflowRunEvent, logger *zap.Logger) (*zip.Reader, func(), error) {
	url, _, err := ghClient.Actions.GetWorkflowRunAttemptLogs(
		ctx,
		e.GetRepo().GetOwner().GetLogin(),
		e.GetRepo().GetName(),
		e.GetWorkflowRun().GetID(),
		e.GetWorkflowRun().GetRunAttempt(),
		10,
	)
	if err != nil {
		logger.Error("Failed to get logs", zap.Error(err))
		return nil, nil, err
	}

	tmpFile, err := downloadLogsToTempFile(url.String(), logger)
	if err != nil {
		return nil, nil, err
	}

	zipReader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		logger.Error("Failed to open zip file", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to open zip file: %w", err)
	}

	cleanup := func() {
		tmpFile.Close()
		zipReader.Close()
		os.Remove(tmpFile.Name())
	}

	return &zipReader.Reader, cleanup, nil
}

func downloadLogsToTempFile(url string, logger *zap.Logger) (*os.File, error) {
	out, err := os.CreateTemp("", "gh-logs-")
	if err != nil {
		logger.Error("Failed to create temp file", zap.Error(err))
		return nil, err
	}

	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		logger.Error("Failed to download logs", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		logger.Error("Failed to save logs to temp file", zap.Error(err))
		return nil, err
	}

	return out, nil
}

func extractJobsAndFilesFromZip(zipReader *zip.Reader, logger *zap.Logger) ([]string, []*zip.File) {
	var jobs []string
	var files []*zip.File

	logger.Debug("Total files in zip", zap.Int("file_count", len(zipReader.File)))

	for i, f := range zipReader.File {
		logger.Debug("Zip entry", zap.Int("entry_index", i), zap.String("file_name", f.Name), zap.Bool("is_dir", f.FileInfo().IsDir()), zap.Uint64("uncompressed_size", f.UncompressedSize64))

		if f.FileInfo().IsDir() {
			// Old format: directories
			jobs = append(jobs, strings.TrimSuffix(f.Name, "/"))
		} else {
			files = append(files, f)
			if strings.Contains(f.Name, "/") {
				jobName := strings.Split(f.Name, "/")[0]
				logger.Debug("Extracted job name", zap.String("job_name", jobName), zap.String("file_name", f.Name))
				jobs = append(jobs, jobName)
				logger.Debug("Added job", zap.String("job_name", jobName), zap.Int("total_jobs", len(jobs)))
			} else {
				logger.Debug("File contains no '/', skipping job extraction", zap.String("file_name", f.Name))
			}
		}
	}

	logger.Debug("Extracted jobs and files", zap.Int("job_count", len(jobs)), zap.Int("file_count", len(files)))
	return jobs, files
}

func processJobLogs(jobName string, files []*zip.File, resourceLogs plog.ResourceLogs, traceID pcommon.TraceID, e *github.WorkflowRunEvent, withTraceInfo bool, logger *zap.Logger) {
	jobLogsScope := resourceLogs.ScopeLogs().AppendEmpty()
	jobLogsScope.Scope().Attributes().PutStr("ci.github.workflow.job.name", jobName)

	for _, logFile := range files {
		if !strings.HasPrefix(logFile.Name, jobName+"/") {
			continue
		}
		logger.Debug("Processing log file",
			zap.String("job_name", jobName),
			zap.String("file_name", logFile.Name))
		processLogFile(logFile, jobName, jobLogsScope, traceID, e, withTraceInfo, logger)
	}

	logger.Debug("Completed job log processing",
		zap.String("job_name", jobName),
		zap.Int("log_records", jobLogsScope.LogRecords().Len()))
}

func processLogFile(logFile *zip.File, jobName string, jobLogsScope plog.ScopeLogs, traceID pcommon.TraceID, e *github.WorkflowRunEvent, withTraceInfo bool, logger *zap.Logger) {
	stepNumber, err := extractStepNumberFromFileName(logFile.Name, jobName)
	if err != nil {
		logger.Error("Invalid step number in filename", zap.String("filename", logFile.Name), zap.Error(err))
		return
	}

	steplog := logger.With(zap.Int("step_number", stepNumber))
	spanID, err := generateStepSpanID(e.GetWorkflowRun().GetID(), e.GetWorkflowRun().GetRunAttempt(), jobName, int64(stepNumber))
	if err != nil {
		steplog.Error("Failed to generate span ID", zap.Error(err))
		return
	}

	fileReader, err := logFile.Open()
	if err != nil {
		steplog.Error("Failed to open log file", zap.Error(err))
		return
	}
	defer fileReader.Close()

	processLogEntries(fileReader, jobLogsScope, spanID, traceID, stepNumber, withTraceInfo, steplog)
}

func extractStepNumberFromFileName(fileName, jobName string) (int, error) {
	baseName := strings.TrimPrefix(fileName, jobName+"/")
	parts := strings.SplitN(baseName, "_", 2)
	return strconv.Atoi(parts[0])
}

func processLogEntries(reader io.Reader, jobLogsScope plog.ScopeLogs, spanID pcommon.SpanID, traceID pcommon.TraceID, stepNumber int, withTraceInfo bool, logger *zap.Logger) {
	var builder logEntryBuilder
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parsedTime, rest, ok := parseTimestamp(line, logger)
		if ok {
			if builder.hasCurrentEntry {
				finalizeLogEntry(&builder, jobLogsScope, spanID, traceID, stepNumber, withTraceInfo)
			}
			builder = logEntryBuilder{
				currentParsedTime: parsedTime,
				currentStepNumber: int64(stepNumber),
				hasCurrentEntry:   true,
			}
			builder.currentBody.WriteString(rest)
		} else {
			if !builder.hasCurrentEntry {
				logger.Error("Orphaned log line without preceding timestamp", zap.String("line", line))
				continue
			}

			if builder.currentBody.Len()+len(line)+1 > maxLogEntryBytes {
				logger.Warn("Skipping line due to size limit", zap.Int("lineSize", len(line)))
				continue
			}

			builder.currentBody.WriteString("\n")
			builder.currentBody.WriteString(line)
		}
	}

	if builder.hasCurrentEntry {
		finalizeLogEntry(&builder, jobLogsScope, spanID, traceID, stepNumber, withTraceInfo)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("Error reading log file", zap.Error(err))
	}
}

func finalizeLogEntry(builder *logEntryBuilder, jobLogsScope plog.ScopeLogs, spanID pcommon.SpanID, traceID pcommon.TraceID, stepNumber int, withTraceInfo bool) {
	record := jobLogsScope.LogRecords().AppendEmpty()
	if withTraceInfo {
		record.SetSpanID(spanID)
		record.SetTraceID(traceID)
	}
	record.Attributes().PutInt("ci.github.workflow.job.step.number", int64(stepNumber))
	record.SetTimestamp(pcommon.NewTimestampFromTime(builder.currentParsedTime))
	record.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	record.Body().SetStr(builder.currentBody.String())
	builder.reset()
}

func parseTimestamp(line string, logger *zap.Logger) (time.Time, string, bool) {
	var parsedTime time.Time
	var err error

	ts, rest, ok := strings.Cut(line, " ")
	if !ok {
		return time.Time{}, "", false
	}

	parsedTime, err = time.Parse(time.RFC3339, strings.TrimSpace(strings.TrimPrefix(ts, "\uFEFF")))
	if err != nil {
		logger.Debug("Failed to parse timestamp", zap.String("timestamp", ts), zap.Error(err))
		return time.Time{}, "", false
	}

	return parsedTime, rest, true
}
