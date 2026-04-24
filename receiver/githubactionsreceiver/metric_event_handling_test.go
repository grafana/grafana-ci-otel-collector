package githubactionsreceiver

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/go-github/v85/github"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"
)

func newTestMetricsHandler(t *testing.T) *metricsHandler {
	t.Helper()
	cache, err := lru.New[string, int64](metricsMaxCacheSize)
	require.NoError(t, err)
	histCache, err := lru.New[string, *histogramState](histogramCacheSize)
	require.NoError(t, err)
	return &metricsHandler{
		countersCache:  cache,
		histogramCache: histCache,
	}
}

func TestCacheKey(t *testing.T) {
	tests := []struct {
		desc       string
		repo       string
		labels     string
		status     interface{}
		conclusion interface{}
		isMain     bool
		expected   string
	}{
		{
			desc:       "Basic workflow job on main",
			repo:       "grafana/grafana",
			labels:     "ubuntu-latest",
			status:     "completed",
			conclusion: "success",
			isMain:     true,
			expected:   "grafana/grafana:ubuntu-latest:completed:success:true",
		},
		{
			desc:       "Basic workflow job on feature branch",
			repo:       "grafana/grafana",
			labels:     "ubuntu-latest",
			status:     "completed",
			conclusion: "success",
			isMain:     false,
			expected:   "grafana/grafana:ubuntu-latest:completed:success:false",
		},
		{
			desc:       "Workflow job with self-hosted labels",
			repo:       "grafana/deployment_tools",
			labels:     "self-hosted,linux",
			status:     "in_progress",
			conclusion: "",
			isMain:     false,
			expected:   "grafana/deployment_tools:self-hosted,linux:in_progress::false",
		},
		{
			desc:       "Workflow job with no labels",
			repo:       "foo/bar",
			labels:     "no labels",
			status:     "queued",
			conclusion: "",
			isMain:     false,
			expected:   "foo/bar:no labels:queued::false",
		},
		{
			desc:       "Workflow job with failure",
			repo:       "test/repo",
			labels:     "macos-latest",
			status:     "completed",
			conclusion: "failure",
			isMain:     true,
			expected:   "test/repo:macos-latest:completed:failure:true",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			key := cacheKey(test.repo, test.labels, test.status, test.conclusion, test.isMain)
			require.Equal(t, test.expected, key)
		})
	}
}

func TestCacheStoreAndLoad(t *testing.T) {
	tests := []struct {
		desc       string
		repo       string
		labels     string
		status     interface{}
		conclusion interface{}
		isMain     bool
		value      int64
	}{
		{
			desc:       "Store and retrieve single entry",
			repo:       "grafana/grafana",
			labels:     "ubuntu-latest",
			status:     "completed",
			conclusion: "success",
			isMain:     true,
			value:      147,
		},
		{
			desc:       "Store and retrieve with empty conclusion",
			repo:       "grafana/grafana",
			labels:     "ubuntu-latest",
			status:     "queued",
			conclusion: "",
			isMain:     false,
			value:      5,
		},
		{
			desc:       "Store and retrieve zero value",
			repo:       "test/repo",
			labels:     "self-hosted",
			status:     "completed",
			conclusion: "cancelled",
			isMain:     false,
			value:      0,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			handler := newTestMetricsHandler(t)
			handler.storeInCache(test.repo, test.labels, test.status, test.conclusion, test.isMain, test.value)

			loadedValue, found := handler.loadFromCache(test.repo, test.labels, test.status, test.conclusion, test.isMain)
			require.True(t, found)
			require.Equal(t, test.value, loadedValue)
		})
	}
}

func TestCacheLoadNonExistent(t *testing.T) {
	handler := newTestMetricsHandler(t)
	value, found := handler.loadFromCache("nonexistent/repo", "ubuntu-latest", "completed", "success", false)
	require.False(t, found)
	require.Equal(t, int64(0), value)
}

