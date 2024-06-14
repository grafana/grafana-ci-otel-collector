package dronereceiver

import (
	"testing"

	"github.com/drone/drone-go/drone"
	"github.com/grafana/grafana-ci-otel-collector/receiver/dronereceiver/internal/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestHandleEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)

	t.Run("Repo & Branch enabled", func(t *testing.T) {
		event := WebhookEvent{
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

		droneMockClient := new(mocks.MockDroneClient)

		config := createDefaultConfig().(*Config)

		config.ReposConfig = map[string][]string{
			"repoA": {"main"},
		}
		// Set up expectations for the Logs() method
		droneMockClient.On("Logs", "", "", 0, 0, 1).Return([]*drone.Line{
			{Number: 1, Message: "message", Timestamp: 123456},
		}, nil)

		traces, logs := handleEvent(event, config, droneMockClient, logger)

		assert.NotNil(t, traces)
		assert.Equal(t, 3, traces.SpanCount())

		assert.NotNil(t, logs)
		assert.Equal(t, 1, logs.ResourceLogs().Len())

	})

	t.Run("Repo not enabled", func(t *testing.T) {
		event := WebhookEvent{
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

		droneMockClient := new(mocks.MockDroneClient)

		config := createDefaultConfig().(*Config)

		// Set up expectations for the Logs() method
		droneMockClient.On("Logs", "", "", 0, 0, 1).Return([]*drone.Line{
			{Number: 1, Message: "message", Timestamp: 123456},
		}, nil)
		traces, logs := handleEvent(event, config, droneMockClient, logger)

		assert.Nil(t, traces)
		assert.Nil(t, logs)
	})

	t.Run("Branch not enabled", func(t *testing.T) {
		event := WebhookEvent{
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

		droneMockClient := new(mocks.MockDroneClient)

		config := createDefaultConfig().(*Config)
		config.ReposConfig = map[string][]string{
			"repoA": {"master"},
		}

		// Set up expectations for the Logs() method
		droneMockClient.On("Logs", "", "", 0, 0, 1).Return([]*drone.Line{
			{Number: 1, Message: "message", Timestamp: 123456},
		}, nil)
		traces, logs := handleEvent(event, config, droneMockClient, logger)

		assert.Nil(t, traces)
		assert.Nil(t, logs)
	})
}
