// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubactionsreceiver

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v77/github"
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
	jobs, filesByJob := extractJobsAndFilesFromZip(zipReader, log)

	log.Debug("Extracted jobs and files from zip", zap.Int("job_count", len(jobs)))
	log.Debug("Job names", zap.Any("job_names", jobs))

	for i, jobName := range jobs {
		log.Debug("Processing job", zap.Int("job_index", i+1), zap.String("job_name", jobName))
		jobFiles := filesByJob[jobName]
		processJobLogs(jobName, jobFiles, resourceLogs, traceID, e, withTraceInfo, log)
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
		if err := tmpFile.Close(); err != nil {
			logger.Warn("Failed to close temp file", zap.Error(err))
		}
		if err := zipReader.Close(); err != nil {
			logger.Warn("Failed to close zip reader", zap.Error(err))
		}
		if err := os.Remove(tmpFile.Name()); err != nil {
			logger.Warn("Failed to remove temp file", zap.Error(err), zap.String("file", tmpFile.Name()))
		}
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
		if closeErr := out.Close(); closeErr != nil {
			logger.Warn("Failed to close temp file after HTTP error", zap.Error(closeErr))
		}
		logger.Error("Failed to download logs", zap.Error(err))
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Warn("Failed to close response body", zap.Error(err))
		}
	}()

	if _, err := io.Copy(out, resp.Body); err != nil {
		if closeErr := out.Close(); closeErr != nil {
			logger.Warn("Failed to close temp file after copy error", zap.Error(closeErr))
		}
		logger.Error("Failed to save logs to temp file", zap.Error(err))
		return nil, err
	}

	return out, nil
}

func extractJobsAndFilesFromZip(zipReader *zip.Reader, logger *zap.Logger) ([]string, map[string][]*zip.File) {
	// Pre-allocate maps with reasonable capacity based on typical GitHub Actions workflows
	estimatedJobs := min(len(zipReader.File)/10, 50) // Estimate ~10 files per job, max 50 jobs
	jobSet := make(map[string]struct{}, estimatedJobs)
	filesByJob := make(map[string][]*zip.File, estimatedJobs)

	logger.Debug("Total files in zip", zap.Int("file_count", len(zipReader.File)))

	for i, f := range zipReader.File {
		logger.Debug("Zip entry", zap.Int("entry_index", i), zap.String("file_name", f.Name), zap.Bool("is_dir", f.FileInfo().IsDir()), zap.Uint64("uncompressed_size", f.UncompressedSize64))

		if f.FileInfo().IsDir() {
			// Old format: directories
			jobName := strings.TrimSuffix(f.Name, "/")
			jobSet[jobName] = struct{}{}
		} else {
			if slashIdx := strings.IndexByte(f.Name, '/'); slashIdx != -1 {
				jobName := f.Name[:slashIdx]
				logger.Debug("Extracted job name", zap.String("job_name", jobName), zap.String("file_name", f.Name))
				jobSet[jobName] = struct{}{}
				if filesByJob[jobName] == nil {
					// Pre-allocate slice with reasonable capacity (typical job has ~10 steps)
					filesByJob[jobName] = make([]*zip.File, 0, 10)
				}
				filesByJob[jobName] = append(filesByJob[jobName], f)
				logger.Debug("Added job", zap.String("job_name", jobName), zap.Int("total_jobs", len(jobSet)))
			} else {
				logger.Debug("File contains no '/', skipping job extraction", zap.String("file_name", f.Name))
			}
		}
	}

	// Convert map to sorted slice for consistent ordering
	jobs := slices.Sorted(maps.Keys(jobSet))

	logger.Debug("Extracted jobs and files", zap.Int("job_count", len(jobs)), zap.Int("total_files", len(zipReader.File)))
	return jobs, filesByJob
}

func processJobLogs(jobName string, files []*zip.File, resourceLogs plog.ResourceLogs, traceID pcommon.TraceID, e *github.WorkflowRunEvent, withTraceInfo bool, logger *zap.Logger) {
	jobLogsScope := resourceLogs.ScopeLogs().AppendEmpty()
	jobLogsScope.Scope().Attributes().PutStr("ci.github.workflow.job.name", jobName)

	// Reuse a single logEntryBuilder for all files in this job
	var builder logEntryBuilder

	for _, logFile := range files {
		logger.Debug("Processing log file",
			zap.String("job_name", jobName),
			zap.String("file_name", logFile.Name))
		processLogFile(logFile, jobName, jobLogsScope, traceID, e, withTraceInfo, logger, &builder)
	}

	logger.Debug("Completed job log processing",
		zap.String("job_name", jobName),
		zap.Int("log_records", jobLogsScope.LogRecords().Len()))
}

