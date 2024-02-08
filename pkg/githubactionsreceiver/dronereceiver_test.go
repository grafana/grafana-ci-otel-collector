package githubactionsreceiver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/httpsignatures-go"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

var requestBody = `"{
  "action": "push",
  "repo": {
    "id": 1,
    "uid": "",
    "user_id": 0,
    "namespace": "",
    "name": "",
    "slug": "repoA",
    "scm": "",
    "git_http_url": "",
    "git_ssh_url": "",
    "link": "",
    "default_branch": "main",
    "private": false,
    "visibility": "",
    "active": false,
    "config_path": "",
    "trusted": false,
    "protected": false,
    "ignore_forks": false,
    "ignore_pull_requests": false,
    "auto_cancel_pull_requests": false,
    "auto_cancel_pushes": false,
    "auto_cancel_running": false,
    "throttle": 0,
    "timeout": 0,
    "counter": 0,
    "synced": 0,
    "created": 0,
    "updated": 0,
    "version": 0,
    "build": {
      "id": 2,
      "repo_id": 0,
      "trigger": "",
      "number": 0,
      "status": "",
      "event": "",
      "action": "",
      "link": "",
      "timestamp": 0,
      "message": "",
      "before": "",
      "after": "",
      "ref": "",
      "source_repo": "",
      "source": "",
      "target": "",
      "author_login": "",
      "author_name": "",
      "author_email": "",
      "author_avatar": "",
      "sender": "",
      "debug": false,
      "started": 0,
      "finished": 12345678,
      "created": 0,
      "updated": 0,
      "version": 0,
      "stages": [
        {
          "id": 1,
          "build_id": 0,
          "number": 0,
          "name": "stageA",
          "status": "",
          "errignore": false,
          "exit_code": 0,
          "os": "",
          "arch": "",
          "started": 0,
          "stopped": 0,
          "created": 0,
          "updated": 0,
          "version": 0,
          "on_success": false,
          "on_failure": false,
          "steps": [
            {
              "id": 1,
              "step_id": 0,
              "number": 1,
              "name": "stepA",
              "status": "success",
              "exit_code": 0,
              "started": 1000,
              "stopped": 1001,
              "version": 0
            }
          ]
        }
      ]
    }
  },
  "system": {
    "host": "host"
  }
}"`

const (
	TestKey   = "TestKey"
	TestDate  = "Thu, 08 Dec 2023 10:31:40 GMT"
	TestKeyId = "Test"
)

func TestVerifySignatureIsValid(t *testing.T) {
	r := requestWithDate()
	err := httpsignatures.DefaultSha256Signer.SignRequest(TestKeyId, TestKey, r)
	assert.Nil(t, err)
	resp := httptest.NewRecorder()
	err = verifySignature(resp, r, TestKey)
	assert.NoError(t, err)

}

func requestWithDate() *http.Request {
	r := &http.Request{
		Header: http.Header{
			"Date": []string{TestDate},
		},
	}
	return r
}

func TestVerifySignatureIsNotValid(t *testing.T) {
	r := &http.Request{
		Header: http.Header{
			"Date": []string{TestDate},
		},
	}
	err := httpsignatures.DefaultSha256Signer.SignRequest(TestKeyId, TestKey, r)
	assert.Nil(t, err)
	resp := httptest.NewRecorder()
	err = verifySignature(resp, r, "badKey")
	assert.Error(t, err)
}

func TestNewgithubactionsreceiver(t *testing.T) {
	mockCfg := &Config{
		DroneConfig: DroneConfig{
			Host:  "host",
			Token: "token",
		},
		WebhookConfig: WebhookConfig{
			Endpoint: "/webhook",
			Secret:   TestKey,
		},
	}

	// Create a test HTTP request to trigger the handler
	req, err := http.NewRequest("POST", mockCfg.WebhookConfig.Endpoint, bytes.NewBufferString(requestBody))
	req.Header = http.Header{"Date": []string{TestDate}}
	err = httpsignatures.DefaultSha256Signer.SignRequest(TestKeyId, TestKey, req)
	assert.NoError(t, err)

	nop := receivertest.NewNopCreateSettings()
	rcvr, err := newgithubactionsreceiver(mockCfg, nop)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()

	rcvr.httpServer.Handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status code 200")
}

func TestNewgithubactionsreceiver_StatusForbidden(t *testing.T) {
	mockCfg := &Config{
		DroneConfig: DroneConfig{
			Host:  "host",
			Token: "token",
		},
		WebhookConfig: WebhookConfig{
			Endpoint: "/webhook",
			Secret:   "badKey",
		},
	}

	// Create a test HTTP request to trigger the handler
	req, err := http.NewRequest("POST", mockCfg.WebhookConfig.Endpoint, bytes.NewBufferString(requestBody))
	req.Header = http.Header{"Date": []string{TestDate}}
	err = httpsignatures.DefaultSha256Signer.SignRequest(TestKeyId, TestKey, req)
	assert.NoError(t, err)

	nop := receivertest.NewNopCreateSettings()
	rcvr, err := newgithubactionsreceiver(mockCfg, nop)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()

	rcvr.httpServer.Handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code, "Expected status code 200")
}
