package dronereceiver

import (
	"encoding/binary"
	"math/rand"

	crand "crypto/rand"

	"github.com/drone/drone-go/drone"
	"github.com/google/uuid"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/collector/semconv/v1.9.0"
	"go.uber.org/zap"
)

type Stage struct {
	ID int
}

type Step struct {
	ID   int
	Name string
}

func generateTraces(droneClient drone.Client, logger *zap.Logger) ptrace.Traces {

	buildList, _ := droneClient.BuildList("grafana", "grafana", drone.ListOptions{
		Page: 3,
		Size: 5,
	})
	traces := ptrace.NewTraces()

	logger.Debug("Processing traces",
		zap.Int("builds", len(buildList)),
	)

	for _, b := range buildList {
		traceId := NewTraceID()
		buildId := NewSpanID()
		build, _ := droneClient.Build("grafana", "grafana", int(b.Number))

		if build.Finished == 0 {
			continue
		}

		logger.Debug("generating trace",
			zap.String("traceId", traceId.String()),
			zap.Int64("build.id", build.Number),
			zap.Int64("build.Created", build.Created*1000000000),
			zap.Int64("build.Finished", build.Finished*1000000000),
			zap.Int("build.Stages", len(build.Stages)),
		)

		resourceSpan := traces.ResourceSpans().AppendEmpty()
		scopeSpans := resourceSpan.ScopeSpans().AppendEmpty()

		scopeSpans.Scope().SetName("dronetracereceiver")
		scopeSpans.Scope().SetVersion("0.1.0")

		resourceAttrs := resourceSpan.Resource().Attributes()
		resourceAttrs.PutStr(conventions.AttributeServiceName, "build")
		resourceAttrs.PutStr(conventions.AttributeServiceVersion, "0.1.0")
		resourceAttrs.PutInt("build.number", build.Number)
		resourceAttrs.PutInt("build.id", build.ID)

		buildSpan := scopeSpans.Spans().AppendEmpty()
		buildSpan.SetTraceID(traceId)
		buildSpan.SetSpanID(buildId)
		buildSpan.SetParentSpanID(pcommon.NewSpanIDEmpty())

		buildSpan.SetName(build.Title)

		buildSpan.SetStartTimestamp(pcommon.Timestamp(build.Created * 1000000000))
		buildSpan.SetEndTimestamp(pcommon.Timestamp(build.Finished * 1000000000))

		for _, stage := range build.Stages {
			stageId := NewSpanID()
			stageSpans := resourceSpan.ScopeSpans().AppendEmpty()
			stageSpan := stageSpans.Spans().AppendEmpty()

			stageSpan.Attributes().PutStr(conventions.AttributeServiceName, stage.Name)
			stageSpan.Attributes().PutInt("stage.number", int64(stage.Number))

			stageSpan.SetName(stage.Name)
			stageSpan.SetTraceID(traceId)
			stageSpan.SetSpanID(stageId)
			stageSpan.SetParentSpanID(buildId)

			stageSpan.SetStartTimestamp(pcommon.Timestamp(stage.Started * 1000000000))
			stageSpan.SetEndTimestamp(pcommon.Timestamp(stage.Stopped * 1000000000))

			for _, step := range stage.Steps {
				stepSpan := stageSpans.Spans().AppendEmpty()
				stepSpan.SetTraceID(traceId)
				stepSpan.SetParentSpanID(stageId)
				stepSpan.SetSpanID(NewSpanID())

				stepSpan.SetName(step.Name)

				stepSpan.SetStartTimestamp(pcommon.Timestamp(step.Started * 1000000000))
				stepSpan.SetEndTimestamp(pcommon.Timestamp(step.Stopped * 1000000000))
			}
		}

	}

	return traces
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
