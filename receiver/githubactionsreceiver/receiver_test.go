// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubactionsreceiver

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-github/v75/github"
	"github.com/grafana/grafana-ci-otel-collector/internal/sharedcomponent"
	"github.com/grafana/grafana-ci-otel-collector/receiver/githubactionsreceiver/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestNewReceiver(t *testing.T) {
	defaultConfig := createDefaultConfig().(*Config)

	tests := []struct {
		desc     string
		config   Config
		consumer consumer.Logs
		err      error
	}{
		{
			desc:     "Default config succeeds",
			config:   *defaultConfig,
			consumer: consumertest.NewNop(),
			err:      nil,
		},
		{
			desc: "User defined config success",
			config: Config{
				ServerConfig: confighttp.ServerConfig{
					Endpoint: "localhost:8080",
				},
				Secret: "mysecret",
			},
			consumer: consumertest.NewNop(),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			rec, err := newReceiver(receivertest.NewNopSettings(receivertest.NopType), &test.config)
			if test.err == nil {
				require.NotNil(t, rec)
			} else {
				require.ErrorIs(t, err, test.err)
				require.Nil(t, rec)
			}
		})
	}
}

func TestEventToTraces(t *testing.T) {
	tests := []struct {
		desc            string
		payloadFilePath string
		eventType       string
		expectedError   error
		expectedSpans   int
	}{
		{
			desc:            "WorkflowJobEvent processing",
			payloadFilePath: "./testdata/completed/5_workflow_job_completed.json",
			eventType:       "workflow_job",
			expectedError:   nil,
			expectedSpans:   10, // 10 spans in the payload
		},
		{
			desc:            "WorkflowRunEvent processing",
			payloadFilePath: "./testdata/completed/8_workflow_run_completed.json",
			eventType:       "workflow_run",
			expectedError:   nil,
			expectedSpans:   1, // Root span
		},
	}

	logger := zaptest.NewLogger(t)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			payload, err := os.ReadFile(test.payloadFilePath)
			require.NoError(t, err)

			event, err := github.ParseWebHook(test.eventType, payload)
			require.NoError(t, err)

			traces, err := eventToTraces(event, &Config{}, logger)

			if test.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, test.expectedError, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, test.expectedSpans, traces.SpanCount(), fmt.Sprintf("%s: unexpected number of spans", test.desc))
		})
	}
}

func TestWorkflowJobEventToMetrics(t *testing.T) {
	tests := []struct {
		desc               string
		payloadFilePath    string
		eventType          string
		expectedMetrics    int
		expectedDataPoints int
	}{
		{
			desc:               "WorkflowJobEvent processing",
			payloadFilePath:    "./testdata/queued/1_workflow_job_queued.json",
			eventType:          "workflow_job",
			expectedMetrics:    1,
			expectedDataPoints: len(metadata.MapAttributeCiGithubWorkflowJobStatus) * len(metadata.MapAttributeCiGithubWorkflowJobConclusion),
		},
		{
			desc:               "WorkflowJobEvent (check run) processing",
			payloadFilePath:    "./testdata/completed/5_workflow_job_check-run_completed.json",
			eventType:          "workflow_job",
			expectedMetrics:    1,
			expectedDataPoints: len(metadata.MapAttributeCiGithubWorkflowJobStatus) * len(metadata.MapAttributeCiGithubWorkflowJobConclusion),
		},
	}

	logger := zaptest.NewLogger(t)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			payload, err := os.ReadFile(test.payloadFilePath)
			require.NoError(t, err)

			event, err := github.ParseWebHook(test.eventType, payload)
			require.NoError(t, err)

			mh := newMetricsHandler(receivertest.NewNopSettings(receivertest.NopType), &Config{
				MetricsBuilderConfig: metadata.MetricsBuilderConfig{
					Metrics: metadata.MetricsConfig{
						WorkflowJobsCount: metadata.MetricConfig{
							Enabled: true,
						},
					},
				},
			}, logger.Named("metricsHandler"))

			metrics := mh.workflowJobEventToMetrics(event.(*github.WorkflowJobEvent))

			require.Equalf(t, test.expectedMetrics, metrics.MetricCount(), "%s: unexpected number of metrics", test.desc)
			require.Equalf(t, test.expectedDataPoints, metrics.DataPointCount(), "%s: unexpected number of datapoints", test.desc)
		})
	}
}

