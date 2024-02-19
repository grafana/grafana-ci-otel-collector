package githubactionsreceiver

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/google/go-github/v58/github"
	"github.com/grafana/grafana-ci-otel-collector/semconv"
	"github.com/grafana/grafana-ci-otel-collector/traceutils"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/collector/semconv/v1.9.0"

	"go.uber.org/zap"
)

type githubactionsWebhookHandler struct {
	logger *zap.Logger

	nextLogsConsumer  consumer.Logs
	nextTraceConsumer consumer.Traces
}

// This hould create the root span
func (d *githubactionsWebhookHandler) onWorkflowRunCompleted(deliveryID, eventName string, event *github.WorkflowRunEvent) error {
	d.logger.Debug("Got request", zap.String("deliveryID", deliveryID), zap.String("eventName", eventName))
	traces := ptrace.NewTraces()

	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()

	// Instrumentation library details
	scopeSpans.Scope().SetName("githubactionsreceiver")
	scopeSpans.Scope().SetVersion("0.1.0")

	resourceAttrs := resourceSpans.Resource().Attributes()

	// Wokflow details
	resourceAttrs.PutStr(conventions.AttributeServiceVersion, "0.1.0")
	resourceAttrs.PutStr(conventions.AttributeServiceName, *event.Workflow.Name)
	resourceAttrs.PutStr(semconv.AttributeCIVendor, semconv.AttributeCIVendorGHA)

	// Repository details
	resourceAttrs.PutStr(semconv.AttributeGitRepoName, *event.Repo.FullName)
	resourceAttrs.PutStr(semconv.AttributeGitHTTPURL, *event.Repo.HTMLURL)
	resourceAttrs.PutStr(semconv.AttributeGitSSHURL, *event.Repo.SSHURL)
	resourceAttrs.PutStr(semconv.AttributeGitBranchName, *event.WorkflowRun.HeadBranch)

	buildSpan := scopeSpans.Spans().AppendEmpty()
	traceId := deterministicTraceID(*event.WorkflowRun.ID, int64(*event.WorkflowRun.RunAttempt))
	buildSpan.SetTraceID(traceId)

	spanId := deterministicSpanID(*event.WorkflowRun.ID, int64(*event.WorkflowRun.RunAttempt))
	d.logger.Debug("onWorkflowRunCompleted", zap.String("spanId", spanId.String()),
		zap.Int64("workflowId", *event.Workflow.ID),
		zap.Int64("workflowRunId", *event.WorkflowRun.ID),
		zap.Int64("runAttempt", int64(*event.WorkflowRun.RunAttempt)),
	)
	buildSpan.SetSpanID(spanId)
	buildSpan.SetParentSpanID(pcommon.NewSpanIDEmpty())

	traceutils.SetStatus(*event.WorkflowRun.Conclusion, buildSpan)

	buildSpan.SetStartTimestamp(pcommon.Timestamp(event.WorkflowRun.GetRunStartedAt().UnixNano()))
	buildSpan.SetEndTimestamp(pcommon.Timestamp(event.WorkflowRun.GetUpdatedAt().UnixNano()))

	if d.nextTraceConsumer != nil {
		// TODO: To avoid needless work, traces should be prepared here
		err := d.nextTraceConsumer.ConsumeTraces(context.Background(), traces)
		if err != nil {
			return fmt.Errorf("cannot consume traces: %v", err)
		}
	}
	return nil
}

// This should create a span for every job in the workflow.
// ParentSpanId should be generated based on the workflowId
func (d *githubactionsWebhookHandler) onWorkflowJobCompleted(deliveryID, eventName string, event *github.WorkflowJobEvent) error {
	d.logger.Debug("Got request", zap.String("deliveryID", deliveryID), zap.String("eventName", eventName))
	traces := ptrace.NewTraces()
	logs := plog.NewLogs()

	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()

	// Instrumentation library details
	scopeSpans.Scope().SetName("githubactionsreceiver")
	scopeSpans.Scope().SetVersion("0.1.0")

	resourceAttrs := resourceSpans.Resource().Attributes()

	// Wokflow details
	resourceAttrs.PutStr(conventions.AttributeServiceVersion, "0.1.0")
	resourceAttrs.PutStr(conventions.AttributeServiceName, *event.WorkflowJob.Name)

	buildSpan := scopeSpans.Spans().AppendEmpty()

	traceId := deterministicTraceID(*event.WorkflowJob.RunID, *event.WorkflowJob.RunAttempt)

	buildSpan.SetTraceID(traceId)
	buildSpan.SetSpanID(traceutils.NewSpanID())

	// parentSpanId is based on the workflowId
	spanId := deterministicSpanID(*event.WorkflowJob.RunID, *event.WorkflowJob.RunAttempt)
	buildSpan.SetParentSpanID(spanId)
	d.logger.Debug("onWorkflowJobCompleted", zap.String("spanId", spanId.String()),
		zap.Int64("workflowId", *event.WorkflowJob.ID),
		zap.Int64("workflowRunId", *event.WorkflowJob.RunID),
		zap.Int64("runAttempt", int64(*event.WorkflowJob.RunAttempt)),
	)

	event.WorkflowJob.GetStartedAt()

	spanEvent := buildSpan.Events().AppendEmpty()
	spanEvent.SetName("job_started")
	spanEvent.SetTimestamp(pcommon.Timestamp(event.WorkflowJob.GetStartedAt().UnixNano()))
	buildSpan.SetStartTimestamp(pcommon.Timestamp(event.WorkflowJob.CreatedAt.UnixNano()))
	buildSpan.SetEndTimestamp(pcommon.Timestamp(event.WorkflowJob.CompletedAt.UnixNano()))

	if d.nextTraceConsumer != nil {
		// TODO: To avoid needless work, traces should be prepared here
		err := d.nextTraceConsumer.ConsumeTraces(context.Background(), traces)
		if err != nil {
			return fmt.Errorf("cannot consume traces: %v", err)
		}
	}
	if d.nextLogsConsumer != nil {
		// TODO: To avoid needless work, logs should be prepared here
		err := d.nextLogsConsumer.ConsumeLogs(context.Background(), logs)
		if err != nil {
			return fmt.Errorf("cannot consume logs: %v", err)
		}
	}
	return nil
}

func generateLogs(traceId pcommon.TraceID, stepSpanId pcommon.SpanID) (plog.Logs, error) {
	logs := plog.NewLogs()

	return logs, nil
}

func deterministicSpanID(workflowRunID, attempt int64) pcommon.SpanID {
	md5hash := md5.New()
	md5hash.Write([]byte(strconv.FormatInt(workflowRunID, 10) + "_" + strconv.FormatInt(attempt, 10)))

	// convert the hash value to a string
	md5string := hex.EncodeToString(md5hash.Sum(nil))

	return pcommon.SpanID([]byte(md5string[0:16]))
}

// Generates a unique trace id based on the workflowId, workflowRunId and attempt number
func deterministicTraceID(workflowRunID, attempt int64) pcommon.TraceID {
	md5hash := md5.New()
	md5hash.Write([]byte(strconv.FormatInt(workflowRunID, 10) + "_" + strconv.FormatInt(attempt, 10)))

	// convert the hash value to a string
	md5string := hex.EncodeToString(md5hash.Sum(nil))

	return pcommon.TraceID([]byte(md5string[0:16]))
}