func TestCacheUpdate(t *testing.T) {
	handler := newTestMetricsHandler(t)

	repo := "grafana/grafana"
	labels := "ubuntu-latest"
	status := "completed"
	conclusion := "success"

	handler.storeInCache(repo, labels, status, conclusion, true, 100)

	value, found := handler.loadFromCache(repo, labels, status, conclusion, true)
	require.True(t, found)
	require.Equal(t, int64(100), value)

	// Update value
	handler.storeInCache(repo, labels, status, conclusion, true, 150)

	value, found = handler.loadFromCache(repo, labels, status, conclusion, true)
	require.True(t, found)
	require.Equal(t, int64(150), value)
}

func TestCacheLRUEviction(t *testing.T) {
	handler := newTestMetricsHandler(t)

	entriesToAdd := 1000

	for i := range entriesToAdd {
		repo := fmt.Sprintf("test-repo-%d", i)
		handler.storeInCache(repo, "ubuntu-latest", "completed", "success", false, int64(i))
	}

	oldRepo := "grafana/loki"
	handler.storeInCache(oldRepo, "ubuntu-latest", "completed", "success", false, 999)

	value, found := handler.loadFromCache(oldRepo, "ubuntu-latest", "completed", "success", false)
	require.True(t, found)
	require.Equal(t, int64(999), value)

	// Access again to make it recently used
	_, found = handler.loadFromCache(oldRepo, "ubuntu-latest", "completed", "success", false)
	require.True(t, found)

	require.Greater(t, handler.countersCache.Len(), 0)
}

func TestCacheMultipleReposAndLabels(t *testing.T) {
	handler := newTestMetricsHandler(t)

	repos := []string{"grafana/grafana", "grafana/tempo", "grafana/loki"}
	labels := []string{"ubuntu-latest", "macos-latest", "self-hosted"}
	statuses := []string{"completed", "queued", "in_progress"}
	conclusions := []string{"success", "failure", ""}

	count := int64(0)
	for _, repo := range repos {
		for _, label := range labels {
			for _, status := range statuses {
				for _, conclusion := range conclusions {
					count++
					handler.storeInCache(repo, label, status, conclusion, false, count)
				}
			}
		}
	}

	count = int64(0)
	for _, repo := range repos {
		for _, label := range labels {
			for _, status := range statuses {
				for _, conclusion := range conclusions {
					count++
					value, found := handler.loadFromCache(repo, label, status, conclusion, false)
					require.True(t, found, "Failed to find: %s, %s, %s, %s", repo, label, status, conclusion)
					require.Equal(t, count, value)
				}
			}
		}
	}
}

func TestCacheIncrement(t *testing.T) {
	handler := newTestMetricsHandler(t)

	repo := "grafana/grafana"
	labels := "ubuntu-latest"
	status := "completed"
	conclusion := "success"

	// Not found
	curVal, found := handler.loadFromCache(repo, labels, status, conclusion, true)
	require.False(t, found)
	require.Equal(t, int64(0), curVal)

	// Store first event
	handler.storeInCache(repo, labels, status, conclusion, true, 1)

	// Found
	curVal, found = handler.loadFromCache(repo, labels, status, conclusion, true)
	require.True(t, found)
	require.Equal(t, int64(1), curVal)

	// Increment
	handler.storeInCache(repo, labels, status, conclusion, true, curVal+1)
	curVal, found = handler.loadFromCache(repo, labels, status, conclusion, true)
	require.True(t, found)
	require.Equal(t, int64(2), curVal)
}

func TestCacheIsMainIndependence(t *testing.T) {
	handler := newTestMetricsHandler(t)

	repo := "grafana/grafana"
	labels := "ubuntu-latest"
	status := "completed"
	conclusion := "success"

	// Simulate a job on main branch
	handler.storeInCache(repo, labels, status, conclusion, true, 1)

	// Simulate a job on a feature branch
	handler.storeInCache(repo, labels, status, conclusion, false, 1)

	// Each should have independent counters
	mainVal, found := handler.loadFromCache(repo, labels, status, conclusion, true)
	require.True(t, found)
	require.Equal(t, int64(1), mainVal, "is_main=true counter should be 1")

	featureVal, found := handler.loadFromCache(repo, labels, status, conclusion, false)
	require.True(t, found)
	require.Equal(t, int64(1), featureVal, "is_main=false counter should be 1")

	// Increment only the main branch counter
	handler.storeInCache(repo, labels, status, conclusion, true, mainVal+1)

	mainVal, _ = handler.loadFromCache(repo, labels, status, conclusion, true)
	require.Equal(t, int64(2), mainVal, "is_main=true counter should be 2 after increment")

	featureVal, _ = handler.loadFromCache(repo, labels, status, conclusion, false)
	require.Equal(t, int64(1), featureVal, "is_main=false counter should still be 1")
}

