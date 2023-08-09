// Code generated by mdatagen. DO NOT EDIT.

package metadata

import "go.opentelemetry.io/collector/confmap"

// MetricConfig provides common config for a particular metric.
type MetricConfig struct {
	Enabled bool `mapstructure:"enabled"`

	enabledSetByUser bool
}

func (ms *MetricConfig) Unmarshal(parser *confmap.Conf) error {
	if parser == nil {
		return nil
	}
	err := parser.Unmarshal(ms, confmap.WithErrorUnused())
	if err != nil {
		return err
	}
	ms.enabledSetByUser = parser.IsSet("enabled")
	return nil
}

// MetricsConfig provides config for dronereceiver metrics.
type MetricsConfig struct {
	BuildsNumber  MetricConfig `mapstructure:"builds_number"`
	RepoInfo      MetricConfig `mapstructure:"repo_info"`
	RestartsTotal MetricConfig `mapstructure:"restarts_total"`
}

func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		BuildsNumber: MetricConfig{
			Enabled: true,
		},
		RepoInfo: MetricConfig{
			Enabled: true,
		},
		RestartsTotal: MetricConfig{
			Enabled: true,
		},
	}
}

// MetricsBuilderConfig is a configuration for dronereceiver metrics builder.
type MetricsBuilderConfig struct {
	Metrics MetricsConfig `mapstructure:"metrics"`
}

func DefaultMetricsBuilderConfig() MetricsBuilderConfig {
	return MetricsBuilderConfig{
		Metrics: DefaultMetricsConfig(),
	}
}