func TestWorkflowRunEventToMetrics(t *testing.T) {
	tests := []struct {
		desc               string
		payloadFilePath    string
		eventType          string
		expectedMetrics    int
		expectedDataPoints int
	}{
		{
			desc:               "WorkflowRunEvent processing",
			payloadFilePath:    "./testdata/completed/8_workflow_run_completed.json",
			eventType:          "workflow_run",
			expectedMetrics:    1,
			expectedDataPoints: len(metadata.MapAttributeCiGithubWorkflowRunStatus) * len(metadata.MapAttributeCiGithubWorkflowRunConclusion),
		},
		{
			desc:               "WorkflowRunEvent processing",
			payloadFilePath:    "./testdata/in_progress/10_workflow_run_in_progress.json",
			eventType:          "workflow_run",
			expectedMetrics:    1,
			expectedDataPoints: len(metadata.MapAttributeCiGithubWorkflowRunStatus) * len(metadata.MapAttributeCiGithubWorkflowRunConclusion),
		},
	}

	logger := zaptest.NewLogger(t)
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			payload, err := os.ReadFile(test.payloadFilePath)
			require.NoError(t, err)

			event, err := github.ParseWebHook(test.eventType, payload)
			require.NoError(t, err)

			mh := newMetricsHandler(receivertest.NewNopSettings(receivertest.NopType), &Config{
				MetricsBuilderConfig: metadata.MetricsBuilderConfig{
					Metrics: metadata.MetricsConfig{
						WorkflowRunsCount: metadata.MetricConfig{
							Enabled: true,
						},
					},
				},
			}, logger.Named("metricsHandler"))

			metrics := mh.workflowRunEventToMetrics(event.(*github.WorkflowRunEvent))

			require.Equalf(t, test.expectedMetrics, metrics.MetricCount(), "%s: unexpected number of metrics", test.desc)
			require.Equalf(t, test.expectedDataPoints, metrics.DataPointCount(), "%s: unexpected number of datapoints", test.desc)
		})
	}
}

func TestProcessSteps(t *testing.T) {
	tests := []struct {
		desc             string
		givenSteps       []*github.TaskStep
		expectedSpans    int
		expectedStatuses []ptrace.StatusCode
	}{
		{
			desc: "Multiple steps with mixed status",

			givenSteps: []*github.TaskStep{
				{Name: getPtr("Checkout"), Status: getPtr("completed"), Conclusion: getPtr("success")},
				{Name: getPtr("Build"), Status: getPtr("completed"), Conclusion: getPtr("failure")},
				{Name: getPtr("Test"), Status: getPtr("completed"), Conclusion: getPtr("success")},
			},
			expectedSpans: 4, // Includes parent span
			expectedStatuses: []ptrace.StatusCode{
				ptrace.StatusCodeOk,
				ptrace.StatusCodeError,
				ptrace.StatusCodeOk,
			},
		},
		{
			desc:             "No steps",
			givenSteps:       []*github.TaskStep{},
			expectedSpans:    1, // Only the parent span should be created
			expectedStatuses: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			logger := zap.NewNop()
			traces := ptrace.NewTraces()
			rs := traces.ResourceSpans().AppendEmpty()
			ss := rs.ScopeSpans().AppendEmpty()

			traceID, _ := generateTraceID(123, 1)
			parentSpanID := createParentSpan(ss, tc.givenSteps, &github.WorkflowJob{}, traceID, logger)

			defaultBranch := "main"
			processSteps(ss, tc.givenSteps, &github.WorkflowJob{}, &defaultBranch, traceID, parentSpanID, logger)

			startIdx := 1 // Skip the parent span if it's the first one
			if len(tc.expectedStatuses) == 0 {
				startIdx = 0 // No steps, only the parent span exists
			}

			require.Equal(t, tc.expectedSpans, ss.Spans().Len(), "Unexpected number of spans")
			for i, expectedStatusCode := range tc.expectedStatuses {
				span := ss.Spans().At(i + startIdx)
				statusCode := span.Status().Code()
				require.Equal(t, expectedStatusCode, statusCode, fmt.Sprintf("Unexpected status code for span #%d", i+startIdx))
			}
		})
	}
}

