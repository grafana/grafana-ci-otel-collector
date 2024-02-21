package githubactionsreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {

	t.Run("Configvalidation", func(t *testing.T) {
		t.Run("Fails when token is not defined", func(t *testing.T) {
			cfg := Config{}

			assert.Error(t, cfg.Validate())
		})

	})

}
