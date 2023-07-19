package dronereceiver

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	drone "github.com/drone/drone-go/drone"
	"github.com/google/uuid"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/collector/semconv/v1.9.0"
	"go.uber.org/zap"
)

type DroneCompletedBuild struct {
	Action string      `json:"action"`
	Repo   drone.Repo  `json:"repo"`
	Build  drone.Build `json:"build"`
}

type droneWebhookHandler struct {
	droneClient drone.Client
	logger      *zap.Logger

	nextLogsConsumer  consumer.Logs
	nextTraceConsumer consumer.Traces
}

const CI_KIND = "ci.kind"
const CI_STAGE = "ci.stage"
const CI_STEP = "ci.step"

func getOtelExitCode(code int) ptrace.StatusCode {
	if code == 0 {
		return ptrace.StatusCodeOk
	}

	if code == 1 {
		return ptrace.StatusCodeError
	}

	return ptrace.StatusCodeUnset
}

func (d *droneWebhookHandler) handler(resp http.ResponseWriter, req *http.Request) {
	// TODO: this is just a stub for now
	d.logger.Info("Got request")

	traces := ptrace.NewTraces()
	logs := plog.NewLogs()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		// TODO: handle this
		return
	}

	var completedBuild DroneCompletedBuild
	err = json.Unmarshal(body, &completedBuild)
	if err != nil {
		// TODO: handle this
		return
	}

	build := completedBuild.Build
	repo := completedBuild.Repo

	if build.Finished == 0 {
		return
	}

	buildCode := 0

	if build.Error != "" {
		buildCode = 1
	}

	traceId := NewTraceID()
	buildId := NewSpanID()

	d.logger.Debug("generating trace",
		zap.String("traceId", traceId.String()),
		zap.Int64("build.id", build.Number),
		zap.Int64("build.Created", build.Created*1000000000),
		zap.Int64("build.Finished", build.Finished*1000000000),
		zap.Int("build.Stages", len(build.Stages)),
	)

	resourceSpan := traces.ResourceSpans().AppendEmpty()
	scopeSpans := resourceSpan.ScopeSpans().AppendEmpty()

	scopeSpans.Scope().SetName("dronereceiver")
	scopeSpans.Scope().SetVersion("0.1.0")

	resourceAttrs := resourceSpan.Resource().Attributes()
	resourceAttrs.PutStr(conventions.AttributeServiceVersion, "0.1.0")
	resourceAttrs.PutStr(conventions.AttributeServiceName, "drone")
	resourceAttrs.PutInt("build.number", build.Number)
	resourceAttrs.PutInt("build.id", build.ID)
	resourceAttrs.PutStr("repo.name", repo.Name)
	resourceAttrs.PutStr("repo.branch", repo.Branch)

	buildSpan := scopeSpans.Spans().AppendEmpty()
	buildSpan.SetTraceID(traceId)
	buildSpan.SetSpanID(buildId)
	buildSpan.SetParentSpanID(pcommon.NewSpanIDEmpty())
	buildSpan.Attributes().PutStr(CI_KIND, "build")
	buildSpan.Status().SetCode(getOtelExitCode(buildCode))

	//buildSpan.SetName(build.Title)

	buildSpan.SetStartTimestamp(pcommon.Timestamp(build.Created * 1000000000))
	buildSpan.SetEndTimestamp(pcommon.Timestamp(build.Finished * 1000000000))

	for _, stage := range build.Stages {
		stageId := NewSpanID()
		stageSpans := resourceSpan.ScopeSpans().AppendEmpty()
		stageSpan := stageSpans.Spans().AppendEmpty()

		stageSpan.Attributes().PutStr(conventions.AttributeServiceName, stage.Name)
		stageSpan.Attributes().PutInt("stage.number", int64(stage.Number))

		stageSpan.Status().SetCode(getOtelExitCode(stage.ExitCode))
		stageSpan.SetName(stage.Name)
		stageSpan.SetTraceID(traceId)
		stageSpan.SetSpanID(stageId)
		stageSpan.SetParentSpanID(buildId)
		stageSpan.Attributes().PutStr(CI_KIND, "stage")

		stageSpan.SetStartTimestamp(pcommon.Timestamp(stage.Started * 1000000000))
		stageSpan.SetEndTimestamp(pcommon.Timestamp(stage.Stopped * 1000000000))

		for _, step := range stage.Steps {
			stepSpanId := NewSpanID()
			stepSpan := stageSpans.Spans().AppendEmpty()
			stepSpan.SetTraceID(traceId)
			stepSpan.SetParentSpanID(stageId)
			stepSpan.SetSpanID(stepSpanId)
			stepSpan.Attributes().PutStr(CI_KIND, "step")
			stepSpan.Attributes().PutStr(CI_STAGE, stage.Name)

			stepSpan.Status().SetCode(getOtelExitCode(step.ExitCode))

			stepSpan.SetName(step.Name)

			stepSpan.SetStartTimestamp(pcommon.Timestamp(step.Started * 1000000000))
			stepSpan.SetEndTimestamp(pcommon.Timestamp(step.Stopped * 1000000000))

			lines, err := d.droneClient.Logs(repo.Namespace, repo.Name, int(build.Number), stage.Number, step.Number)
			if err != nil {
				d.logger.Error("error retrieving logs", zap.Error(err))
				continue
			}

			log := logs.ResourceLogs().AppendEmpty()
			logScope := log.ScopeLogs().AppendEmpty()

			now := pcommon.NewTimestampFromTime(time.Now())

			for _, line := range lines {
				record := logScope.LogRecords().AppendEmpty()
				record.SetTraceID(traceId)
				record.SetSpanID(stepSpanId)

				record.SetObservedTimestamp(now)
				record.SetTimestamp(pcommon.Timestamp((step.Started + line.Timestamp) * 1000000000))
				record.Attributes().PutStr(CI_STAGE, stage.Name)
				record.Attributes().PutStr(CI_STEP, step.Name)
				record.Body().SetStr(line.Message)
			}
		}
	}

	if d.nextTraceConsumer != nil {
		d.nextTraceConsumer.ConsumeTraces(req.Context(), traces)
	}
	if d.nextLogsConsumer != nil {
		d.nextLogsConsumer.ConsumeLogs(req.Context(), logs)
	}
}

func NewTraceID() pcommon.TraceID {
	return pcommon.TraceID(uuid.New())
}

func NewSpanID() pcommon.SpanID {
	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)
	randSource := rand.New(rand.NewSource(rngSeed))

	var sid [8]byte
	randSource.Read(sid[:])
	spanID := pcommon.SpanID(sid)

	return spanID
}
