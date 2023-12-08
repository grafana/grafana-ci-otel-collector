package traceutils

import (
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"testing"
)

func Test_getOtelExitCode(t *testing.T) {
	tests := []struct {
		name string
		code string
		want ptrace.StatusCode
	}{
		{name: "failure - expect error", code: "failure", want: ptrace.StatusCodeError},
		{name: "error - expect error", code: "error", want: ptrace.StatusCodeError},
		{name: "success - expect OK", code: "success", want: ptrace.StatusCodeOk},
		{name: "unknown - expect unset", code: "unknown", want: ptrace.StatusCodeUnset},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getOtelExitCode(tt.code); got != tt.want {
				t.Errorf("getOtelExitCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSpanID(t *testing.T) {
	spanId := NewSpanID()
	require.NotEmpty(t, spanId.String())
	require.Len(t, spanId.String(), 16)
}

func TestNewTraceID(t *testing.T) {
	traceId := NewTraceID()
	require.NotEmpty(t, traceId.String())
	require.Len(t, traceId.String(), 32)
}
