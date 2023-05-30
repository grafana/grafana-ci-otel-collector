package dronereceiver

import (
	"context"

	drone "github.com/drone/drone-go/drone"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"golang.org/x/oauth2"
)

type droneReceiver struct {
	cfg         *Config
	set         receiver.CreateSettings
	droneClient drone.Client
}

func newDroneReceiver(cfg *Config, set receiver.CreateSettings) (*droneReceiver, error) {
	set.Logger.Info("Creating the drone receiver")

	oauthConfig := new(oauth2.Config)
	httpClient := oauthConfig.Client(
		context.Background(),
		&oauth2.Token{
			AccessToken: cfg.Token,
		},
	)
	droneClient := drone.NewClient(cfg.Host, httpClient)

	receiver := &droneReceiver{
		cfg:         cfg,
		set:         set,
		droneClient: droneClient,
	}

	return receiver, nil
}

func (r *droneReceiver) Start(_ context.Context, host component.Host) error {
	// TODO: start an HTTP server to receive webhook events from drone
	r.set.Logger.Info("Starting the drone receiver")
	return nil
}

func (r *droneReceiver) Shutdown(_ context.Context) error {
	// TODO: implement
	return nil
}

func (r *droneReceiver) registerTraceConsumer(consumer consumer.Traces) error {
	// TODO: implement
	return nil
}

func (r *droneReceiver) registerLogsConsumer(consumer consumer.Logs) error {
	// TODO: implement
	return nil
}

func (r *droneReceiver) registerMetricsConsumer(consumer consumer.Metrics) error {
	// TODO: implement
	return nil
}
