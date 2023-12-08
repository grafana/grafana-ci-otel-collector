package dronereceiver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drone/drone-go/drone"
	"github.com/grafana/grafana-ci-otel-collector/dronereceiver/internal/mocks"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestHandler_ValidWebhookEvent(t *testing.T) {
	validEvent := WebhookEvent{
		"push",
		&RepoEvt{
			drone.Repo{
				ID:     1,
				Slug:   "repoA",
				Branch: "main",
			},
			&drone.Build{ID: 2, Finished: 12345678, Stages: []*drone.Stage{
				{
					ID:   1,
					Name: "stageA",
					Steps: []*drone.Step{
						{
							ID:      1,
							Number:  1,
							Name:    "stepA",
							Status:  drone.StatusPassing,
							Started: 1000,
							Stopped: 1001,
						},
					}},
			}},
		},
		drone.System{Host: "host"},
	}

	// Convert the event to JSON
	payload, err := json.Marshal(validEvent)
	assert.NoError(t, err)

	// Create a mock HTTP request with the JSON payload
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))

	// Create a mock HTTP response
	resp := httptest.NewRecorder()

	core, _ := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	droneMockClient := new(mocks.MockDroneClient)

	cp, err := consumer.NewLogs(func(context.Context, plog.Logs) error { return nil })
	assert.NoError(t, err)
	assert.NoError(t, cp.ConsumeLogs(context.Background(), plog.NewLogs()))

	ct, err := consumer.NewTraces(func(context.Context, ptrace.Traces) error { return nil })
	assert.NoError(t, err)
	assert.NoError(t, ct.ConsumeTraces(context.Background(), ptrace.NewTraces()))

	handler := &droneWebhookHandler{
		droneClient: droneMockClient,
		logger:      logger,
		reposConfig: map[string][]string{
			"repoA": {"main"},
		},
		nextLogsConsumer:  cp,
		nextTraceConsumer: ct,
	}

	// Set up expectations for the Logs() method
	droneMockClient.On("Logs", "", "", 0, 0, 1).Return([]*drone.Line{
		{1, "message", 123456},
	}, nil)

	err = handler.handler(resp, req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NotNil(t, handler.nextLogsConsumer)
	assert.NotNil(t, handler.nextTraceConsumer)
}

func TestHandler_InvalidWebhookEvent_BuildIsNil(t *testing.T) {
	validEvent := WebhookEvent{
		"push",
		&RepoEvt{
			drone.Repo{
				ID:     1,
				Slug:   "repoA",
				Branch: "main",
			},
			nil,
		},
		drone.System{Host: "host"},
	}

	// Convert the event to JSON
	payload, err := json.Marshal(validEvent)
	assert.NoError(t, err)

	// Create a mock HTTP request with the JSON payload
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))

	// Create a mock HTTP response
	resp := httptest.NewRecorder()

	core, _ := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	droneMockClient := new(mocks.MockDroneClient)

	handler := &droneWebhookHandler{
		droneClient: droneMockClient,
		logger:      logger,
		reposConfig: map[string][]string{
			"repoA": {"main"},
		},
	}

	err = handler.handler(resp, req)
	assert.Error(t, err)
}

func TestHandler_ValidWebhookEvent_RepoNotEnabled_CheckLogger(t *testing.T) {
	validEvent := WebhookEvent{
		"push",
		&RepoEvt{
			drone.Repo{
				ID:     1,
				Slug:   "repoA",
				Branch: "main",
			},
			&drone.Build{ID: 2, Finished: 12345678, Stages: []*drone.Stage{
				{
					ID:   1,
					Name: "stageA",
					Steps: []*drone.Step{
						{
							ID:      1,
							Number:  1,
							Name:    "stepA",
							Status:  drone.StatusPassing,
							Started: 1000,
							Stopped: 1001,
						},
					}},
			}},
		},
		drone.System{Host: "host"},
	}

	// Convert the event to JSON
	payload, err := json.Marshal(validEvent)
	assert.NoError(t, err)

	// Create a mock HTTP request with the JSON payload
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))

	// Create a mock HTTP response
	resp := httptest.NewRecorder()

	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	droneMockClient := new(mocks.MockDroneClient)

	handler := &droneWebhookHandler{
		droneClient: droneMockClient,
		logger:      logger,
		reposConfig: map[string][]string{
			"repoB": {"master"},
		},
	}

	err = handler.handler(resp, req)
	entry := logs.All()[0]
	if entry.Level != zap.WarnLevel || entry.Message != "repo not enabled" {
		t.Errorf("Invalid log entry %v", entry)
	}
}

func TestHandler_ValidWebhookEvent_BranchNotConfigured_CheckLogger(t *testing.T) {
	validEvent := WebhookEvent{
		"push",
		&RepoEvt{
			drone.Repo{
				ID:     1,
				Slug:   "repoA",
				Branch: "main",
			},
			&drone.Build{ID: 2, Finished: 12345678, Stages: []*drone.Stage{
				{
					ID:   1,
					Name: "stageA",
					Steps: []*drone.Step{
						{
							ID:      1,
							Number:  1,
							Name:    "stepA",
							Status:  drone.StatusPassing,
							Started: 1000,
							Stopped: 1001,
						},
					}},
			}},
		},
		drone.System{Host: "host"},
	}

	// Convert the event to JSON
	payload, err := json.Marshal(validEvent)
	assert.NoError(t, err)

	// Create a mock HTTP request with the JSON payload
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))

	// Create a mock HTTP response
	resp := httptest.NewRecorder()

	core, logs := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	droneMockClient := new(mocks.MockDroneClient)

	handler := &droneWebhookHandler{
		droneClient: droneMockClient,
		logger:      logger,
		reposConfig: map[string][]string{
			"repoA": {"master"},
		},
	}

	err = handler.handler(resp, req)
	entry := logs.All()[0]
	if entry.Level != zap.WarnLevel || entry.Message != "branch not enabled" {
		t.Errorf("Invalid log entry %v", entry)
	}
}
