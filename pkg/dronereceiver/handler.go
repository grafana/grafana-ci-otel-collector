package dronereceiver

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	drone "github.com/drone/drone-go/drone"
	"github.com/grafana/grafana-ci-otel-collector/traceutils"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/collector/semconv/v1.9.0"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type RepoEvt struct {
	drone.Repo
	Build *drone.Build `json:"build"`
}

type WebhookEvent struct {
	Action string   `json:"action"`
	Repo   *RepoEvt `json:"repo"`
}

type droneWebhookHandler struct {
	droneClient drone.Client
	logger      *zap.Logger

	reposConfig map[string][]string

	nextLogsConsumer  consumer.Logs
	nextTraceConsumer consumer.Traces
}

const CI_KIND = "ci.kind"
const CI_STAGE = "ci.stage"
const CI_STEP = "ci.step"

func (d *droneWebhookHandler) handler(resp http.ResponseWriter, req *http.Request) {
	d.logger.Debug("Got request")

	body, err := io.ReadAll(req.Body)
	if err != nil {
		// TODO: handle this
		return
	}

	var evt WebhookEvent
	err = json.Unmarshal(body, &evt)
	if err != nil {
		// TODO: handle this
		return
	}

	// Skip unfinished builds (i.e. builds that are still running)
	// In theory, according to the docs in https://docs.drone.io/webhooks/examples/, build.Action should be "completed" when a build is completed.
	// However, in practice, it seems that build.Action is always "updated" as per https://github.com/harness/drone/issues/2977.
	// so, we check if build.Finished is set to a non-zero value to determine if the build is finished.
	// However this appears to be sent when a build completes; The structure however changes slightly as in
	// the `repo.build` seems to be absent in the first one.
	// TODO: Revisit this, we may not need the Finished check.
	if evt.Repo.Build == nil || evt.Repo.Build.Finished == 0 {
		return
	}

	repo := evt.Repo
	build := evt.Repo.Build

	// Skip traces for repos that are not enabled
	allowedBranches, ok := d.reposConfig[evt.Repo.Slug]
	if !ok {
		d.logger.Info("repo not enabled", zap.String("repo", evt.Repo.Slug))
		return
	}

	// Skip traces for branches that are not configured
	if !slices.Contains[string](allowedBranches, repo.Branch) {
		d.logger.Info("branch not enabled", zap.String("branch", repo.Branch))
		return
	}

	traces := ptrace.NewTraces()
	logs := plog.NewLogs()

	traceId := traceutils.NewTraceID()
	buildId := traceutils.NewSpanID()

	d.logger.Debug("generating trace",
		zap.String("traceId", traceId.String()),
		zap.Int64("build.id", build.ID),
		zap.Int64("build.number", build.Number),
		zap.Int64("build.Created", build.Created*1000000000),
		zap.Int64("build.Finished", build.Finished*1000000000),
		zap.Int("build.Stages", len(build.Stages)),
	)

	resourceSpan := traces.ResourceSpans().AppendEmpty()
	scopeSpan := resourceSpan.ScopeSpans().AppendEmpty()

	scopeSpan.Scope().SetName("dronereceiver")
	scopeSpan.Scope().SetVersion("0.1.0")

	resourceAttrs := resourceSpan.Resource().Attributes()
	resourceAttrs.PutStr(conventions.AttributeServiceVersion, "0.1.0")
	resourceAttrs.PutStr(conventions.AttributeServiceName, "drone")
	resourceAttrs.PutStr("repo.name", repo.Slug)
	resourceAttrs.PutStr("repo.branch", repo.Branch)

	buildSpan := scopeSpan.Spans().AppendEmpty()
	buildSpan.SetTraceID(traceId)
	buildSpan.SetSpanID(buildId)
	buildSpan.SetParentSpanID(pcommon.NewSpanIDEmpty())
	buildSpan.Attributes().PutStr(CI_KIND, "build")

	buildSpan.Attributes().PutInt("build.number", build.Number)
	buildSpan.Attributes().PutInt("build.id", build.ID)

	// TODO: the trigger seems to be the username, should we keep it?
	buildSpan.Attributes().PutStr("build.trigger", build.Trigger)

	// Set build title and message
	// The root span name will be the build title if it is set, otherwise it will be the build message.
	buildSpan.Attributes().PutStr("build.title", build.Title)
	buildSpan.Attributes().PutStr("build.message", build.Message)

	if build.Title != "" {
		buildSpan.SetName(build.Title)
	} else {
		buildSpan.SetName(build.Message)
	}

	traceutils.SetStatus(build.Status, buildSpan)

	buildSpan.SetStartTimestamp(pcommon.Timestamp(build.Created * 1000000000))
	buildSpan.SetEndTimestamp(pcommon.Timestamp(build.Finished * 1000000000))

	for _, stage := range build.Stages {
		stageId := traceutils.NewSpanID()
		stageSpans := resourceSpan.ScopeSpans().AppendEmpty()
		stageSpan := stageSpans.Spans().AppendEmpty()

		stageSpan.Attributes().PutStr(conventions.AttributeServiceName, stage.Name)
		stageSpan.Attributes().PutInt("stage.number", int64(stage.Number))

		traceutils.SetStatus(stage.Status, stageSpan)
		stageSpan.SetName(stage.Name)
		stageSpan.SetTraceID(traceId)
		stageSpan.SetSpanID(stageId)
		stageSpan.SetParentSpanID(buildId)
		stageSpan.Attributes().PutStr(CI_KIND, "stage")

		stageSpan.SetStartTimestamp(pcommon.Timestamp(stage.Started * 1000000000))
		stageSpan.SetEndTimestamp(pcommon.Timestamp(stage.Stopped * 1000000000))

		for _, step := range stage.Steps {
			if step.Status == "skipped" {
				continue
			}

			stepSpanId := traceutils.NewSpanID()
			stepSpan := stageSpans.Spans().AppendEmpty()
			stepSpan.SetTraceID(traceId)
			stepSpan.SetParentSpanID(stageId)
			stepSpan.SetSpanID(stepSpanId)
			stepSpan.Attributes().PutStr(CI_KIND, "step")
			stepSpan.Attributes().PutStr(CI_STAGE, stage.Name)
			stepSpan.Attributes().PutInt("step.number", int64(step.Number))

			traceutils.SetStatus(step.Status, stepSpan)

			stepSpan.SetName(step.Name)

			stepSpan.SetStartTimestamp(pcommon.Timestamp(step.Started * 1000000000))
			stepSpan.SetEndTimestamp(pcommon.Timestamp(step.Stopped * 1000000000))

			newLogs, err := generateLogs(d.droneClient, repo.Repo, *build, *stage, *step, traceId, stepSpanId)
			if err != nil {
				d.logger.Error("error retrieving logs", zap.Error(err))
				continue
			}

			newLogs.ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
		}
	}

	if d.nextTraceConsumer != nil {
		d.nextTraceConsumer.ConsumeTraces(req.Context(), traces)
	}
	if d.nextLogsConsumer != nil {
		d.nextLogsConsumer.ConsumeLogs(req.Context(), logs)
	}
}

