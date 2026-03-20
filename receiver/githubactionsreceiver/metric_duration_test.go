package githubactionsreceiver

import (
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestComputeBucketCounts(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected []uint64
	}{
		{
			name:     "zero value falls in first bucket",
			value:    0,
			expected: []uint64{1, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "on boundary falls in that bucket",
			value:    5,
			expected: []uint64{1, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "between boundaries",
			value:    10,
			expected: []uint64{0, 1, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "on second boundary",
			value:    15,
			expected: []uint64{0, 1, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "between 30 and 60",
			value:    45,
			expected: []uint64{0, 0, 0, 1, 0, 0, 0, 0},
		},
		{
			name:     "overflow beyond last boundary",
			value:    5000,
			expected: []uint64{0, 0, 0, 0, 0, 0, 0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeBucketCounts(tt.value, durationBucketBounds)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestSortedLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   []string
		expected string
	}{
		{
			name:     "empty returns no labels",
			labels:   []string{},
			expected: "no labels",
		},
		{
			name:     "nil returns no labels",
			labels:   nil,
			expected: "no labels",
		},
		{
			name:     "single label",
			labels:   []string{"Ubuntu-Latest"},
			expected: "ubuntu-latest",
		},
		{
			name:     "multiple labels sorted and lowercased",
			labels:   []string{"Self-Hosted", "Linux", "ARM64"},
			expected: "arm64,linux,self-hosted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedLabels(tt.labels)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestAppendJobDurationMetric_Completed(t *testing.T) {
	payload, err := os.ReadFile("./testdata/completed/5_workflow_job_completed.json")
	require.NoError(t, err)

	raw, err := github.ParseWebHook("workflow_job", payload)
	require.NoError(t, err)
	event := raw.(*github.WorkflowJobEvent)

	ms := pmetric.NewMetricSlice()
	appendJobDurationMetric(ms, event)

	require.Equal(t, 1, ms.Len())

	m := ms.At(0)
	require.Equal(t, "workflow.jobs.duration", m.Name())
	require.Equal(t, "s", m.Unit())
	require.Equal(t, pmetric.MetricTypeHistogram, m.Type())
	require.Equal(t, pmetric.AggregationTemporalityDelta, m.Histogram().AggregationTemporality())

	require.Equal(t, 1, m.Histogram().DataPoints().Len())
	dp := m.Histogram().DataPoints().At(0)

	// Duration = completed_at(10:11:44) - started_at(10:11:34) = 10s
	require.Equal(t, uint64(1), dp.Count())
	require.Equal(t, 10.0, dp.Sum())

	// 10s falls in bucket [5, 15] → index 1
	expectedBuckets := []uint64{0, 1, 0, 0, 0, 0, 0, 0}
	require.Equal(t, expectedBuckets, dp.BucketCounts().AsRaw())

	attrs := dp.Attributes()
	assertStrAttr(t, attrs, "vcs.repository.name", "foo/webhook-testing")
	assertStrAttr(t, attrs, "ci.github.workflow.name", "Tests")
	assertStrAttr(t, attrs, "ci.github.workflow.job.name", "pre-commit")
	assertStrAttr(t, attrs, "ci.github.workflow.job.labels", "ubuntu-latest")
	assertStrAttr(t, attrs, "ci.github.workflow.job.conclusion", "success")
	assertBoolAttr(t, attrs, "ci.github.workflow.job.head_branch.is_main", true)
}

func TestAppendJobDurationMetric_NonCompleted(t *testing.T) {
	payload, err := os.ReadFile("./testdata/queued/1_workflow_job_queued.json")
	require.NoError(t, err)

	raw, err := github.ParseWebHook("workflow_job", payload)
	require.NoError(t, err)
	event := raw.(*github.WorkflowJobEvent)

	ms := pmetric.NewMetricSlice()
	appendJobDurationMetric(ms, event)
	require.Equal(t, 0, ms.Len())
}

func TestAppendJobDurationMetric_NilEvent(t *testing.T) {
	ms := pmetric.NewMetricSlice()
	appendJobDurationMetric(ms, nil)
	require.Equal(t, 0, ms.Len())
}

func TestAppendRunDurationMetric_Completed(t *testing.T) {
	payload, err := os.ReadFile("./testdata/completed/8_workflow_run_completed.json")
	require.NoError(t, err)

	raw, err := github.ParseWebHook("workflow_run", payload)
	require.NoError(t, err)
	event := raw.(*github.WorkflowRunEvent)

	ms := pmetric.NewMetricSlice()
	appendRunDurationMetric(ms, event)

	require.Equal(t, 1, ms.Len())

	m := ms.At(0)
	require.Equal(t, "workflow.runs.duration", m.Name())
	require.Equal(t, "s", m.Unit())
	require.Equal(t, pmetric.MetricTypeHistogram, m.Type())
	require.Equal(t, pmetric.AggregationTemporalityDelta, m.Histogram().AggregationTemporality())

	require.Equal(t, 1, m.Histogram().DataPoints().Len())
	dp := m.Histogram().DataPoints().At(0)

	// Duration = updated_at(10:12:10) - run_started_at(10:11:25) = 45s
	require.Equal(t, uint64(1), dp.Count())
	require.Equal(t, 45.0, dp.Sum())

	// 45s falls in bucket [30, 60] → index 3
	expectedBuckets := []uint64{0, 0, 0, 1, 0, 0, 0, 0}
	require.Equal(t, expectedBuckets, dp.BucketCounts().AsRaw())

	attrs := dp.Attributes()
	assertStrAttr(t, attrs, "vcs.repository.name", "foo/webhook-testing")
	assertStrAttr(t, attrs, "ci.github.workflow.name", "Tests")
	assertStrAttr(t, attrs, "ci.github.workflow.run.conclusion", "success")
	assertBoolAttr(t, attrs, "ci.github.workflow.run.head_branch.is_main", true)
}

func TestAppendRunDurationMetric_NonCompleted(t *testing.T) {
	payload, err := os.ReadFile("./testdata/in_progress/10_workflow_run_in_progress.json")
	require.NoError(t, err)

	raw, err := github.ParseWebHook("workflow_run", payload)
	require.NoError(t, err)
	event := raw.(*github.WorkflowRunEvent)

	ms := pmetric.NewMetricSlice()
	appendRunDurationMetric(ms, event)
	require.Equal(t, 0, ms.Len())
}

func makeJobEvent(action, conclusion, headBranch, defaultBranch string, startedAt, completedAt time.Time) *github.WorkflowJobEvent {
	return &github.WorkflowJobEvent{
		Action: &action,
		WorkflowJob: &github.WorkflowJob{
			WorkflowName: strPtr("CI"),
			Name:         strPtr("build"),
			Conclusion:   &conclusion,
			HeadBranch:   &headBranch,
			Labels:       []string{"ubuntu-latest"},
			StartedAt:    &github.Timestamp{Time: startedAt},
			CompletedAt:  &github.Timestamp{Time: completedAt},
		},
		Repo: &github.Repository{
			FullName:      strPtr("org/repo"),
			DefaultBranch: &defaultBranch,
		},
	}
}

func makeRunEvent(action, conclusion, headBranch, defaultBranch string, runStartedAt, updatedAt time.Time) *github.WorkflowRunEvent {
	return &github.WorkflowRunEvent{
		Action: &action,
		WorkflowRun: &github.WorkflowRun{
			Name:         strPtr("CI"),
			Conclusion:   &conclusion,
			HeadBranch:   &headBranch,
			RunStartedAt: &github.Timestamp{Time: runStartedAt},
			UpdatedAt:    &github.Timestamp{Time: updatedAt},
		},
		Repo: &github.Repository{
			FullName:      strPtr("org/repo"),
			DefaultBranch: &defaultBranch,
		},
	}
}

func strPtr(s string) *string { return &s }

func TestAppendJobDurationMetric_Scenarios(t *testing.T) {
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name               string
		event              *github.WorkflowJobEvent
		wantMetric         bool
		wantSum            float64
		wantConclusion     string
		wantIsMain         bool
		wantBucketIndex    int
	}{
		{
			name:            "completed success on main",
			event:           makeJobEvent("completed", "success", "main", "main", base, base.Add(10*time.Second)),
			wantMetric:      true,
			wantSum:         10.0,
			wantConclusion:  "success",
			wantIsMain:      true,
			wantBucketIndex: 1, // 5-15 bucket
		},
		{
			name:            "completed failure on main",
			event:           makeJobEvent("completed", "failure", "main", "main", base, base.Add(45*time.Second)),
			wantMetric:      true,
			wantSum:         45.0,
			wantConclusion:  "failure",
			wantIsMain:      true,
			wantBucketIndex: 3, // 30-60 bucket
		},
		{
			name:            "completed cancelled on main",
			event:           makeJobEvent("completed", "cancelled", "main", "main", base, base.Add(3*time.Second)),
			wantMetric:      true,
			wantSum:         3.0,
			wantConclusion:  "cancelled",
			wantIsMain:      true,
			wantBucketIndex: 0, // 0-5 bucket
		},
		{
			name:            "completed timed_out on feature branch",
			event:           makeJobEvent("completed", "timed_out", "feature/foo", "main", base, base.Add(2000*time.Second)),
			wantMetric:      true,
			wantSum:         2000.0,
			wantConclusion:  "timed_out",
			wantIsMain:      false,
			wantBucketIndex: 7, // overflow bucket
		},
		{
			name:       "queued action produces no metric",
			event:      makeJobEvent("queued", "", "main", "main", base, time.Time{}),
			wantMetric: false,
		},
		{
			name:       "in_progress action produces no metric",
			event:      makeJobEvent("in_progress", "", "main", "main", base, time.Time{}),
			wantMetric: false,
		},
		{
			name:       "completed but zero started_at produces no metric",
			event:      makeJobEvent("completed", "success", "main", "main", time.Time{}, base.Add(10*time.Second)),
			wantMetric: false,
		},
		{
			name:       "nil event produces no metric",
			event:      nil,
			wantMetric: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := pmetric.NewMetricSlice()
			appendJobDurationMetric(ms, tt.event)

			if !tt.wantMetric {
				require.Equal(t, 0, ms.Len())
				return
			}

			require.Equal(t, 1, ms.Len())
			dp := ms.At(0).Histogram().DataPoints().At(0)
			require.Equal(t, uint64(1), dp.Count())
			require.Equal(t, tt.wantSum, dp.Sum())
			assertStrAttr(t, dp.Attributes(), "ci.github.workflow.job.conclusion", tt.wantConclusion)
			assertBoolAttr(t, dp.Attributes(), "ci.github.workflow.job.head_branch.is_main", tt.wantIsMain)

			buckets := dp.BucketCounts().AsRaw()
			for i, count := range buckets {
				if i == tt.wantBucketIndex {
					require.Equal(t, uint64(1), count, "expected count=1 in bucket %d", i)
				} else {
					require.Equal(t, uint64(0), count, "expected count=0 in bucket %d", i)
				}
			}
		})
	}
}

func TestAppendRunDurationMetric_Scenarios(t *testing.T) {
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		event           *github.WorkflowRunEvent
		wantMetric      bool
		wantSum         float64
		wantConclusion  string
		wantIsMain      bool
		wantBucketIndex int
	}{
		{
			name:            "completed success on main",
			event:           makeRunEvent("completed", "success", "main", "main", base, base.Add(45*time.Second)),
			wantMetric:      true,
			wantSum:         45.0,
			wantConclusion:  "success",
			wantIsMain:      true,
			wantBucketIndex: 3, // 30-60 bucket
		},
		{
			name:            "completed failure on feature branch",
			event:           makeRunEvent("completed", "failure", "feature/bar", "main", base, base.Add(120*time.Second)),
			wantMetric:      true,
			wantSum:         120.0,
			wantConclusion:  "failure",
			wantIsMain:      false,
			wantBucketIndex: 4, // 60-300 bucket
		},
		{
			name:            "completed cancelled",
			event:           makeRunEvent("completed", "cancelled", "main", "main", base, base.Add(8*time.Second)),
			wantMetric:      true,
			wantSum:         8.0,
			wantConclusion:  "cancelled",
			wantIsMain:      true,
			wantBucketIndex: 1, // 5-15 bucket
		},
		{
			name:       "in_progress produces no metric",
			event:      makeRunEvent("in_progress", "", "main", "main", base, base.Add(10*time.Second)),
			wantMetric: false,
		},
		{
			name:       "requested produces no metric",
			event:      makeRunEvent("requested", "", "main", "main", time.Time{}, time.Time{}),
			wantMetric: false,
		},
		{
			name:       "nil event produces no metric",
			event:      nil,
			wantMetric: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := pmetric.NewMetricSlice()
			appendRunDurationMetric(ms, tt.event)

			if !tt.wantMetric {
				require.Equal(t, 0, ms.Len())
				return
			}

			require.Equal(t, 1, ms.Len())
			dp := ms.At(0).Histogram().DataPoints().At(0)
			require.Equal(t, uint64(1), dp.Count())
			require.Equal(t, tt.wantSum, dp.Sum())
			assertStrAttr(t, dp.Attributes(), "ci.github.workflow.run.conclusion", tt.wantConclusion)
			assertBoolAttr(t, dp.Attributes(), "ci.github.workflow.run.head_branch.is_main", tt.wantIsMain)

			buckets := dp.BucketCounts().AsRaw()
			for i, count := range buckets {
				if i == tt.wantBucketIndex {
					require.Equal(t, uint64(1), count, "expected count=1 in bucket %d", i)
				} else {
					require.Equal(t, uint64(0), count, "expected count=0 in bucket %d", i)
				}
			}
		})
	}
}

func assertStrAttr(t *testing.T, attrs pcommon.Map, key, expected string) {
	t.Helper()
	v, ok := attrs.Get(key)
	require.True(t, ok, "attribute %q not found", key)
	require.Equal(t, expected, v.Str(), "attribute %q", key)
}

func assertBoolAttr(t *testing.T, attrs pcommon.Map, key string, expected bool) {
	t.Helper()
	v, ok := attrs.Get(key)
	require.True(t, ok, "attribute %q not found", key)
	require.Equal(t, expected, v.Bool(), "attribute %q", key)
}
