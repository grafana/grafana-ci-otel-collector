package githubactionsreceiver

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// createMultiJobZip builds a ZIP with the given number of jobs, steps per job, and log lines per step.
// Each log line has an RFC3339 timestamp; no multi-line entries are generated (linesPerStep < 10).
func createMultiJobZip(t *testing.T, jobs, stepsPerJob, linesPerStep int) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	for j := range jobs {
		jobName := fmt.Sprintf("job-%d", j)
		_, err := zw.Create(jobName + "/")
		require.NoError(t, err)

		baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		for s := range stepsPerJob {
			stepFile, err := zw.Create(fmt.Sprintf("%s/%d_step.log", jobName, s+1))
			require.NoError(t, err)
			for l := range linesPerStep {
				ts := baseTime.Add(time.Duration(j*stepsPerJob*linesPerStep+s*linesPerStep+l) * time.Second)
				line := fmt.Sprintf("%s job=%d step=%d line=%d\n", ts.Format(time.RFC3339), j, s+1, l)
				_, err := stepFile.Write([]byte(line))
				require.NoError(t, err)
			}
		}
	}

	require.NoError(t, zw.Close())
	return buf.Bytes()
}

// TestEventToLogsMultipleJobsParallel verifies that concurrent job processing produces
// the same output as sequential processing would: all jobs present, correct log record
// counts, correct scope attributes, and alphabetical job ordering.
func TestEventToLogsMultipleJobsParallel(t *testing.T) {
	const (
		numJobs      = 5
		stepsPerJob  = 3
		linesPerStep = 4
	)

	zipData := createMultiJobZip(t, numJobs, stepsPerJob, linesPerStep)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/logs") {
			w.Header().Set("Location", server.URL+"/fetch")
			w.WriteHeader(http.StatusFound)
			return
		}
		if r.URL.Path == "/fetch" {
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipData)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	payload, err := os.ReadFile("./testdata/completed/8_workflow_run_completed.json")
	require.NoError(t, err)

	event, err := github.ParseWebHook("workflow_run", payload)
	require.NoError(t, err)

	cfg := createDefaultConfig().(*Config)
	cfg.GitHubAPIConfig.BaseURL = server.URL
	cfg.GitHubAPIConfig.UploadURL = server.URL
	cfg.GitHubAPIConfig.Auth.Token = "testtoken"

	client := github.NewClient(nil)
	client, err = client.WithEnterpriseURLs(cfg.GitHubAPIConfig.BaseURL, cfg.GitHubAPIConfig.UploadURL)
	require.NoError(t, err)
	client = client.WithAuthToken(cfg.GitHubAPIConfig.Auth.Token)

	logs, err := eventToLogs(event, cfg, client, zap.NewNop(), false)
	require.NoError(t, err)
	require.NotNil(t, logs)

	require.Equal(t, 1, logs.ResourceLogs().Len(), "expected one resource")
	scopeLogs := logs.ResourceLogs().At(0).ScopeLogs()
	require.Equal(t, numJobs, scopeLogs.Len(), "expected one ScopeLogs per job")

	// Jobs are sorted alphabetically (job-0 … job-4), so index == job number.
	for i := range numJobs {
		sl := scopeLogs.At(i)
		expectedJobName := fmt.Sprintf("job-%d", i)

		jobAttr, ok := sl.Scope().Attributes().Get("ci.github.workflow.job.name")
		assert.True(t, ok, "scope %d missing job name attribute", i)
		assert.Equal(t, expectedJobName, jobAttr.Str(), "scope %d wrong job name", i)

		expectedRecords := stepsPerJob * linesPerStep
		assert.Equal(t, expectedRecords, sl.LogRecords().Len(),
			"scope %d (%s): expected %d log records", i, expectedJobName, expectedRecords)

		// Spot-check that step number attributes are populated.
		for r := range sl.LogRecords().Len() {
			record := sl.LogRecords().At(r)
			stepAttr, ok := record.Attributes().Get("ci.github.workflow.job.step.number")
			assert.True(t, ok, "scope %d record %d missing step number", i, r)
			assert.Greater(t, stepAttr.Int(), int64(0), "scope %d record %d step number should be > 0", i, r)
		}
	}
}
