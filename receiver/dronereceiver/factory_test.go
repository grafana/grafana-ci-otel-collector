package dronereceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestCreateMetricsReceiver(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	_, err := NewFactory().CreateMetrics(
		context.Background(),
		receivertest.NewNopSettings(),
		cfg,
		consumertest.NewNop(),
	)
	require.NoError(t, err)
}
