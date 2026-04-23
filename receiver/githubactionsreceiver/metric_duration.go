package githubactionsreceiver

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v85/github"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var durationBucketBounds = []float64{5, 15, 30, 60, 300, 600, 1800}

func sortedLabels(labels []string) string {
	if len(labels) == 0 {
		return "no labels"
	}
	s := make([]string, len(labels))
	for i, l := range labels {
		s[i] = strings.ToLower(l)
	}
	sort.Strings(s)
	return strings.Join(s, ",")
}

type histogramState struct {
	count        uint64
	sum          float64
	bucketCounts []uint64
	lastSeen     time.Time
}

func (h *histogramState) observe(duration float64, bounds []float64) {
	h.count++
	h.sum += duration
	h.lastSeen = time.Now()
	for i, b := range bounds {
		if duration <= b {
			h.bucketCounts[i]++
			return
		}
	}
	h.bucketCounts[len(bounds)]++
}

func newHistogramState(bounds []float64) *histogramState {
	return &histogramState{
		bucketCounts: make([]uint64, len(bounds)+1),
	}
}

type durationMetricParams struct {
	name      string
	strAttrs  map[string]string
	boolAttrs map[string]bool
}

func appendDurationMetric(ms pmetric.MetricSlice, p durationMetricParams, state *histogramState) {
	m := ms.AppendEmpty()
	m.SetName(p.name)
	m.SetUnit("s")
	m.SetEmptyHistogram()
	m.Histogram().SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	dp := m.Histogram().DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetCount(state.count)
	dp.SetSum(state.sum)
	dp.ExplicitBounds().FromRaw(durationBucketBounds)
	dp.BucketCounts().FromRaw(state.bucketCounts)

	for k, v := range p.strAttrs {
		dp.Attributes().PutStr(k, v)
	}
	for k, v := range p.boolAttrs {
		dp.Attributes().PutBool(k, v)
	}
}

func (m *metricsHandler) appendJobDurationMetric(ms pmetric.MetricSlice, event *github.WorkflowJobEvent) {
	if event == nil || event.GetWorkflowJob() == nil || event.GetAction() != "completed" {
		return
	}
	job := event.GetWorkflowJob()
	startedAt := job.GetStartedAt()
	completedAt := job.GetCompletedAt()
	if startedAt.IsZero() || completedAt.IsZero() {
		return
	}

	isMain := false
	defaultBranch := ""
	if event.GetRepo() != nil && event.GetRepo().DefaultBranch != nil {
		defaultBranch = *event.GetRepo().DefaultBranch
		isMain = job.GetHeadBranch() == defaultBranch
	}

	repo := ""
	if event.GetRepo() != nil {
		repo = event.GetRepo().GetFullName()
	}

	duration := completedAt.Time.Sub(startedAt.Time).Seconds()
	labels := sortedLabels(job.Labels)
	conclusion := job.GetConclusion()

	cacheKey := fmt.Sprintf("hist:job:%s:%s:%s:%s:%s:%t",
		repo, job.GetWorkflowName(), job.GetName(), labels, conclusion, isMain)

	state, ok := m.histogramCache.Get(cacheKey)
	if !ok {
		state = newHistogramState(durationBucketBounds)
	}
	state.observe(duration, durationBucketBounds)
	m.histogramCache.Add(cacheKey, state)

	appendDurationMetric(ms, durationMetricParams{
		name: "workflow.jobs.duration",
		strAttrs: map[string]string{
			"vcs.repository.name":               repo,
			"ci.github.workflow.name":           job.GetWorkflowName(),
			"ci.github.workflow.job.name":       job.GetName(),
			"ci.github.workflow.job.labels":     labels,
			"ci.github.workflow.job.conclusion": conclusion,
		},
		boolAttrs: map[string]bool{
			"ci.github.workflow.job.head_branch.is_main": isMain,
		},
	}, state)
}

func (m *metricsHandler) appendRunDurationMetric(ms pmetric.MetricSlice, event *github.WorkflowRunEvent) {
	if event == nil || event.GetWorkflowRun() == nil || event.GetAction() != "completed" {
		return
	}
	run := event.GetWorkflowRun()
	runStartedAt := run.GetRunStartedAt()
	updatedAt := run.GetUpdatedAt()
	if runStartedAt.IsZero() || updatedAt.IsZero() {
		return
	}

	isMain := false
	defaultBranch := ""
	if event.GetRepo() != nil && event.GetRepo().DefaultBranch != nil {
		defaultBranch = *event.GetRepo().DefaultBranch
		isMain = run.GetHeadBranch() == defaultBranch
	}

	repo := ""
	if event.GetRepo() != nil {
		repo = event.GetRepo().GetFullName()
	}

	duration := updatedAt.Time.Sub(runStartedAt.Time).Seconds()
	conclusion := run.GetConclusion()

	cacheKey := fmt.Sprintf("hist:run:%s:%s:%s:%t",
		repo, run.GetName(), conclusion, isMain)

	state, ok := m.histogramCache.Get(cacheKey)
	if !ok {
		state = newHistogramState(durationBucketBounds)
	}
	state.observe(duration, durationBucketBounds)
	m.histogramCache.Add(cacheKey, state)

	appendDurationMetric(ms, durationMetricParams{
		name: "workflow.runs.duration",
		strAttrs: map[string]string{
			"vcs.repository.name":               repo,
			"ci.github.workflow.name":           run.GetName(),
			"ci.github.workflow.run.conclusion": conclusion,
		},
		boolAttrs: map[string]bool{
			"ci.github.workflow.run.head_branch.is_main": isMain,
		},
	}, state)
}
