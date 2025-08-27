// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
// bump

package githubactionsreceiver // import "github.com/grafana/grafana-ci-otel-collector/receiver/githubactionsreceiver"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver"

	"github.com/grafana/grafana-ci-otel-collector/internal/sharedcomponent"
	"github.com/grafana/grafana-ci-otel-collector/receiver/githubactionsreceiver/internal/metadata"
)

// This file implements factory for GitHub Actions receiver.

const (
	defaultBindEndpoint = "0.0.0.0:19418"
	defaultPath         = "/ghaevents"
)

// NewFactory creates a new GitHub Actions receiver factory
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(newTracesReceiver, metadata.TracesStability),
		receiver.WithLogs(newLogsReceiver, metadata.LogsStability),
		receiver.WithMetrics(newMetricsReceiver, metadata.MetricsStability),
	)
}

// createDefaultConfig creates the default configuration for GitHub Actions receiver.
func createDefaultConfig() component.Config {
	return &Config{
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		ServerConfig: confighttp.ServerConfig{
			Endpoint: defaultBindEndpoint,
		},
		Path:   defaultPath,
		Secret: "",
	}
}

// This is the map of already created githubactions receivers for particular configurations.
// We maintain this map because the Factory is asked log and metric receivers separately
// when it gets CreateLogsReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one receiver object per configuration.
var receivers = sharedcomponent.NewSharedComponents()
