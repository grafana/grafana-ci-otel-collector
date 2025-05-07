package dronereceiver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/httpsignatures-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestVerifySignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc      string
		signKey   string
		verifyKey string
		err       error
		status    int
	}{
		{
			desc:      "Valid signature",
			signKey:   "TestKey",
			verifyKey: "TestKey",
			status:    http.StatusOK,
		},
		{
			desc:      "Invalid signature",
			signKey:   "TestKey",
			verifyKey: "badKey",
			err:       errSignatureNotValid,
			status:    http.StatusForbidden,
		},
		{
			desc:      "Request not signed",
			verifyKey: "TestKey",
			err:       errParsingSignature,
			status:    http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			r := &http.Request{
				Header: http.Header{
					"Date": []string{"Thu, 08 Dec 2023 10:31:40 GMT"},
				},
			}
			if test.signKey != "" {
				err := httpsignatures.DefaultSha256Signer.SignRequest("keyID", test.signKey, r)
				assert.NoError(t, err)
			}

			resp := httptest.NewRecorder()
			err := verifySignature(resp, r, test.verifyKey)

			if test.err != nil {
				assert.ErrorIs(t, err, test.err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, test.status, resp.Result().StatusCode)
		})
	}

}

func TestNewReceiver(t *testing.T) {
	defaultConfig := createDefaultConfig().(*Config)

	tests := []struct {
		desc     string
		config   Config
		consumer consumer.Logs
		err      error
	}{
		{
			desc:     "Default config succeeds",
			config:   *defaultConfig,
			consumer: consumertest.NewNop(),
			err:      nil,
		},
		{
			desc: "User defined config success",
			config: Config{
				ServerConfig: confighttp.ServerConfig{
					Endpoint: "localhost:8080",
				},
				Secret: "mysecret",
			},
			consumer: consumertest.NewNop(),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			rec, err := newReceiver(receivertest.NewNopSettings(receivertest.NopType), &test.config)
			if test.err == nil {
				require.NotNil(t, rec)
			} else {
				require.ErrorIs(t, err, test.err)
				require.Nil(t, rec)
			}
		})
	}
}