// newFullMetricsHandler creates a metricsHandler with all fields populated, suitable
// for concurrency tests that exercise workflowJobEventToMetrics / workflowRunEventToMetrics.
func newFullMetricsHandler(t *testing.T) *metricsHandler {
	t.Helper()
	cfg := createDefaultConfig().(*Config)
	return newMetricsHandler(receivertest.NewNopSettings(receivertest.NopType), cfg, zap.NewNop())
}

func TestWorkflowJobEventToMetricsConcurrency(t *testing.T) {
	jobPayload, err := os.ReadFile("./testdata/completed/5_workflow_job_completed.json")
	require.NoError(t, err)

	raw, err := github.ParseWebHook("workflow_job", jobPayload)
	require.NoError(t, err)
	event, ok := raw.(*github.WorkflowJobEvent)
	require.True(t, ok)

	handler := newFullMetricsHandler(t)

	const goroutines = 20
	const callsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range callsPerGoroutine {
				_ = handler.workflowJobEventToMetrics(event)
			}
		}()
	}
	wg.Wait()
}

func TestWorkflowRunEventToMetricsConcurrency(t *testing.T) {
	runPayload, err := os.ReadFile("./testdata/completed/8_workflow_run_completed.json")
	require.NoError(t, err)

	raw, err := github.ParseWebHook("workflow_run", runPayload)
	require.NoError(t, err)
	event, ok := raw.(*github.WorkflowRunEvent)
	require.True(t, ok)

	handler := newFullMetricsHandler(t)

	const goroutines = 20
	const callsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range callsPerGoroutine {
				_ = handler.workflowRunEventToMetrics(event)
			}
		}()
	}
	wg.Wait()
}

func TestWorkflowEventsMixedConcurrency(t *testing.T) {
	jobPayload, err := os.ReadFile("./testdata/completed/5_workflow_job_completed.json")
	require.NoError(t, err)
	runPayload, err := os.ReadFile("./testdata/completed/8_workflow_run_completed.json")
	require.NoError(t, err)

	rawJob, err := github.ParseWebHook("workflow_job", jobPayload)
	require.NoError(t, err)
	jobEvent, ok := rawJob.(*github.WorkflowJobEvent)
	require.True(t, ok)

	rawRun, err := github.ParseWebHook("workflow_run", runPayload)
	require.NoError(t, err)
	runEvent, ok := rawRun.(*github.WorkflowRunEvent)
	require.True(t, ok)

	handler := newFullMetricsHandler(t)

	const goroutines = 20
	const callsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range callsPerGoroutine {
				_ = handler.workflowJobEventToMetrics(jobEvent)
			}
		}()
		go func() {
			defer wg.Done()
			for range callsPerGoroutine {
				_ = handler.workflowRunEventToMetrics(runEvent)
			}
		}()
	}
	wg.Wait()
}

func TestSweepStaleHistograms(t *testing.T) {
	handler := newTestMetricsHandler(t)

	// Add a fresh entry
	fresh := newHistogramState(durationBucketBounds)
	fresh.lastSeen = time.Now()
	handler.histogramCache.Add("fresh-key", fresh)

	// Add a stale entry (last seen 25h ago, beyond 24h TTL)
	stale := newHistogramState(durationBucketBounds)
	stale.lastSeen = time.Now().Add(-25 * time.Hour)
	handler.histogramCache.Add("stale-key", stale)

	require.Equal(t, 2, handler.histogramCache.Len())

	handler.sweepStaleHistograms()

	require.Equal(t, 1, handler.histogramCache.Len())
	_, freshFound := handler.histogramCache.Get("fresh-key")
	require.True(t, freshFound)
	_, staleFound := handler.histogramCache.Get("stale-key")
	require.False(t, staleFound)
}