func processLogFile(logFile *zip.File, jobName string, jobLogsScope plog.ScopeLogs, traceID pcommon.TraceID, e *github.WorkflowRunEvent, withTraceInfo bool, logger *zap.Logger, builder *logEntryBuilder) {
	stepNumber, err := extractStepNumberFromFileName(logFile.Name, jobName)
	if err != nil {
		if strings.Contains(err.Error(), "skipping system file") {
			logger.Debug("Skipping system file", zap.String("filename", logFile.Name))
		} else {
			logger.Error("Invalid step number in filename", zap.String("filename", logFile.Name), zap.Error(err))
		}
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
	defer func() {
		if err := fileReader.Close(); err != nil {
			steplog.Warn("Failed to close file reader", zap.Error(err))
		}
	}()

	processLogEntries(fileReader, jobLogsScope, spanID, traceID, stepNumber, withTraceInfo, steplog, builder)
}

func extractStepNumberFromFileName(fileName, jobName string) (int, error) {
	prefixLen := len(jobName) + 1 // +1 for the "/"
	if len(fileName) <= prefixLen {
		return 0, fmt.Errorf("filename %q does not contain job prefix %q/", fileName, jobName)
	}
	baseName := fileName[prefixLen:]

	// Skip system files that don't follow the step number pattern
	// Seems like it's a bug in with GitHub Actions since it arbitrarily started
	// appearing in Aug 2025.
	if baseName == "system.txt" {
		return 0, fmt.Errorf("skipping system file: %q", fileName)
	}

	underscoreIdx := strings.IndexByte(baseName, '_')
	if underscoreIdx == -1 {
		return strconv.Atoi(baseName)
	}
	return strconv.Atoi(baseName[:underscoreIdx])
}

func processLogEntries(reader io.Reader, jobLogsScope plog.ScopeLogs, spanID pcommon.SpanID, traceID pcommon.TraceID, stepNumber int, withTraceInfo bool, logger *zap.Logger, builder *logEntryBuilder) {
	scanner := bufio.NewScanner(reader)

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxLogEntryBytes)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		parsedTime, rest, ok := parseTimestamp(line, logger)
		if ok {
			if builder.hasCurrentEntry {
				finalizeLogEntry(builder, jobLogsScope, spanID, traceID, stepNumber, withTraceInfo)
			}

			// Reuse the builder
			builder.currentParsedTime = parsedTime
			builder.currentStepNumber = int64(stepNumber)
			builder.hasCurrentEntry = true
			builder.currentBody.Reset()
			builder.currentBody.Write(rest)
		} else {
			if !builder.hasCurrentEntry {
				logger.Error("Orphaned log line without preceding timestamp", zap.String("line", string(line)))
				continue
			}

			if builder.currentBody.Len()+len(line)+1 > maxLogEntryBytes {
				logger.Warn("Skipping line due to size limit", zap.Int("lineSize", len(line)))
				continue
			}

			builder.currentBody.WriteByte('\n')
			builder.currentBody.Write(line)
		}
	}

	if builder.hasCurrentEntry {
		finalizeLogEntry(builder, jobLogsScope, spanID, traceID, stepNumber, withTraceInfo)
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

func parseTimestamp(line []byte, logger *zap.Logger) (time.Time, []byte, bool) {
	var parsedTime time.Time
	var err error

	spaceIdx := bytes.IndexByte(line, ' ')
	if spaceIdx == -1 {
		return time.Time{}, nil, false
	}

	ts := line[:spaceIdx]
	rest := line[spaceIdx+1:]

	// Handle BOM and trim spaces
	tsStr := strings.TrimSpace(strings.TrimPrefix(string(ts), "\uFEFF"))
	parsedTime, err = time.Parse(time.RFC3339, tsStr)
	if err != nil {
		logger.Debug("Failed to parse timestamp", zap.String("timestamp", tsStr), zap.Error(err))
		return time.Time{}, nil, false
	}

	return parsedTime, rest, true
}
