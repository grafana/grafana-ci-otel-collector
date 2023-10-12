package dronereceiver

import (
	"context"
	"time"

	"github.com/grafana/grafana-ci-otel-collector/dronereceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultScraperControllerSettings(metadata.Type)
	cfg.CollectionInterval = 10 * time.Second

	return &Config{
		ScraperControllerSettings: cfg,
		MetricsBuilderConfig:      metadata.DefaultMetricsBuilderConfig(),
		WebhookConfig: WebhookConfig{
			Endpoint: "/drone/webhook",
			Port:     3333,
		},
	}
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(createTraceReceiver, metadata.TracesStability),
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability))
}

func createTraceReceiver(_ context.Context, set receiver.CreateSettings, cfg component.Config, consumer consumer.Traces) (receiver.Traces, error) {
	r, err := getOrAddReceiver(set, cfg)
	if err != nil {
		return nil, err
	}

	r.Unwrap().enableTraces(consumer)
	return r, nil

}

func createMetricsReceiver(_ context.Context, set receiver.CreateSettings, rConf component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	cfg := rConf.(*Config)
	ns := newDroneScraper(set, cfg)
	scraper, err := scraperhelper.NewScraper(metadata.Type, ns.scrape, scraperhelper.WithStart(ns.start))

	if err != nil {
		return nil, err
	}

	return scraperhelper.NewScraperControllerReceiver(
		&cfg.ScraperControllerSettings, set, consumer,
		scraperhelper.AddScraper(scraper),
	)
}

func createLogsReceiver(_ context.Context, set receiver.CreateSettings, cfg component.Config, consumer consumer.Logs) (receiver.Logs, error) {
	r, err := getOrAddReceiver(set, cfg)
	if err != nil {
		return nil, err
	}

	r.Unwrap().enableLogs(consumer)
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
