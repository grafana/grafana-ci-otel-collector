package githubactionsreceiver

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

func TestParseTimestamp(t *testing.T) {
	tests := map[string]struct {
		logline      string
		expectError  bool
		expectedTime time.Time
	}{
		"github-10-7": {
			logline:      "2025-07-17T10:26:38.7039891Z ##[group]Run actions/cache@v4",
			expectedTime: time.Date(2025, time.July, 17, 10, 26, 38, 703989100, time.UTC),
		},
		"rfc3339-nano": {
			logline:      "2025-07-17T10:26:38.703989101Z ##[group]Run actions/cache@v4",
			expectedTime: time.Date(2025, time.July, 17, 10, 26, 38, 703989101, time.UTC),
		},
		"rfc3339": {
			logline:      "2025-07-17T10:26:38Z ##[group]Run actions/cache@v4",
			expectedTime: time.Date(2025, time.July, 17, 10, 26, 38, 0, time.UTC),
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			logger := zap.NewNop()
			ts, _, ok := parseTimestamp([]byte(test.logline), logger)
			if test.expectError {
				require.False(t, ok)
			} else {
				require.True(t, ok)
				require.Equal(t, test.expectedTime, ts)
			}
		})
	}
}

func TestExtractStepNumberFromFileName(t *testing.T) {
	tests := map[string]struct {
		fileName      string
		jobName       string
		expectedStep  int
		expectError   bool
		errorContains string
	}{
		"step number with underscore": {
			fileName:     "test/2_Run tests.txt",
			jobName:      "test",
			expectedStep: 2,
		},
		"system.txt file should be skipped": {
			fileName:      "Shellcheck scripts/system.txt",
			jobName:       "Shellcheck scripts",
			expectError:   true,
			errorContains: "skipping system file",
		},
		"job name with spaces": {
			fileName:     "Build and Test/1_Setup.txt",
			jobName:      "Build and Test",
			expectedStep: 1,
		},
		"invalid step number": {
			fileName:      "build/abc_Invalid.txt",
			jobName:       "build",
			expectError:   true,
			errorContains: "invalid syntax",
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			stepNum, err := extractStepNumberFromFileName(test.fileName, test.jobName)
			if test.expectError {
				require.Error(t, err)
				if test.errorContains != "" {
					require.Contains(t, err.Error(), test.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedStep, stepNum)
			}
		})
	}
}

type recordingConsumer struct {
	calls []plog.Logs
}

func (c *recordingConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *recordingConsumer) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	c.calls = append(c.calls, ld)
	return nil
}

func buildMultiJobZip(t *testing.T, jobs []string, linesPerJob int) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, jobName := range jobs {
		_, err := zw.Create(jobName + "/")
		require.NoError(t, err)

		stepFile, err := zw.Create(fmt.Sprintf("%s/1_step.log", jobName))
		require.NoError(t, err)

		for l := range linesPerJob {
			ts := base.Add(time.Duration(l) * time.Second).Format(time.RFC3339)
			_, err := fmt.Fprintf(stepFile, "%s log line %d\n", ts, l)
			require.NoError(t, err)
		}
	}

	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func newLogsTestServer(t *testing.T, zipData []byte) (*github.Client, func()) {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/logs") {
			w.Header().Set("Location", server.URL+"/zip")
			w.WriteHeader(http.StatusFound)
			return
		}
		if r.URL.Path == "/zip" {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipData)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	cfg := createDefaultConfig().(*Config)
	cfg.GitHubAPIConfig.BaseURL = server.URL
	cfg.GitHubAPIConfig.UploadURL = server.URL
	cfg.GitHubAPIConfig.Auth.Token = "testtoken"

	client := github.NewClient(nil).WithAuthToken(cfg.GitHubAPIConfig.Auth.Token)
	client, err := client.WithEnterpriseURLs(cfg.GitHubAPIConfig.BaseURL, cfg.GitHubAPIConfig.UploadURL)
	require.NoError(t, err)

	return client, server.Close
}

// TestEventToLogsStreaming verifies that eventToLogs calls ConsumeLogs once per
// job (streaming / bounded-memory behaviour) and that each call contains the
// correct number of log records scoped to the right job name.
func TestEventToLogsStreaming(t *testing.T) {
	jobNames := []string{"build", "test", "lint"}
	linesPerJob := 5

	zipData := buildMultiJobZip(t, jobNames, linesPerJob)

	payload, err := os.ReadFile("./testdata/completed/8_workflow_run_completed.json")
	require.NoError(t, err)
	raw, err := github.ParseWebHook("workflow_run", payload)
	require.NoError(t, err)
	event := raw.(*github.WorkflowRunEvent)

	ghClient, cleanup := newLogsTestServer(t, zipData)
	defer cleanup()

	cfg := createDefaultConfig().(*Config)
	consumer := &recordingConsumer{}

	err = eventToLogs(context.Background(), event, cfg, ghClient, consumer, zap.NewNop(), false)
	require.NoError(t, err)

	// One ConsumeLogs call per job.
	require.Len(t, consumer.calls, len(jobNames),
		"expected one ConsumeLogs call per job (streaming); got %d for %d jobs",
		len(consumer.calls), len(jobNames))

	// Each call should carry exactly linesPerJob records under the correct job scope.
	seenJobs := make(map[string]int) // job name -> record count
	for _, ld := range consumer.calls {
		require.Equal(t, 1, ld.ResourceLogs().Len(), "each ConsumeLogs batch should have exactly one ResourceLogs")
		rl := ld.ResourceLogs().At(0)
		require.Equal(t, 1, rl.ScopeLogs().Len())
		sl := rl.ScopeLogs().At(0)

		jobAttr, ok := sl.Scope().Attributes().Get("ci.github.workflow.job.name")
		require.True(t, ok, "scope must carry ci.github.workflow.job.name attribute")
		seenJobs[jobAttr.Str()] = sl.LogRecords().Len()
	}

	for _, jobName := range jobNames {
		count, ok := seenJobs[jobName]
		require.True(t, ok, "no ConsumeLogs call found for job %q", jobName)
		require.Equal(t, linesPerJob, count,
			"job %q: expected %d log records, got %d", jobName, linesPerJob, count)
	}
}
