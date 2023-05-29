package dronetracereceiver

import (
	"context"
	"time"

	"github.com/drone/drone-go/drone"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"golang.org/x/oauth2"
)

const (
	typeStr         = "dronetracereceiver"
	defaultInterval = 1 * time.Minute
)

func createDefaultConfig() component.Config {
	return &Config{
		Interval: string(defaultInterval),
	}
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithTraces(createTracesReceiver, component.StabilityLevelAlpha))
}

func createTracesReceiver(_ context.Context, params receiver.CreateSettings, baseCfg component.Config, consumer consumer.Traces) (receiver.Traces, error) {
	if consumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	logger := params.Logger
	config := baseCfg.(*Config)

	oauthConfig := new(oauth2.Config)
	httpClient := oauthConfig.Client(
		context.Background(),
		&oauth2.Token{
			AccessToken: config.Token,
		},
	)
	drone := drone.NewClient(config.Host, httpClient)

	traceRcvr := &dronereceiverReceiver{
		logger:       logger,
		nextConsumer: consumer,
		config:       config,
		drone:        drone,
	}

	return traceRcvr, nil
}
