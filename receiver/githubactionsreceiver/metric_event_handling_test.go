package githubactionsreceiver

import (
	"fmt"
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stretchr/testify/require"
)

func resetCache() {
	var err error
	cache, err = lru.New[string, int64](metricsMaxCacheSize)
	if err != nil {
		panic(fmt.Sprintf("Failed to reset cache: %v", err))
	}
}

func TestCacheKey(t *testing.T) {
	tests := []struct {
		desc       string
		repo       string
		labels     string
		status     interface{}
		conclusion interface{}
		expected   string
	}{
		{
			desc:       "Basic workflow job",
			repo:       "grafana/grafana",
			labels:     "ubuntu-latest",
			status:     "completed",
			conclusion: "success",
			expected:   "grafana/grafana:ubuntu-latest:completed:success",
		},
		{
			desc:       "Workflow job with self-hosted labels",
			repo:       "grafana/deployment_tools",
			labels:     "self-hosted,linux",
			status:     "in_progress",
			conclusion: "",
			expected:   "grafana/deployment_tools:self-hosted,linux:in_progress:",
		},
		{
			desc:       "Workflow job with no labels",
			repo:       "foo/bar",
			labels:     "no labels",
			status:     "queued",
			conclusion: "",
			expected:   "foo/bar:no labels:queued:",
		},
		{
			desc:       "Workflow job with failure",
			repo:       "test/repo",
			labels:     "macos-latest",
			status:     "completed",
			conclusion: "failure",
			expected:   "test/repo:macos-latest:completed:failure",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			key := cacheKey(test.repo, test.labels, test.status, test.conclusion)
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
		value      int64
	}{
		{
			desc:       "Store and retrieve single entry",
			repo:       "grafana/grafana",
			labels:     "ubuntu-latest",
			status:     "completed",
			conclusion: "success",
			value:      147,
		},
		{
			desc:       "Store and retrieve with empty conclusion",
			repo:       "grafana/grafana",
			labels:     "ubuntu-latest",
			status:     "queued",
			conclusion: "",
			value:      5,
		},
		{
			desc:       "Store and retrieve zero value",
			repo:       "test/repo",
			labels:     "self-hosted",
			status:     "completed",
			conclusion: "cancelled",
			value:      0,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			storeInCache(test.repo, test.labels, test.status, test.conclusion, test.value)

			loadedValue, found := loadFromCache(test.repo, test.labels, test.status, test.conclusion)
			require.True(t, found)
			require.Equal(t, test.value, loadedValue)
		})
	}
}

func TestCacheLoadNonExistent(t *testing.T) {
	value, found := loadFromCache("nonexistent/repo", "ubuntu-latest", "completed", "success")
	require.False(t, found)
	require.Equal(t, int64(0), value)
}

func TestCacheUpdate(t *testing.T) {
	resetCache()

	repo := "grafana/grafana"
	labels := "ubuntu-latest"
	status := "completed"
	conclusion := "success"

	storeInCache(repo, labels, status, conclusion, 100)

	value, found := loadFromCache(repo, labels, status, conclusion)
	require.True(t, found)
	require.Equal(t, int64(100), value)

	// Update value
	storeInCache(repo, labels, status, conclusion, 150)

	value, found = loadFromCache(repo, labels, status, conclusion)
	require.True(t, found)
	require.Equal(t, int64(150), value)
}

func TestCacheLRUEviction(t *testing.T) {
	resetCache()

	entriesToAdd := 1000

	for i := range entriesToAdd {
		repo := fmt.Sprintf("test-repo-%d", i)
		storeInCache(repo, "ubuntu-latest", "completed", "success", int64(i))
	}

	oldRepo := "grafana/loki"
	storeInCache(oldRepo, "ubuntu-latest", "completed", "success", 999)

	value, found := loadFromCache(oldRepo, "ubuntu-latest", "completed", "success")
	require.True(t, found)
	require.Equal(t, int64(999), value)

	// Access again to make it recently used
	_, found = loadFromCache(oldRepo, "ubuntu-latest", "completed", "success")
	require.True(t, found)

	require.Greater(t, cache.Len(), 0)
}

func TestCacheMultipleReposAndLabels(t *testing.T) {
	resetCache()

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
					storeInCache(repo, label, status, conclusion, count)
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
					value, found := loadFromCache(repo, label, status, conclusion)
					require.True(t, found, "Failed to find: %s, %s, %s, %s", repo, label, status, conclusion)
					require.Equal(t, count, value)
				}
			}
		}
	}
}

func TestCacheIncrement(t *testing.T) {
	resetCache()

	repo := "grafana/grafana"
	labels := "ubuntu-latest"
	status := "completed"
	conclusion := "success"

	// Not found
	curVal, found := loadFromCache(repo, labels, status, conclusion)
	require.False(t, found)
	require.Equal(t, int64(0), curVal)

	// Store first event
	storeInCache(repo, labels, status, conclusion, 1)

	// Found
	curVal, found = loadFromCache(repo, labels, status, conclusion)
	require.True(t, found)
	require.Equal(t, int64(1), curVal)

	// Increment
	storeInCache(repo, labels, status, conclusion, curVal+1)
	curVal, found = loadFromCache(repo, labels, status, conclusion)
	require.True(t, found)
	require.Equal(t, int64(2), curVal)
}
