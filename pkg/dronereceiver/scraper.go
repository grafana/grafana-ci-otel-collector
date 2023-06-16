package dronereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/apachereceiver"

import (
	"context"
	"time"

	"github.com/grafana/grafana-collector/dronereceiver/internal/metadata"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

type droneScraper struct {
	settings component.TelemetrySettings
	db       *pgx.Conn
	mb       *metadata.MetricsBuilder
}

func newDroneScraper(settings receiver.CreateSettings, cfg *Config) *droneScraper {
	return &droneScraper{
		settings: settings.TelemetrySettings,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
	}
}

func (r *droneScraper) start(_ context.Context, host component.Host) error {
	r.settings.Logger.Info("Starting the drone scraper")
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:postgres@localhost:5432/drone?sslmode=disable")

	if err != nil {
		return err
	}

	r.db = conn

	return nil
}

func (r *droneScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	r.settings.Logger.Info("Scraping...")

	errs := &scrapererror.ScrapeErrors{}

	now := pcommon.NewTimestampFromTime(time.Now())

	r.scrapeBuilds(ctx, now, errs)

	return r.mb.Emit(), errs.Combine()
}

func (r *droneScraper) scrapeBuilds(ctx context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	var buildCount int
	builds := r.db.QueryRow(ctx, "SELECT count(*) FROM builds")
	builds.Scan(&buildCount)
	r.mb.RecordTotalBuildsDataPoint(now, int64(buildCount))

	builds = r.db.QueryRow(ctx, "SELECT count(*) FROM builds WHERE build_status = 'pending'")
	builds.Scan(&buildCount)
	r.mb.RecordPendingBuildsDataPoint(now, int64(buildCount))

	builds = r.db.QueryRow(ctx, "SELECT count(*) FROM builds WHERE build_status = 'running'")
	builds.Scan(&buildCount)
	r.mb.RecordRunningBuildsDataPoint(now, int64(buildCount))
}
