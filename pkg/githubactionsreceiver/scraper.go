package githubactionsreceiver

import (
	"context"

	"github.com/grafana/grafana-ci-otel-collector/githubactionsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

const (
	timeout = 120
)

type githubactionsScraper struct {
	settings component.TelemetrySettings
	mb       *metadata.MetricsBuilder
	cfg      *Config
}

func newDroneScraper(settings receiver.CreateSettings, cfg *Config) *githubactionsScraper {
	return &githubactionsScraper{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
	}
}

func (r *githubactionsScraper) start(_ context.Context, host component.Host) error {
	r.settings.Logger.Info("Starting the drone scraper")

	return nil
}

func (r *githubactionsScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	r.settings.Logger.Debug("Scraping...")

	errs := &scrapererror.ScrapeErrors{}

	return r.mb.Emit(), errs.Combine()
}
