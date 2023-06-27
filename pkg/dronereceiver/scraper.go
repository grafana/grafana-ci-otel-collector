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
	"go.uber.org/zap"
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
	var buildCount int64
	rows, err := r.db.Query(ctx, "SELECT count(*), build_status FROM builds GROUP BY build_status")

	if err != nil {
		r.settings.Logger.Error("Query error", zap.Error(err))
	}

	for rows.Next() {
		var status string
		err := rows.Scan(&buildCount, &status)
		if err != nil {
			r.settings.Logger.Error("Error scanning row", zap.Error(err))
			continue
		}

		if statusAttr, ok := metadata.MapAttributeBuildStatus[status]; ok {
			r.mb.RecordBuildsTotalDataPoint(now, buildCount, statusAttr)
		} else {
			r.mb.RecordBuildsTotalDataPoint(now, buildCount, statusAttr)
		}
	}

	builds := r.db.QueryRow(ctx, "SELECT SUM(occurrence_count - 1) AS total_occurrence_count FROM ( SELECT count(*) AS occurrence_count FROM builds GROUP BY build_after, build_source HAVING COUNT(*) > 1) subquery")
	builds.Scan(&buildCount)
	r.mb.RecordRestartsTotalDataPoint(now, buildCount)
}
