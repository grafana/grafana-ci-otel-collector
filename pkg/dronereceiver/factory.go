package dronereceiver

import (
	"context"

	"github.com/grafana/grafana-ci-otel-collector/dronereceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

const (
	defaultBindEndpoint = "0.0.0.0:3333"
	defaultPath         = "/drone/webhook"
)

func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultScraperControllerSettings(metadata.Type)

	return &Config{
		ScraperControllerSettings: cfg,
		MetricsBuilderConfig:      metadata.DefaultMetricsBuilderConfig(),
		ServerConfig: confighttp.ServerConfig{
			Endpoint: defaultBindEndpoint,
		},
		Path:   defaultPath,
		Secret: "",
	}
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(newTracesReceiver, metadata.TracesStability),
		receiver.WithMetrics(newMetricsReceiver, metadata.MetricsStability),
		receiver.WithLogs(newLogsReceiver, metadata.LogsStability))
}

func newTracesReceiver(_ context.Context, set receiver.CreateSettings, cfg component.Config, consumer consumer.Traces) (receiver.Traces, error) {
	rCfg := cfg.(*Config)
	var err error

	if consumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	r := receivers.GetOrAdd(cfg, func() component.Component {
		var rcv component.Component
		rcv, err = newReceiver(set, rCfg)
		return rcv
	})
	if err != nil {
		return nil, err
	}

	r.Unwrap().(*droneReceiver).tracesConsumer = consumer
	return r, nil
}

func newMetricsReceiver(_ context.Context, set receiver.CreateSettings, rConf component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	cfg := rConf.(*Config)

	if consumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	ns := newDroneScraper(set, cfg)
	scraper, err := scraperhelper.NewScraper(string(metadata.Type), ns.scrape, scraperhelper.WithStart(ns.start))

	if err != nil {
		return nil, err
	}

	return scraperhelper.NewScraperControllerReceiver(
		&cfg.ScraperControllerSettings, set, consumer,
		scraperhelper.AddScraper(scraper),
	)
}

func newLogsReceiver(_ context.Context, set receiver.CreateSettings, cfg component.Config, consumer consumer.Logs) (receiver.Logs, error) {
	rCfg := cfg.(*Config)
	var err error

	if consumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	r := receivers.GetOrAdd(cfg, func() component.Component {
		var rcv component.Component
		rcv, err = newReceiver(set, rCfg)
		return rcv
	})
	if err != nil {
		return nil, err
	}

	r.Unwrap().(*droneReceiver).logsConsumer = consumer
	return r, nil
}

// the receiver is able to handle all types of data, we only create one instance per ID
var receivers = NewSharedComponents()
