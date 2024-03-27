package dronereceiver

import (
	"fmt"

	"github.com/grafana/grafana-ci-otel-collector/dronereceiver/internal/metadata"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

type DBConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"db"`
	Host     string `mapstructure:"host"`
}

type DroneConfig struct {
	Token    string   `mapstructure:"token"`
	Host     string   `mapstructure:"host"`
	Database DBConfig `mapstructure:"database"`
}

// Config defines configuration for dronereceiver receiver.
type Config struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	metadata.MetricsBuilderConfig           `mapstructure:",squash"`
	confighttp.ServerConfig                 `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
	Path                                    string                   `mapstructure:"path"`   // path for data collection. Default is <host>:<port>/events
	Secret                                  string                   `mapstructure:"secret"` // webhook hash signature. Default is empty
	DroneConfig                             DroneConfig              `mapstructure:"drone"`
	ReposConfig                             map[string][]string      `mapstructure:"repos"`
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	if cfg.DroneConfig.Host == "" {
		return fmt.Errorf("host must be defined")
	}
	if cfg.DroneConfig.Token == "" {
		return fmt.Errorf("token must be defined")
	}
	if cfg.Secret == "" {
		return fmt.Errorf("webhook secret must be defined")
	}

	// Validates that the repos and branches are defined.
	// At least one repo and one branch must be defined.
	// branches may appear only once per repo.
	if cfg.ReposConfig == nil || len(cfg.ReposConfig) == 0 {
		return fmt.Errorf("repos must be defined")
	}

	for repo, branches := range cfg.ReposConfig {
		if len(branches) == 0 {
			return fmt.Errorf("at least one branch must be defined for repo %s", repo)
		}

		branchMap := make(map[string]bool)
		for _, branch := range branches {
			if branchMap[branch] {
				return fmt.Errorf("branch %s is duplicated for repo %s", branch, repo)
			}
			branchMap[branch] = true
		}
	}

	return nil
}
