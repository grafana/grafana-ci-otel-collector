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

var mCache = sync.Map{}

func newMetricsHandler(settings receiver.Settings, cfg *Config, logger *zap.Logger) *metricsHandler {
	return &metricsHandler{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
		logger:   logger,
	}
}

func (m *metricsHandler) eventToMetrics(event *github.WorkflowJobEvent) pmetric.Metrics {
	m.logger.Debug("conclusion", zap.String("conclusion", event.GetWorkflowJob().GetConclusion()))
	if event.GetWorkflowJob().GetConclusion() == "skipped" ||
		// Check runs are also reported via WorkflowJobEvent, we want to skip them when generating metrics.
		// see https://github.com/actions/actions-runner-controller/issues/2118
		(event.GetAction() == "completed" && event.GetWorkflowJob().GetRunnerID() == 0) {
		return m.mb.Emit()
	}

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

	now := pcommon.NewTimestampFromTime(time.Now())

	if status, ok := metadata.MapAttributeCiGithubWorkflowJobStatus[event.GetAction()]; ok {
		curVal, found := loadFromCache(repo, labels, status)

		// If the value was not found in the cache, we record a 0 value for all other possible statuses
		// so that counter resets are properly handled.
		if !found {
			for _, s := range metadata.MapAttributeCiGithubWorkflowJobStatus {
				if s == status {
					continue
				}

				storeInCache(repo, labels, s, 0)
				m.mb.RecordWorkflowJobsTotalDataPoint(now, 0, repo, labels, s)
			}
		}

		storeInCache(repo, labels, status, curVal+1)
		m.mb.RecordWorkflowJobsTotalDataPoint(now, curVal+1, repo, labels, status)
	}

	return m.mb.Emit()
}

func storeInCache(repo, labels string, status metadata.AttributeCiGithubWorkflowJobStatus, value int64) {
	middleMap, _ := mCache.LoadOrStore(repo, &sync.Map{})
	innerMap, _ := middleMap.(*sync.Map).LoadOrStore(labels, &sync.Map{})
	innerMap.(*sync.Map).Store(status, value)
}

// Helper function to load values from the nested sync.Map structure
func loadFromCache(repo, labels string, status metadata.AttributeCiGithubWorkflowJobStatus) (int64, bool) {
	middleMap, ok := mCache.Load(repo)
	if !ok {
		return 0, false
	}

	innerMap, ok := middleMap.(*sync.Map).Load(labels)
	if !ok {
		return 0, false
	}

	value, ok := innerMap.(*sync.Map).Load(status)
	if !ok {
		return 0, false
	}

	return value.(int64), true
}
