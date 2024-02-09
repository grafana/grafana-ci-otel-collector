package githubactionsreceiver

import (
	"context"
	"fmt"

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

func (d *githubactionsWebhookHandler) onWorkflowRunCompleted(deliveryID, eventName string, event *github.WorkflowRunEvent) error {
	d.logger.Debug("Got request")
	traces := ptrace.NewTraces()
	logs := plog.NewLogs()

	resourceSpans := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpans.ScopeSpans().AppendEmpty()

	scopeSpans.Scope().SetName("githubactionsreceiver")
	scopeSpans.Scope().SetVersion("0.1.0")

	resourceAttrs := resourceSpans.Resource().Attributes()
	resourceAttrs.PutStr(conventions.AttributeServiceVersion, "0.1.0")
	resourceAttrs.PutStr(conventions.AttributeServiceName, *event.Workflow.Name)
	resourceAttrs.PutStr(semconv.AttributeCIVendor, semconv.AttributeCIVendorGHA)

	resourceAttrs.PutStr(semconv.AttributeGitRepoName, *event.Repo.FullName)
	resourceAttrs.PutStr(semconv.AttributeGitHTTPURL, *event.Repo.HTMLURL)
	resourceAttrs.PutStr(semconv.AttributeGitSSHURL, *event.Repo.SSHURL)
	resourceAttrs.PutStr(semconv.AttributeGitBranchName, *event.WorkflowRun.HeadBranch)

	buildSpan := scopeSpans.Spans().AppendEmpty()
	buildSpan.SetTraceID(traceutils.NewTraceID())
	buildSpan.SetSpanID(traceutils.NewSpanID())
	buildSpan.SetParentSpanID(pcommon.NewSpanIDEmpty())
	// buildAttributes := buildSpan.Attributes()

	// buildAttributes.PutStr(semconv.AttributeDroneWorkflowEvent, build.Event)

	// buildAttributes.PutInt(semconv.AttributeDroneBuildNumber, build.Number)
	// buildAttributes.PutInt(semconv.AttributeDroneBuildID, build.ID)

	traceutils.SetStatus(*event.WorkflowRun.Status, buildSpan)

	event.WorkflowRun.GetRunStartedAt()

	buildSpan.SetStartTimestamp(pcommon.Timestamp(event.WorkflowRun.GetRunStartedAt().UnixNano()))
	buildSpan.SetEndTimestamp(pcommon.Timestamp(event.WorkflowRun.GetUpdatedAt().UnixNano()))

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
