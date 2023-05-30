package dronereceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	typeStr = "dronetracereceiver"
)

func createDefaultConfig() component.Config {
	return &Config{}
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithTraces(createTraceReceiver, component.StabilityLevelAlpha),
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelAlpha),
		receiver.WithLogs(createLogsReceiver, component.StabilityLevelAlpha))
}

func createTraceReceiver(_ context.Context, set receiver.CreateSettings, cfg component.Config, consumer consumer.Traces) (receiver.Traces, error) {
	r, err := getOrAddReceiver(set, cfg)
	if err != nil {
		return nil, err
	}

	if err = r.Unwrap().registerTraceConsumer(consumer); err != nil {
		return nil, err
	}
	return r, nil

}

func createMetricsReceiver(_ context.Context, set receiver.CreateSettings, cfg component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	r, err := getOrAddReceiver(set, cfg)
	if err != nil {
		return nil, err
	}

	if err = r.Unwrap().registerMetricsConsumer(consumer); err != nil {
		return nil, err
	}
	return r, nil

}

func createLogsReceiver(_ context.Context, set receiver.CreateSettings, cfg component.Config, consumer consumer.Logs) (receiver.Logs, error) {
	r, err := getOrAddReceiver(set, cfg)
	if err != nil {
		return nil, err
	}

	if err = r.Unwrap().registerLogsConsumer(consumer); err != nil {
		return nil, err
	}
	return r, nil

}

func getOrAddReceiver(set receiver.CreateSettings, cfg component.Config) (*SharedComponent[*droneReceiver], error) {
	oCfg := cfg.(*Config)
	r, err := receivers.GetOrAdd(set.ID, func() (*droneReceiver, error) {
		return newDroneReceiver(oCfg, set)
	})
	if err != nil {
		return nil, err
	}

	return r, nil
}

// the receiver is able to handle all types of data, we only create one instance per ID
var receivers = NewSharedComponents[component.ID, *droneReceiver]()
