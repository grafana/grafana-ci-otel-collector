package githubactionsreceiver

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/go-github/v85/github"
	"github.com/grafana/grafana-ci-otel-collector/receiver/githubactionsreceiver/internal/metadata"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/prometheus/common/version"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type metricsHandler struct {
	mu             sync.Mutex
	settings       component.TelemetrySettings
	mb             *metadata.MetricsBuilder
	cfg            *Config
	logger         *zap.Logger
	countersCache  *lru.Cache[string, int64]
	histogramCache *lru.Cache[string, *histogramState]
}

const metricsMaxCacheSize = 100000
const histogramCacheSize = 50000
const histogramTTL = 24 * time.Hour

func cacheKey(repo, labels string, status, conclusion interface{}, isMain bool) string {
	return fmt.Sprintf("%s:%s:%v:%v:%t", repo, labels, status, conclusion, isMain)
}

func newMetricsHandler(settings receiver.Settings, cfg *Config, logger *zap.Logger) *metricsHandler {
	settings.BuildInfo = component.BuildInfo{
		Command:     "githubactionsreceiver",
		Description: "GitHub Actions Receiver",
		Version:     version.Version,
	}

	countersCache, err := lru.New[string, int64](metricsMaxCacheSize)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize counters cache: %v", err))
	}

	// histogramCache stores cumulative histogram state per unique dimension set.
	// We emit histograms with cumulative temporality (required by Prometheus-compatible
	// backends), so each emission must include running totals of count/sum/buckets
	// across all observations — not just the latest event.
	histCache, err2 := lru.New[string, *histogramState](histogramCacheSize)
	if err2 != nil {
		panic(fmt.Sprintf("Failed to initialize histogram cache: %v", err2))
	}

	mh := &metricsHandler{
		cfg:            cfg,
		settings:       settings.TelemetrySettings,
		mb:             metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
		logger:         logger,
		countersCache:  countersCache,
		histogramCache: histCache,
	}

	return mh
}

func (m *metricsHandler) buildInfoMetrics() pmetric.Metrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	ts := pcommon.NewTimestampFromTime(time.Now())
	m.mb.RecordBuildInfoDataPoint(ts, 1, version.Version)
	return m.mb.Emit()
}

func (m *metricsHandler) workflowJobEventToMetrics(event *github.WorkflowJobEvent) pmetric.Metrics {
	if event == nil || event.GetRepo() == nil || event.GetWorkflowJob() == nil {
		m.logger.Debug("Received nil event or missing required fields")
		m.mu.Lock()
		defer m.mu.Unlock()
		return m.mb.Emit()
	}

	repo := event.GetRepo().GetFullName()
	if repo == "" {
		m.logger.Debug("Repository name is empty")
		m.mu.Lock()
		defer m.mu.Unlock()
		return m.mb.Emit()
	}

	// Track what we've recorded in this emission to prevent duplicates
	// Create new map per emission to avoid race condition
	recorded := make(map[string]bool)

	labels := sortedLabels(event.GetWorkflowJob().Labels)

	// Acquire the mutex only for the remainder of the function, which
	// interacts with shared state such as m.mb and m.countersCache.
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debug("Processing workflow_job event",
		zap.String("repo", repo),
		zap.Int64("run_id", event.GetWorkflowJob().GetRunID()),
		zap.Int64("id", event.GetWorkflowJob().GetID()),
		zap.String("name", event.GetWorkflowJob().GetName()),
		zap.String("workflow_name", event.GetWorkflowJob().GetWorkflowName()),
		zap.String("action", event.GetAction()),
		zap.String("status", event.GetWorkflowJob().GetStatus()),
		zap.String("conclusion", event.GetWorkflowJob().GetConclusion()),
		zap.String("labels", labels),
		zap.Any("steps", event.GetWorkflowJob().Steps),
		zap.Int64("runner", event.GetWorkflowJob().GetRunnerID()),
	)

	now := pcommon.NewTimestampFromTime(time.Now())

	status, actionOk := metadata.MapAttributeCiGithubWorkflowJobStatus[event.GetAction()]
	conclusion, conclusionOk := metadata.MapAttributeCiGithubWorkflowJobConclusion[event.GetWorkflowJob().GetConclusion()]
	if status == metadata.AttributeCiGithubWorkflowJobStatusCompleted && conclusion == metadata.AttributeCiGithubWorkflowJobConclusionCancelled && len(event.GetWorkflowJob().Steps) == 1 {
		status = metadata.AttributeCiGithubWorkflowJobStatusAborted
	}
	if !conclusionOk {
		conclusion = metadata.AttributeCiGithubWorkflowJobConclusionNull
	}

	defaultBranch := event.GetRepo().DefaultBranch
	var isMain bool

	if defaultBranch != nil && event.GetWorkflowJob().GetHeadBranch() == *defaultBranch {
		isMain = true
	}

	// Validate required fields before recording metrics
	if actionOk && repo != "" && status.String() != "" && conclusion.String() != "" {
		curVal, found := m.loadFromCache(repo, labels, status, conclusion, isMain)

		metricKey := fmt.Sprintf("job:%s:%s:%s:%s:%t", repo, labels, status.String(), conclusion.String(), isMain)

		if !recorded[metricKey] {
			recorded[metricKey] = true
			if !found {
				for _, s := range metadata.MapAttributeCiGithubWorkflowJobStatus {
					for _, c := range metadata.MapAttributeCiGithubWorkflowJobConclusion {
						if s == status && c == conclusion {
							continue
						}
						m.storeInCache(repo, labels, s, c, isMain, 0)
						otherKey := fmt.Sprintf("job:%s:%s:%s:%s:%t", repo, labels, s.String(), c.String(), isMain)
						if !recorded[otherKey] {
							recorded[otherKey] = true
							m.mb.RecordWorkflowJobsCountDataPoint(now, 0, repo, labels, s, c, isMain)
						}
					}
				}
			}
			m.storeInCache(repo, labels, status, conclusion, isMain, curVal+1)
			m.mb.RecordWorkflowJobsCountDataPoint(now, curVal+1, repo, labels, status, conclusion, isMain)
		}
	}

	metrics := m.mb.Emit()
	ms := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	m.appendJobDurationMetric(ms, event)
	m.sweepStaleHistograms()
	return metrics
}

