package dronereceiver

import (
	"context"
	"time"

	"github.com/drone/drone-go/drone"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

type dronereceiverReceiver struct {
	host         component.Host
	cancel       context.CancelFunc
	logger       *zap.Logger
	nextConsumer consumer.Traces
	config       *Config
	drone        drone.Client
}

func (dronereceiverRcvr *dronereceiverReceiver) Start(ctx context.Context, host component.Host) error {
	dronereceiverRcvr.host = host
	ctx = context.Background()
	ctx, dronereceiverRcvr.cancel = context.WithCancel(ctx)

	// interval, _ := time.ParseDuration(dronereceiverRcvr.config.Interval)
	go func() {
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				dronereceiverRcvr.logger.Info("I should start processing traces now!")
				dronereceiverRcvr.nextConsumer.ConsumeTraces(ctx, generateTraces(dronereceiverRcvr.drone, dronereceiverRcvr.logger))
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (dronereceiverRcvr *dronereceiverReceiver) Shutdown(ctx context.Context) error {
	dronereceiverRcvr.cancel()
	return nil
}
