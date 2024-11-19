package githubactionsreceiver

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v62/github"
	"github.com/grafana/grafana-ci-otel-collector/receiver/githubactionsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type metricsHandler struct {
	settings component.TelemetrySettings
	mb       *metadata.MetricsBuilder
	cfg      *Config
	logger   *zap.Logger
}

var repoMap = sync.Map{}

func newMetricsHandler(settings receiver.Settings, cfg *Config, logger *zap.Logger) *metricsHandler {
	return &metricsHandler{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
		logger:   logger,
	}
}

func (m *metricsHandler) eventToMetrics(event *github.WorkflowJobEvent) pmetric.Metrics {
	repo := event.GetRepo().GetFullName()

	labels := ""
	if len(event.GetWorkflowJob().Labels) > 0 {
		labelsSlice := event.GetWorkflowJob().Labels
		for i, label := range labelsSlice {
			labelsSlice[i] = strings.ToLower(label)
		}
		sort.Strings(labelsSlice)
		labels = strings.Join(labelsSlice, ",")
	} else {
		labels = "no labels"
	}

	m.logger.Info("Processing workflow_job event",
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

	if actionOk {
		curVal, found := loadFromCache(repo, labels, status, conclusion)

		// If the value was not found in the cache, we record a 0 value for all other possible statuses
		// so that all counters for a given labels combination are always present and reset at the same time.
		if !found {
			for _, s := range metadata.MapAttributeCiGithubWorkflowJobStatus {
				for _, c := range metadata.MapAttributeCiGithubWorkflowJobConclusion {
					if s == status && c == conclusion {
						continue
					}

					storeInCache(repo, labels, s, c, 0)
					m.mb.RecordWorkflowJobsCountDataPoint(now, 0, repo, labels, s, c)
				}

			}
		}

		storeInCache(repo, labels, status, conclusion, curVal+1)
		m.mb.RecordWorkflowJobsCountDataPoint(now, curVal+1, repo, labels, status, conclusion)
	}

	return m.mb.Emit()
}

func storeInCache(repo, labels string, status metadata.AttributeCiGithubWorkflowJobStatus, conclusion metadata.AttributeCiGithubWorkflowJobConclusion, value int64) {
	labelsMap, _ := repoMap.LoadOrStore(repo, &sync.Map{})
	statusesMap, _ := labelsMap.(*sync.Map).LoadOrStore(labels, &sync.Map{})
	conclusionsMap, _ := statusesMap.(*sync.Map).LoadOrStore(status, &sync.Map{})
	conclusionsMap.(*sync.Map).Store(conclusion, value)
}

// Helper function to load values from the nested sync.Map structure
func loadFromCache(repo, labels string, status metadata.AttributeCiGithubWorkflowJobStatus, conclusion metadata.AttributeCiGithubWorkflowJobConclusion) (int64, bool) {
	labelsMap, ok := repoMap.Load(repo)
	if !ok {
		return 0, false
	}

	statusesMap, ok := labelsMap.(*sync.Map).Load(labels)
	if !ok {
		return 0, false
	}

	conclusionsMap, ok := statusesMap.(*sync.Map).Load(status)
	if !ok {
		return 0, false
	}

	value, ok := conclusionsMap.(*sync.Map).Load(conclusion)
	if !ok {
		return 0, false
	}

	return value.(int64), true
}
