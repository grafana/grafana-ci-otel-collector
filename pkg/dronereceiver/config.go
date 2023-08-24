package dronereceiver

import (
	"fmt"

	"github.com/grafana/grafana-collector/dronereceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

type DroneConfig struct {
	Token string `mapstructure:"token"`
	Host  string `mapstructure:"host"`
}

type WebhookConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Port     int    `mapstructure:"port"`
	Secret   string `mapstructure:"secret"`
}

// Config defines configuration for dronereceiver receiver.
type Config struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	metadata.MetricsBuilderConfig           `mapstructure:",squash"`
	WebhookConfig                           WebhookConfig `mapstructure:"webhook"`
	DroneConfig                             DroneConfig   `mapstructure:"drone"`
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.DroneConfig.Host == "" {
		return fmt.Errorf("host must be defined")
	}
	if cfg.DroneConfig.Token == "" {
		return fmt.Errorf("token must be defined")
	}
	if cfg.WebhookConfig.Secret == "" {
		return fmt.Errorf("webhook secret must be defined")
	}
	return nil
}
