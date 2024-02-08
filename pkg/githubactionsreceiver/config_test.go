package githubactionsreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {

	t.Run("DroneConfig validation", func(t *testing.T) {
		t.Run("Fails when Drone host is not defined", func(t *testing.T) {
			cfg := Config{
				WebhookConfig: WebhookConfig{
					Secret: "secret",
				},
				DroneConfig: DroneConfig{
					Token: "token",
				},
				ReposConfig: map[string][]string{
					"repo1": {"branch1", "branch2"},
				},
			}

			assert.Error(t, cfg.Validate())
		})

		t.Run("Fails when Drone token is not defined", func(t *testing.T) {
			cfg := Config{
				WebhookConfig: WebhookConfig{
					Secret: "secret",
				},
				DroneConfig: DroneConfig{
					Host: "http://localhost:8080",
				},
				ReposConfig: map[string][]string{
					"repo1": {"branch1", "branch2"},
				},
			}

			assert.Error(t, cfg.Validate())
		})
	})

	t.Run("WebhookConfig validation", func(t *testing.T) {
		t.Run("Fails when Secret is not defined", func(t *testing.T) {
			cfg := Config{
				DroneConfig: DroneConfig{
					Token: "token",
					Host:  "http://localhost:8080",
				},
				ReposConfig: map[string][]string{
					"repo1": {"branch1", "branch2"},
				},
			}

			assert.Error(t, cfg.Validate())
		})
	})

	t.Run("ReposConfig validation", func(t *testing.T) {
		t.Run("Fails when no repo is  defined", func(t *testing.T) {
			cfg := Config{
				WebhookConfig: WebhookConfig{
					Secret: "secret",
				},
				DroneConfig: DroneConfig{
					Token: "token",
					Host:  "http://localhost:8080",
				},
			}

			assert.Error(t, cfg.Validate())
		})

		t.Run("Fails when a repo contains no branches", func(t *testing.T) {
			cfg := Config{
				WebhookConfig: WebhookConfig{
					Secret: "secret",
				},
				DroneConfig: DroneConfig{
					Token: "token",
					Host:  "http://localhost:8080",
				},
				ReposConfig: map[string][]string{
					"repo1": {},
				},
			}

			assert.Error(t, cfg.Validate())
		})

		t.Run("Fails when a repo contains duplicated branches", func(t *testing.T) {
			cfg := Config{
				WebhookConfig: WebhookConfig{
					Secret: "secret",
				},
				DroneConfig: DroneConfig{
					Token: "token",
					Host:  "http://localhost:8080",
				},
				ReposConfig: map[string][]string{
					"repo1": {"branch1", "branch1"},
				},
			}

			assert.Error(t, cfg.Validate())
		})
	})

	t.Run("Succeeds when all required properties are defined", func(t *testing.T) {
		cfg := Config{
			DroneConfig: DroneConfig{
				Host:  "http://localhost:8080",
				Token: "token",
			},
			WebhookConfig: WebhookConfig{
				Secret: "secret",
			},
			ReposConfig: map[string][]string{
				"repo1": {"branch1", "branch2"},
			},
		}

		assert.NoError(t, cfg.Validate())
	})
}
