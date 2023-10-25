package dronereceiver

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/grafana/grafana-ci-otel-collector/dronereceiver/internal/metadata"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scrapererror"
)

const (
	timeout = 120
)

type droneScraper struct {
	settings component.TelemetrySettings
	dbPool   *pgxpool.Pool
	mb       *metadata.MetricsBuilder
	cfg      *Config
}

func newDroneScraper(settings receiver.CreateSettings, cfg *Config) *droneScraper {
	return &droneScraper{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
	}
}

func (r *droneScraper) start(_ context.Context, host component.Host) error {
	r.settings.Logger.Info("Starting the drone scraper")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeoutExceeded := time.After(timeout * time.Second)
	success := false
	for !success {
		select {
		case <-timeoutExceeded:
			r.settings.Logger.Error("db connection failed after %d second(s) timeout", zap.Int64("timeout", timeout))
			os.Exit(1)

		case <-ticker.C:
			connString := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", r.cfg.DroneConfig.Database.Username, r.cfg.DroneConfig.Database.Password, r.cfg.DroneConfig.Database.Host, r.cfg.DroneConfig.Database.DB)
			err := r.dbConnect(connString)
			if err == nil {
				success = true
				break
			}
			r.settings.Logger.Error("failed attempt to connect to db %w", zap.Error(err))
		}
	}

	return nil
}

func (r *droneScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	r.settings.Logger.Debug("Scraping...")

	errs := &scrapererror.ScrapeErrors{}

	now := pcommon.NewTimestampFromTime(time.Now())

	r.scrapeBuilds(ctx, now, errs)
	r.scrapeRestartedBuilds(ctx, now, errs)
	r.scrapeInfo(ctx, now, errs)

	return r.mb.Emit(), errs.Combine()
}

// repo_slug, build_source, build_status
type Builds map[string]map[string]map[metadata.AttributeCiWorkflowItemStatus]int64

func (r *droneScraper) scrapeBuilds(ctx context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {
	var buildCount int64

	conditions := make([]string, 0)
	repoSlugs := make([]string, 0)
	for repoSlug, buildSources := range r.cfg.ReposConfig {
		repoSlugs = append(repoSlugs, repoSlug)
		conditions = append(conditions, fmt.Sprintf("WHEN repo_slug = '%s' AND build_source IN ('%s') THEN build_source", repoSlug, fmt.Sprintf(strings.Join(buildSources, "', '"))))
	}

	rows, err := r.dbPool.Query(ctx, fmt.Sprintf(`
		SELECT 
			count(*),
			build_status,
			CASE 
				WHEN repo_slug IN ('%s') THEN repo_slug
				ELSE 'other'
			END AS slug,
			CASE 
				%s
				ELSE 'other'
			END AS source
		FROM 
			builds 
		LEFT JOIN
			repos r
		ON 
			build_repo_id = r.repo_id  
		GROUP BY 
			build_status,
			slug,
			source
	`, strings.Join(repoSlugs, "', '"), strings.Join(conditions, " ")))

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
			values[slug] = make(map[string]map[metadata.AttributeCiWorkflowItemStatus]int64)
		}

		if _, ok := values[slug][source]; !ok {
			values[slug][source] = make(map[metadata.AttributeCiWorkflowItemStatus]int64)
		}

		if key, ok := metadata.MapAttributeCiWorkflowItemStatus[status]; ok {
			values[slug][source][key] = buildCount
		} else {
			values[slug][source][key] += buildCount
		}
	}

	for slug, repo := range values {
		for branch, source := range repo {
			for _, statusAttr := range metadata.MapAttributeCiWorkflowItemStatus {
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
	builds := r.dbPool.QueryRow(ctx, "SELECT COALESCE(SUM(occurrence_count - 1), 0) AS total_occurrence_count FROM ( SELECT count(*) AS occurrence_count FROM builds GROUP BY build_after, build_source HAVING COUNT(*) > 1) subquery")
	err := builds.Scan(&count)
	if err != nil {
		errs.Add(err)
	}
	r.mb.RecordRestartsTotalDataPoint(now, count)
}

func (r *droneScraper) scrapeInfo(ctx context.Context, now pcommon.Timestamp, errs *scrapererror.ScrapeErrors) {

	conditions := make([]string, 0)
	for repoSlug, buildSources := range r.cfg.ReposConfig {
		conditions = append(conditions, fmt.Sprintf("repo_slug = '%s' AND build_source IN ('%s')", repoSlug, fmt.Sprintf(strings.Join(buildSources, "', '"))))
	}

	rows, err := r.dbPool.Query(ctx, fmt.Sprintf(`
		SELECT build_status, r.repo_slug, build_source FROM builds
		LEFT JOIN
			repos r
		ON 
			build_repo_id = r.repo_id 
		WHERE build_id IN (
			SELECT MAX(build_id) 
			FROM 
				builds 
			JOIN 
				repos r 
			ON 
				build_repo_id = r.repo_id
			WHERE 
				build_status NOT IN ('running','waiting_on_dependencies','pending') 
				AND (%s)
			GROUP BY build_repo_id, build_source
		)
	`, strings.Join(conditions, " OR ")))

	if err != nil {
		errs.Add(err)
	}

	for rows.Next() {
		var status string
		var slug string
		var source string
		err := rows.Scan(&status, &slug, &source)
		if err != nil {
			errs.Add(err)
			continue
		}

		r.mb.RecordRepoInfoDataPoint(now, 1, metadata.MapAttributeCiWorkflowItemStatus[status], slug, source)
	}
}

func (r *droneScraper) dbConnect(connString string) error {
	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return err
	}
	r.settings.Logger.Info("successfully connected to db!")
	r.dbPool = pool
	return nil
}
