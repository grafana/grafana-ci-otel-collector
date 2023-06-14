package dronereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/apachereceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
)

type droneScraper struct {
	settings component.TelemetrySettings
}

func newDroneScraper(settings receiver.CreateSettings) *droneScraper {
	return &droneScraper{
		settings: settings.TelemetrySettings,
	}
}

func (r *droneScraper) start(_ context.Context, host component.Host) error {
	r.settings.Logger.Info("Starting the drone scraper")
	// TODO: maybe we need to do some setup here, i.e. connecting to the DB (or maybe we want to connect only when scraping)
	return nil
}

func (r *droneScraper) scrape(context.Context) (pmetric.Metrics, error) {
	r.settings.Logger.Info("Scraping...")
	metrics := pmetric.NewMetrics()
	// TODO: do queries and populate metrics
	// m := metrics.ResourceMetrics().AppendEmpty()
	// m.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()

	// now := pcommon.NewTimestampFromTime(time.Now())
	return metrics, nil
}
