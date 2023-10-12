package traceutils

import (
	crand "crypto/rand"

	"encoding/binary"
	"math/rand"

	"github.com/google/uuid"
	"github.com/grafana/grafana-ci-otel-collector/semconv"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/collector/semconv/v1.18.0"
)

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

func getOtelExitCode(code string) ptrace.StatusCode {
	switch code {
	case "failure":
		fallthrough
	case "error":
		return ptrace.StatusCodeError
	case "success":
		return ptrace.StatusCodeOk
	default:
		return ptrace.StatusCodeUnset
	}
}

func SetStatus(status string, span ptrace.Span) {
	span.Attributes().PutStr(semconv.AttributeCIStatus, status)
	span.Status().SetCode(getOtelExitCode(status))
}

func CreateSpan(parent ptrace.ScopeSpans, p2 ptrace.Span) ptrace.Span {
	child := parent.Spans().AppendEmpty()

	return child

}

func CreateRootSpan(scope ptrace.ScopeSpans) ptrace.Span {
	span := scope.Spans().AppendEmpty()

	span.SetTraceID(NewTraceID())
	span.SetSpanID(NewSpanID())
	span.SetParentSpanID(pcommon.NewSpanIDEmpty())

	return span
}

type trace struct {
	traces   ptrace.Traces
	NewChild func() ptrace.Span
}

func NewTrace() trace {

	traces := ptrace.NewTraces()
	b := trace{
		traces: traces,
		NewChild: func() ptrace.Span {
			a := traces.ResourceSpans().AppendEmpty()
			b := a.ScopeSpans().AppendEmpty()

			return b.Spans().AppendEmpty()
		},
	}

	b.traces.ResourceSpans().AppendEmpty()

	resourceSpans := b.traces.ResourceSpans().AppendEmpty()

	resourceAttrs := resourceSpans.Resource().Attributes()
	resourceAttrs.PutStr(conventions.AttributeServiceVersion, "0.1.0")
	resourceAttrs.PutStr(conventions.AttributeServiceName, "drone")

	return b
}
