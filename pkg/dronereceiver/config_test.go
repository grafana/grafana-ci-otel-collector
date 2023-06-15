package dronereceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	t.Run("Fails when Drone host is not defined", func(t *testing.T) {
		cfg := Config{
			DroneConfig: DroneConfig{},
		}

		assert.Error(t, cfg.Validate())
	})

	t.Run("Fails when Drone token is not defined", func(t *testing.T) {
		cfg := Config{
			DroneConfig: DroneConfig{
				Host: "http://localhost:8080",
			},
		}

		assert.Error(t, cfg.Validate())
	})

	t.Run("Succeeds when both Drone host and token are defined", func(t *testing.T) {
		cfg := Config{
			DroneConfig: DroneConfig{
				Host:  "http://localhost:8080",
				Token: "token",
			},
		}

		assert.NoError(t, cfg.Validate())
	})
}
