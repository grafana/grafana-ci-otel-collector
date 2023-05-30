package dronereceiver

import (
	"fmt"
)

// Config defines configuration for dronereceiver receiver.
type Config struct {
	Token string `mapstructure:"token"`
	Host  string `mapstructure:"host"`
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Host == "" {
		return fmt.Errorf("host must be defined")
	}
	if cfg.Token == "" {
		return fmt.Errorf("token must be defined")
	}
	return nil
}