func (m *metricsHandler) workflowRunEventToMetrics(event *github.WorkflowRunEvent) pmetric.Metrics {
	// Validate event and required fields before acquiring the lock.
	if event == nil || event.GetRepo() == nil || event.GetWorkflowRun() == nil {
		m.logger.Debug("Received nil event or missing required fields")
		m.mu.Lock()
		metrics := m.mb.Emit()
		m.mu.Unlock()
		return metrics
	}

	repo := event.GetRepo().GetFullName()
	if repo == "" {
		m.logger.Debug("Repository name is empty")
		m.mu.Lock()
		metrics := m.mb.Emit()
		m.mu.Unlock()
		return metrics
	}

	// Track what we've recorded in this emission to prevent duplicates.
	// Create new map per emission to avoid race conditions.
	recorded := make(map[string]bool)

	m.logger.Debug("Processing workflow_run event",
		zap.String("repo", repo),
		zap.Int64("id", event.GetWorkflowRun().GetID()),
		zap.String("name", event.GetWorkflowRun().GetName()),
		zap.String("action", event.GetAction()),
		zap.String("status", event.GetWorkflowRun().GetStatus()),
		zap.String("conclusion", event.GetWorkflowRun().GetConclusion()),
	)

	now := pcommon.NewTimestampFromTime(time.Now())

	status, actionOk := metadata.MapAttributeCiGithubWorkflowRunStatus[event.GetAction()]
	conclusion, conclusionOk := metadata.MapAttributeCiGithubWorkflowRunConclusion[event.GetWorkflowRun().GetConclusion()]
	if !conclusionOk {
		conclusion = metadata.AttributeCiGithubWorkflowRunConclusionNull
	}

	defaultBranch := event.GetRepo().DefaultBranch
	var isMain bool

	if defaultBranch != nil && event.GetWorkflowRun().GetHeadBranch() == *defaultBranch {
		isMain = true
	}

	// Acquire the lock only around cache mutations, MetricsBuilder updates, and Emit().
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate required fields before recording metrics
	if actionOk && repo != "" && status.String() != "" && conclusion.String() != "" {
		curVal, found := m.loadFromCache(repo, "default", status, conclusion, isMain)

		metricKey := fmt.Sprintf("run:%s:%s:%s:%s:%t", repo, "default", status.String(), conclusion.String(), isMain)

		if !recorded[metricKey] {
			recorded[metricKey] = true
			if !found {
				for _, s := range metadata.MapAttributeCiGithubWorkflowRunStatus {
					for _, c := range metadata.MapAttributeCiGithubWorkflowRunConclusion {
						if s == status && c == conclusion {
							continue
						}
						m.storeInCache(repo, "default", s, c, isMain, 0)
						otherKey := fmt.Sprintf("run:%s:%s:%s:%s:%t", repo, "default", s.String(), c.String(), isMain)
						if !recorded[otherKey] {
							recorded[otherKey] = true
							m.mb.RecordWorkflowRunsCountDataPoint(now, 0, repo, "default", s, c, isMain)
						}
					}
				}
			}
			m.storeInCache(repo, "default", status, conclusion, isMain, curVal+1)
			m.mb.RecordWorkflowRunsCountDataPoint(now, curVal+1, repo, "default", status, conclusion, isMain)
		}
	}

	metrics := m.mb.Emit()
	ms := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	m.appendRunDurationMetric(ms, event)
	m.sweepStaleHistograms()
	return metrics
}

func (m *metricsHandler) storeInCache(repo, labels string, status interface{}, conclusion interface{}, isMain bool, value int64) {
	key := cacheKey(repo, labels, status, conclusion, isMain)
	m.countersCache.Add(key, value)
}

func (m *metricsHandler) loadFromCache(repo, labels string, status interface{}, conclusion interface{}, isMain bool) (int64, bool) {
	key := cacheKey(repo, labels, status, conclusion, isMain)
	return m.countersCache.Get(key)
}

// sweepStaleHistograms removes histogram cache entries that haven't been
// updated within histogramTTL. Called under m.mu.
func (m *metricsHandler) sweepStaleHistograms() {
	now := time.Now()
	for _, key := range m.histogramCache.Keys() {
		state, ok := m.histogramCache.Peek(key)
		if ok && now.Sub(state.lastSeen) >= histogramTTL {
			m.histogramCache.Remove(key)
		}
	}
}
