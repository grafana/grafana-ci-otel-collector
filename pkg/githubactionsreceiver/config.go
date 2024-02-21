package githubactionsreceiver

import (
	"fmt"

	"github.com/grafana/grafana-ci-otel-collector/githubactionsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

type WebhookConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Port     int    `mapstructure:"port"`
	Secret   string `mapstructure:"secret"`
}

// Config defines configuration for githubactionsreceiver receiver.
type Config struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	metadata.MetricsBuilderConfig           `mapstructure:",squash"`
	WebhookConfig                           WebhookConfig `mapstructure:"webhook"`

	Token string `mapstructure:"token"`
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	// the token should be required only if logs are enabled, however there is no way to check
	// if a log pipeline with this receiver is configured at this stage.
	if cfg.Token == "" {
		return fmt.Errorf("token must be defined")
	}

	return nil
}
