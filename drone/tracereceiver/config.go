package dronetracereceiver

import (
	"fmt"
	"time"
)

// Config defines configuration for dronereceiver receiver.
type Config struct {
	Interval string `mapstructure:"interval"`
	Token    string `mapstructure:"token"`
	Host     string `mapstructure:"host"`
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	interval, _ := time.ParseDuration(cfg.Interval)
	if interval.Minutes() < 1 {
		return fmt.Errorf("when defined, the interval has to be set to at least 1 minute (1m)")
	}

	if cfg.Host == "" {
		return fmt.Errorf("host must be defined")
	}
	if cfg.Token == "" {
		return fmt.Errorf("token must be defined")
	}
	return nil
}
