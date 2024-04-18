package dronereceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewTracesReceiver(t *testing.T) {
	t.Run("Missing consumer fails", func(t *testing.T) {
		rec, err := newTracesReceiver(context.Background(), receivertest.NewNopCreateSettings(), createDefaultConfig().(*Config), nil)

		require.ErrorIs(t, err, component.ErrNilNextConsumer)
		require.Nil(t, rec)
	})
}

func TestNewLogsReceiver(t *testing.T) {
	t.Run("Missing consumer fails", func(t *testing.T) {
		rec, err := newLogsReceiver(context.Background(), receivertest.NewNopCreateSettings(), createDefaultConfig().(*Config), nil)

		require.ErrorIs(t, err, component.ErrNilNextConsumer)
		require.Nil(t, rec)
	})
}

func TestCreateMetricsReceiver(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	_, err := NewFactory().CreateMetricsReceiver(
		context.Background(),
		receivertest.NewNopCreateSettings(),
		cfg,
		consumertest.NewNop(),
	)
	require.NoError(t, err)
}