func generateLogs(drone drone.Client, repo drone.Repo, build drone.Build, stage drone.Stage, step drone.Step, traceId pcommon.TraceID, stepSpanId pcommon.SpanID) (plog.Logs, error) {
	logs := plog.NewLogs()

	lines, err := drone.Logs(repo.Namespace, repo.Name, int(build.Number), stage.Number, step.Number)
	if err != nil {
		return logs, err
	}

	log := logs.ResourceLogs().AppendEmpty()
	logScope := log.ScopeLogs().AppendEmpty()

	now := pcommon.NewTimestampFromTime(time.Now())

	prevLineTimestamp := int64(0)
	delta := int64(0)
	for _, line := range lines {
		if line.Timestamp == prevLineTimestamp {
			delta++
		} else {
			delta = 0
			prevLineTimestamp = line.Timestamp
		}

		record := logScope.LogRecords().AppendEmpty()
		record.SetTraceID(traceId)
		record.SetSpanID(stepSpanId)

		record.SetObservedTimestamp(now)
		record.SetTimestamp(pcommon.Timestamp((step.Started+line.Timestamp)*1000000000 + delta))
		record.Attributes().PutStr(CI_STAGE, stage.Name)
		record.Attributes().PutStr(CI_STEP, step.Name)
		record.Attributes().PutInt("build.number", build.Number)
		record.Body().SetStr(line.Message)
	}

	return logs, nil

}