func TestResourceAndSpanAttributesCreation(t *testing.T) {
	tests := []struct {
		desc            string
		payloadFilePath string
		expectedSteps   []map[string]string
	}{
		{
			desc:            "WorkflowJobEvent Step Attributes",
			payloadFilePath: "./testdata/completed/5_workflow_job_completed.json",
			expectedSteps: []map[string]string{
				{"ci.github.workflow.job.step.name": "Set up job", "ci.github.workflow.job.step.number": "1"},
				{"ci.github.workflow.job.step.name": "Run actions/checkout@v3", "ci.github.workflow.job.step.number": "2"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			logger := zaptest.NewLogger(t)

			payload, err := os.ReadFile(tc.payloadFilePath)
			require.NoError(t, err)

			event, err := github.ParseWebHook("workflow_job", payload)
			require.NoError(t, err)

			traces, err := eventToTraces(event, &Config{}, logger)
			require.NoError(t, err)

			rs := traces.ResourceSpans().At(0)
			ss := rs.ScopeSpans().At(0)

			for _, expectedStep := range tc.expectedSteps {
				stepFound := false

				for i := 0; i < ss.Spans().Len() && !stepFound; i++ {
					span := ss.Spans().At(i)
					attrs := span.Attributes()

					stepValue, found := attrs.Get("ci.github.workflow.job.step.name")
					stepName := stepValue.Str()

					if !found || stepName == "" { // Skip if the attribute is not found or name is empty
						continue
					}

					isMainValue, found := attrs.Get("ci.github.workflow.job.head_branch.is_main")
					if !found || isMainValue.AsString() == "" { // Skip if the attribute is not found or name is empty
						continue
					}

					require.True(t, isMainValue.Bool())

					expectedStepName := expectedStep["ci.github.workflow.job.step.name"]

					if stepName == expectedStepName {
						stepFound = true
						for attrKey, expectedValue := range expectedStep {
							attrValue, found := attrs.Get(attrKey)
							if !found {
								require.Fail(t, fmt.Sprintf("Attribute '%s' not found in span for step '%s'", attrKey, stepName))
								continue
							}
							actualValue := attributeValueToString(attrValue)
							require.Equal(t, expectedValue, actualValue, "Attribute '%s' does not match expected value for step '%s'", attrKey, stepName)
						}
					}
				}

				require.True(t, stepFound, "Step '%s' not found in any span", expectedStep["ci.github.workflow.job.step.name"])
			}

		})
	}
}

func TestReceiverWithAppAndEnterprise(t *testing.T) {
	logsSink := new(consumertest.LogsSink)
	ghTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	t.Cleanup(func() {
		ghTestServer.Close()
	})

	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = "localhost:0" // Let OS choose port

	tmpDir := t.TempDir()
	pkPath := filepath.Join(tmpDir, "private-key.dat")
	require.NoError(t, os.WriteFile(pkPath, []byte(`-----BEGIN PRIVATE KEY-----
MIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEA2VSTKXeFpfXPIzsb
xdaegmusCFe+5IawGOdSJ7/Ca7/7it4FnT9QfLwLNdR2qrNLdOzfsYHSZjkFXaIs
cGPDvwIDAQABAkBqnhIf+rHHLCL1Pq8uTE6w3s+jvCA7DlRfs0PbmjhwEPQFBEOH
zOluI3xhI3E0cJXhLJQ2ydtxC+tq1A2Kz+eRAiEA/J8HB8qvB/51c7gg+mkw70lx
fqD+ro9M1rOeBcuukycCIQDcPLZA699lYWurLViBAqnX2iZqvm9cB/np7S76V23J
qQIhANq6NrQgYfxh7gAL5UHr4lrNFF+3tcwed0FOs/wAp17xAiAWoS5g8VudASud
BSXI68sj4Mh9w1+R50fon3RqSL2BMQIgI/blZZM+Hf1YHbDY8KfrKuLUtoiz6ePQ
jQgTMp1cZEM=
-----END PRIVATE KEY-----
`), 0644))

	cfg.GitHubAPIConfig.Auth.AppID = 123
	cfg.GitHubAPIConfig.Auth.InstallationID = 123
	cfg.GitHubAPIConfig.Auth.PrivateKeyPath = pkPath
	cfg.GitHubAPIConfig.BaseURL = ghTestServer.URL
	cfg.GitHubAPIConfig.UploadURL = ghTestServer.URL

	// Create receiver with test consumers
	recv, err := newLogsReceiver(
		context.Background(),
		receivertest.NewNopSettings(receivertest.NopType),
		cfg,
		logsSink,
	)
	require.NoError(t, err)
	sharedComp, ok := recv.(*sharedcomponent.SharedComponent)
	require.True(t, ok, "Receiver must be a shared component")

	rcvr, ok := sharedComp.Unwrap().(*githubActionsReceiver)
	require.True(t, ok, "Unwrapped component must be githubActionsReceiver")

	require.Equal(t, ghTestServer.URL+"/api/v3/", rcvr.ghitr.BaseURL)
}

func TestLogsReceiverEndToEnd(t *testing.T) {
	// Setup test secrets
	testSecret := "testsecret123"
	validSig := func(payload []byte) string {
		mac := hmac.New(sha256.New, []byte(testSecret))
		mac.Write(payload)
		return "sha256=" + hex.EncodeToString(mac.Sum(nil))
	}

	// Create mock GitHub API server for logs download
	var ghTestServer *httptest.Server

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The calling code expects to be redirected to the logs location.
		if strings.HasSuffix(r.URL.Path, "/logs") {
			w.Header().Set("Location", ghTestServer.URL+"/fetch")
			w.WriteHeader(http.StatusFound)
			return
		}
		if r.URL.Path == "/fetch" {
			createTestZip(t, w)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	ghTestServer = httptest.NewServer(handler)
	defer ghTestServer.Close()

	tests := []struct {
		name            string
		payloadFile     string
		eventType       string
		secret          string
		wantStatus      int
		wantLogCount    int
		withTraceInfo   bool
		ghClientEnabled bool
	}{
		{
			name:            "ValidWorkflowRunEventWithLogs",
			payloadFile:     "./testdata/completed/8_workflow_run_completed.json",
			eventType:       "workflow_run",
			secret:          testSecret,
			wantStatus:      http.StatusAccepted,
			wantLogCount:    4, // Expecting logs from test zip
			withTraceInfo:   true,
			ghClientEnabled: true,
		},
		{
			name:            "NonCompletedEvent",
			payloadFile:     "./testdata/in_progress/10_workflow_run_in_progress.json",
			eventType:       "workflow_run",
			secret:          testSecret,
			wantStatus:      http.StatusNoContent,
			wantLogCount:    0,
			ghClientEnabled: true,
		},
		{
			name:            "MissingGitHubClient",
			payloadFile:     "./testdata/completed/8_workflow_run_completed.json",
			eventType:       "workflow_run",
			secret:          testSecret,
			wantStatus:      http.StatusAccepted,
			wantLogCount:    0, // Shouldn't process logs without client
			ghClientEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test receiver
			logsSink := new(consumertest.LogsSink)
			tracesSink := new(consumertest.TracesSink)

			cfg := createDefaultConfig().(*Config)
			cfg.Secret = tt.secret
			cfg.Endpoint = "localhost:0" // Let OS choose port

			if tt.ghClientEnabled {
				cfg.GitHubAPIConfig.Auth.Token = "testtoken"
				cfg.GitHubAPIConfig.BaseURL = ghTestServer.URL
				cfg.GitHubAPIConfig.UploadURL = ghTestServer.URL
			}

			// Create receiver with test consumers
			recv, err := newLogsReceiver(
				context.Background(),
				receivertest.NewNopSettings(receivertest.NopType),
				cfg,
				logsSink,
			)
			require.NoError(t, err)

			// Unwrap the sharedcomponent to get accesss to receiver fields
			sharedComp, ok := recv.(*sharedcomponent.SharedComponent)
			require.True(t, ok, "Receiver must be a shared component")

			rcvr, ok := sharedComp.Unwrap().(*githubActionsReceiver)
			require.True(t, ok, "Unwrapped component must be githubActionsReceiver")

			rcvr.tracesConsumer = tracesSink

			// Start receiver
			err = rcvr.Start(context.Background(), componenttest.NewNopHost())
			require.NoError(t, err)
			defer func() {
				err := rcvr.Shutdown(context.Background())
				require.NoError(t, err)
			}()

			// Load test payload
			payload, err := os.ReadFile(tt.payloadFile)
			require.NoError(t, err)

			// Create request
			req := httptest.NewRequest("POST", "/ghaevents", bytes.NewReader(payload))
			req.Header.Set("X-GitHub-Event", tt.eventType)
			req.Header.Set("X-Hub-Signature-256", validSig(payload))
			req.Header.Set("Content-Type", "application/json")

			// Send request to receiver
			w := httptest.NewRecorder()
			rcvr.ServeHTTP(w, req)

			// Verify HTTP response
			require.Equal(t, tt.wantStatus, w.Code)

			// Verify logs consumer
			allLogs := logsSink.AllLogs()
			if tt.wantLogCount == 0 {
				require.Empty(t, allLogs)
				return
			}

			require.Len(t, allLogs, 1)
			logs := allLogs[0]

			// Verify log contents
			resourceLogs := logs.ResourceLogs()
			require.Greater(t, resourceLogs.Len(), 0)

			scopeLogs := resourceLogs.At(0).ScopeLogs()
			require.Greater(t, scopeLogs.Len(), 0)

			logRecords := scopeLogs.At(0).LogRecords()
			require.Equal(t, tt.wantLogCount, logRecords.Len())

			require.True(t, strings.Contains(logRecords.At(2).Body().Str(), "Step 2 started"))
			require.True(t, strings.Contains(logRecords.At(2).Body().Str(), "some additional information about step 2."))

			// Verify trace information if expected
			if tt.withTraceInfo {
				for i := range logRecords.Len() {
					record := logRecords.At(i)
					require.False(t, record.TraceID().IsEmpty())
					require.False(t, record.SpanID().IsEmpty())
				}
			}

			// Verify job attributes
			attrs := resourceLogs.At(0).Resource().Attributes()
			require.Equal(t, "foo/webhook-testing", attrs.AsRaw()["scm.git.repo"])
		})
	}
}

func createTestZip(t *testing.T, w io.Writer) {
	t.Helper()
	zw := zip.NewWriter(w)
	defer func() {
		require.NoError(t, zw.Close())
	}()

	// Create test job directory
	_, err := zw.Create("test-job/")
	require.NoError(nil, err)

	// Create step log files
	steps := []struct {
		number int
		lines  []string
	}{
		{
			number: 1,
			lines: []string{
				"2023-01-01T12:00:00Z Step 1 started",
				"2023-01-01T12:00:05Z Step 1 completed",
			},
		},
		{
			number: 2,
			lines: []string{
				"2023-01-01T12:00:10Z Step 2 started",
				"some additional information about step 2. this should be rolled into the previous log line.",
				"2023-01-01T12:00:15Z Step 2 completed",
			},
		},
	}

	for _, step := range steps {
		f, err := zw.Create(fmt.Sprintf("test-job/%d_step.log", step.number))
		require.NoError(nil, err)

		for _, line := range step.lines {
			_, err := f.Write([]byte(line + "\n"))
			require.NoError(nil, err)
		}
	}
}

// attributeValueToString converts an attribute value to a string regardless of its actual type
func attributeValueToString(attr pcommon.Value) string {
	switch attr.Type() {
	case pcommon.ValueTypeStr:
		return attr.Str()
	case pcommon.ValueTypeInt:
		return strconv.FormatInt(attr.Int(), 10)
	case pcommon.ValueTypeDouble:
		return strconv.FormatFloat(attr.Double(), 'f', -1, 64)
	case pcommon.ValueTypeBool:
		return strconv.FormatBool(attr.Bool())
	case pcommon.ValueTypeMap:
		return "<Map Value>"
	case pcommon.ValueTypeSlice:
		return "<Slice Value>"
	default:
		return "<Unknown Value Type>"
	}
}

func getPtr(str string) *string {
	return &str
}
