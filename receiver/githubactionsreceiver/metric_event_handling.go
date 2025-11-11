package githubactionsreceiver

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v78/github"
	"github.com/grafana/grafana-ci-otel-collector/receiver/githubactionsreceiver/internal/metadata"
	"github.com/prometheus/common/version"
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
	settings.BuildInfo = component.BuildInfo{
		Command:     "githubactionsreceiver",
		Description: "GitHub Actions Receiver",
		Version:     version.Version,
	}
	mh := &metricsHandler{
		cfg:      cfg,
		settings: settings.TelemetrySettings,
		mb:       metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings),
		logger:   logger,
	}

	// Record build info metric
	ts := pcommon.NewTimestampFromTime(time.Now())
	mh.mb.RecordBuildInfoDataPoint(ts, 1, settings.BuildInfo.Version)

	return mh
}

func (m *metricsHandler) workflowJobEventToMetrics(event *github.WorkflowJobEvent) pmetric.Metrics {
	if event == nil || event.GetRepo() == nil || event.GetWorkflowJob() == nil {
		m.logger.Debug("Received nil event or missing required fields")
		return m.mb.Emit()
	}

	repo := event.GetRepo().GetFullName()
	if repo == "" {
		m.logger.Debug("Repository name is empty")
		return m.mb.Emit()
	}

	// Track what we've recorded in this emission to prevent duplicates
	// Create new map per emission to avoid race condition
	recorded := make(map[string]bool)

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
		curVal, found := loadFromCache(repo, labels, status, conclusion)

		metricKey := fmt.Sprintf("job:%s:%s:%s:%s:%t", repo, labels, status.String(), conclusion.String(), isMain)

		if !recorded[metricKey] {
			recorded[metricKey] = true
			if !found {
				for _, s := range metadata.MapAttributeCiGithubWorkflowJobStatus {
					for _, c := range metadata.MapAttributeCiGithubWorkflowJobConclusion {
						if s == status && c == conclusion {
							continue
						}
						storeInCache(repo, labels, s, c, 0)
						otherKey := fmt.Sprintf("job:%s:%s:%s:%s:%t", repo, labels, s.String(), c.String(), isMain)
						if !recorded[otherKey] {
							recorded[otherKey] = true
							m.mb.RecordWorkflowJobsCountDataPoint(now, 0, repo, labels, s, c, isMain)
						}
					}
				}
			}
			storeInCache(repo, labels, status, conclusion, curVal+1)
			m.mb.RecordWorkflowJobsCountDataPoint(now, curVal+1, repo, labels, status, conclusion, isMain)
		}
	}

	return m.mb.Emit()
}

func (m *metricsHandler) workflowRunEventToMetrics(event *github.WorkflowRunEvent) pmetric.Metrics {
	if event == nil || event.GetRepo() == nil || event.GetWorkflowRun() == nil {
		m.logger.Debug("Received nil event or missing required fields")
		return m.mb.Emit()
	}

	repo := event.GetRepo().GetFullName()
	if repo == "" {
		m.logger.Debug("Repository name is empty")
		return m.mb.Emit()
	}

	// Track what we've recorded in this emission to prevent duplicates
	// Create new map per emission to avoid race condition
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

	// Validate required fields before recording metrics
	if actionOk && repo != "" && status.String() != "" && conclusion.String() != "" {
		curVal, found := loadFromCache(repo, "default", status, conclusion)

		metricKey := fmt.Sprintf("run:%s:%s:%s:%s:%t", repo, "default", status.String(), conclusion.String(), isMain)

		if !recorded[metricKey] {
			recorded[metricKey] = true
			if !found {
				for _, s := range metadata.MapAttributeCiGithubWorkflowRunStatus {
					for _, c := range metadata.MapAttributeCiGithubWorkflowRunConclusion {
						if s == status && c == conclusion {
							continue
						}
						storeInCache(repo, "default", s, c, 0)
						otherKey := fmt.Sprintf("run:%s:%s:%s:%s:%t", repo, "default", s.String(), c.String(), isMain)
						if !recorded[otherKey] {
							recorded[otherKey] = true
							m.mb.RecordWorkflowRunsCountDataPoint(now, 0, repo, "default", s, c, isMain)
						}
					}
				}
			}
			storeInCache(repo, "default", status, conclusion, curVal+1)
			m.mb.RecordWorkflowRunsCountDataPoint(now, curVal+1, repo, "default", status, conclusion, isMain)
		}
	}

	return m.mb.Emit()
}

func storeInCache(repo, labels string, status interface{}, conclusion interface{}, value int64) {
	labelsMap, _ := repoMap.LoadOrStore(repo, &sync.Map{})
	statusesMap, _ := labelsMap.(*sync.Map).LoadOrStore(labels, &sync.Map{})
	conclusionsMap, _ := statusesMap.(*sync.Map).LoadOrStore(status, &sync.Map{})
	conclusionsMap.(*sync.Map).Store(conclusion, value)
}

// Helper function to load values from the nested sync.Map structure
func loadFromCache(repo, labels string, status interface{}, conclusion interface{}) (int64, bool) {
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
