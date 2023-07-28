package dronereceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/apachereceiver"

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"time"

	"github.com/grafana/grafana-collector/dronereceiver/internal/metadata"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

const (
	localhost = "localhost"
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
	networkHost := localhost
	if networkHostEnv, ok := os.LookupEnv("NETWORK_HOST"); ok {
		networkHost = networkHostEnv
	} else {
		r.settings.Logger.Info("NETWORK_HOST env var is missing, defaulting to localhost")
	}
	err := godotenv.Load()
	if err != nil {
		r.settings.Logger.Warn("Error loading .env file, variables will be taken from the host environment")
	}
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB")

	r.settings.Logger.Info("Starting the drone scraper")
	conn, err := pgx.Connect(context.Background(), fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", postgresUser, postgresPassword, networkHost, postgresDB))

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
	r.scrapeRestartedBuilds(ctx, now, errs)

	return r.mb.Emit(), errs.Combine()
}

// repo_slug, build_source, build_status
type Builds map[string]map[string]map[metadata.AttributeBuildStatus]int64

func (r *droneScraper) scrapeBuilds(ctx context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	var buildCount int64
	rows, err := r.db.Query(ctx, `
		SELECT 
			count(*),
			build_status,
			r.repo_slug,
			build_source 
		FROM 
			builds 
		LEFT JOIN
			repos r 
		ON 
			build_repo_id = r.repo_id  
		GROUP BY 
			build_status,
			r.repo_slug,
			build_source
	`)

	if err != nil {
		errs.Add(err)
	}

	values := make(Builds)
	for rows.Next() {
		var status string
		var slug string
		var source string
		err := rows.Scan(&buildCount, &status, &slug, &source)
		if err != nil {
			errs.Add(err)
			continue
		}

		if _, ok := values[slug]; !ok {
			values[slug] = make(map[string]map[metadata.AttributeBuildStatus]int64)
		}

		if _, ok := values[slug][source]; !ok {
			values[slug][source] = make(map[metadata.AttributeBuildStatus]int64)
		}

		if key, ok := metadata.MapAttributeBuildStatus[status]; ok {
			values[slug][source][key] = buildCount
		} else {
			values[slug][source][key] += buildCount
		}
	}

	for slug, repo := range values {
		for branch, source := range repo {
			for _, statusAttr := range metadata.MapAttributeBuildStatus {
				if val, ok := source[statusAttr]; ok {
					r.mb.RecordBuildsNumberDataPoint(now, val, statusAttr, slug, branch)
				} else {
					r.mb.RecordBuildsNumberDataPoint(now, 0, statusAttr, slug, branch)
				}
			}
		}
	}

}

func (r *droneScraper) scrapeRestartedBuilds(ctx context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	var count int64
	builds := r.db.QueryRow(ctx, "SELECT COALESCE(SUM(occurrence_count - 1), 0) AS total_occurrence_count FROM ( SELECT count(*) AS occurrence_count FROM builds GROUP BY build_after, build_source HAVING COUNT(*) > 1) subquery")
	err := builds.Scan(&count)
	if err != nil {
		errs.Add(err)
	}
	r.mb.RecordRestartsTotalDataPoint(now, count)
}
