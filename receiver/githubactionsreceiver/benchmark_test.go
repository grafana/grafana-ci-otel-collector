// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

	"github.com/google/go-github/v75/github"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// createBenchmarkZip creates a zip file with configurable size for benchmarking log processing
func createBenchmarkZip(b *testing.B, jobs, stepsPerJob, linesPerStep int) []byte {
	b.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	for j := range jobs {
		jobName := fmt.Sprintf("job-%d", j)
		// Create job directory
		_, err := zw.Create(jobName + "/")
		require.NoError(b, err)

		for s := range stepsPerJob {
			stepFile, err := zw.Create(fmt.Sprintf("%s/%d_step.log", jobName, s+1))
			require.NoError(b, err)

			// Generate realistic log lines with timestamps
			baseTime := time.Now().Add(-time.Hour)
			for l := range linesPerStep {
				timestamp := baseTime.Add(time.Duration(l) * time.Second)

				line := fmt.Sprintf("%s Log line %d for step %d in job %s\n",
					timestamp.Format(time.RFC3339), l, s+1, jobName)

				_, err := stepFile.Write([]byte(line))
				require.NoError(b, err)

				// Add some multi-line entries to simulate real logs
				if l%10 == 0 {
					_, err := stepFile.Write([]byte("  Additional context line 1\n"))
					require.NoError(b, err)

					_, err = stepFile.Write([]byte("  Additional context line 2\n"))
					require.NoError(b, err)
				}
			}
		}
	}

	err := zw.Close()
	require.NoError(b, err)

	return buf.Bytes()
}

// createBenchmarkGitHubServer creates a test server serving custom ZIP data (extends existing pattern)
func createBenchmarkGitHubServer(b *testing.B, zipData []byte) *httptest.Server {
	b.Helper()
	var server *httptest.Server

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GitHub API redirects to logs location (matches existing pattern)
		if strings.HasSuffix(r.URL.Path, "/logs") {
			w.Header().Set("Location", server.URL+"/fetch")
			w.WriteHeader(http.StatusFound)

			return
		}

		// Serve custom ZIP data
		if r.URL.Path == "/fetch" {
			w.Header().Set("Content-Type", "application/zip")
			if _, err := w.Write(zipData); err != nil {
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	server = httptest.NewServer(handler)
	return server
}

// setupTestGitHubClient creates a GitHub client configured for testing
func setupTestGitHubClient(cfg *Config) *github.Client {
	client := github.NewClient(nil)
	if cfg.GitHubAPIConfig.BaseURL != "" && cfg.GitHubAPIConfig.UploadURL != "" {
		client, _ = client.WithEnterpriseURLs(cfg.GitHubAPIConfig.BaseURL, cfg.GitHubAPIConfig.UploadURL)
	}
	if cfg.GitHubAPIConfig.Auth.Token != "" {
		client = client.WithAuthToken(cfg.GitHubAPIConfig.Auth.Token)
	}
	return client
}

// BenchmarkEventToLogs benchmarks log processing with different ZIP sizes
func BenchmarkEventToLogs(b *testing.B) {
	zipSizes := []struct {
		name         string
		jobs         int
		stepsPerJob  int
		linesPerStep int
	}{
		{"small", 1, 5, 100},
		{"medium", 5, 10, 500},
		{"large", 10, 20, 1000},
		{"xlarge", 20, 20, 2000},
	}

	// Load a workflow_run event payload
	workflowRunPayload, err := os.ReadFile("./testdata/completed/8_workflow_run_completed.json")
	require.NoError(b, err)

	event, err := github.ParseWebHook("workflow_run", workflowRunPayload)
	require.NoError(b, err)

	for _, size := range zipSizes {
		b.Run(size.name, func(b *testing.B) {
			zipData := createBenchmarkZip(b, size.jobs, size.stepsPerJob, size.linesPerStep)
			ghServer := createBenchmarkGitHubServer(b, zipData)

			cfg := createDefaultConfig().(*Config)
			cfg.GitHubAPIConfig.BaseURL = ghServer.URL
			cfg.GitHubAPIConfig.UploadURL = ghServer.URL
			cfg.GitHubAPIConfig.Auth.Token = "testtoken"

			ghClient := setupTestGitHubClient(cfg)
			b.Cleanup(func() {
				ghServer.Close()
			})
			logger := zap.NewNop()

			// Benchmark
			b.ReportAllocs()
			for b.Loop() {
				_, _ = eventToLogs(event, cfg, ghClient, logger, true)
			}
		})
	}
}
