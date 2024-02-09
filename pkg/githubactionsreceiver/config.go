package githubactionsreceiver

import (
	"github.com/grafana/grafana-ci-otel-collector/githubactionsreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

type DBConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"db"`
	Host     string `mapstructure:"host"`
}

type WebhookConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Port     int    `mapstructure:"port"`
}

// Config defines configuration for githubactionsreceiver receiver.
type Config struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	metadata.MetricsBuilderConfig           `mapstructure:",squash"`
	WebhookConfig                           WebhookConfig `mapstructure:"webhook"`
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {

	return nil
}
