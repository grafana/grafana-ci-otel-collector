package traceutils

import (
	crand "crypto/rand"

	"encoding/binary"
	"math/rand"

	"github.com/google/uuid"
	"github.com/grafana/grafana-ci-otel-collector/semconv"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
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
	span.Attributes().PutStr(semconv.AttributeCIWorkflowItemStatus, status)
	span.Status().SetCode(getOtelExitCode(status))
}
