// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package githubactionsreceiver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v83/github"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"
)

var errMissingEndpoint = errors.New("missing a receiver endpoint")

type githubActionsReceiver struct {
	logsConsumer    consumer.Logs
	tracesConsumer  consumer.Traces
	metricsConsumer consumer.Metrics
	metricsHandler  metricsHandler
	config          *Config
	server          *http.Server
	shutdownWG      sync.WaitGroup
	createSettings  receiver.Settings
	logger          *zap.Logger
	obsrecv         *receiverhelper.ObsReport
	ghClient        *github.Client
	ghitr           *ghinstallation.Transport
}

func newReceiver(
	params receiver.Settings,
	config *Config,
) (*githubActionsReceiver, error) {
	if config.NetAddr.Endpoint == "" {
		return nil, errMissingEndpoint
	}

	transport := "http"
	if config.TLS.HasValue() {
		transport = "https"
	}

	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             params.ID,
		Transport:              transport,
		ReceiverCreateSettings: params,
	})

	if err != nil {
		return nil, err
	}

	var ghClient *github.Client
	var httpClient *http.Client
	var itr *ghinstallation.Transport
	if config.GitHubAPIConfig.Auth.AppID != 0 && config.GitHubAPIConfig.Auth.InstallationID != 0 && config.GitHubAPIConfig.Auth.PrivateKeyPath != "" {
		itr, err = ghinstallation.NewKeyFromFile(http.DefaultTransport, config.GitHubAPIConfig.Auth.AppID, config.GitHubAPIConfig.Auth.InstallationID, config.GitHubAPIConfig.Auth.PrivateKeyPath)
		if err != nil {
			return nil, err
		}

		if config.GitHubAPIConfig.BaseURL != "" && config.GitHubAPIConfig.UploadURL != "" {
			// The BaseURL should point to the /api/v3 prefix. We can use the
			// WithEnterpriseURLs helper for that which does validation and
			// proper suffixing if necessary.
			tmp := github.NewClient(nil).WithAuthToken("")
			tmp, err = tmp.WithEnterpriseURLs(config.GitHubAPIConfig.BaseURL, config.GitHubAPIConfig.UploadURL)
			if err != nil {
				return nil, fmt.Errorf("enterprise URLs are invalid: %w", err)
			}
			itr.BaseURL = tmp.BaseURL.String()
		}

		httpClient = &http.Client{Transport: itr}
	}
	ghClient = github.NewClient(httpClient)

	if config.GitHubAPIConfig.Auth.Token != "" {
		ghClient = ghClient.WithAuthToken(config.GitHubAPIConfig.Auth.Token)
	}

	if config.GitHubAPIConfig.BaseURL != "" && config.GitHubAPIConfig.UploadURL != "" {
		ghClient, err = ghClient.WithEnterpriseURLs(config.GitHubAPIConfig.BaseURL, config.GitHubAPIConfig.UploadURL)
		if err != nil {
			return nil, err
		}
	}

	gar := &githubActionsReceiver{
		config:         config,
		createSettings: params,
		logger:         params.Logger,
		obsrecv:        obsrecv,
		ghClient:       ghClient,
		ghitr:          itr,
		metricsHandler: *newMetricsHandler(params, config, params.Logger.Named("metricsHandler")),
	}

	return gar, nil
}

// newLogsReceiver creates a trace receiver based on provided config.
func newTracesReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	consumer consumer.Traces,
) (receiver.Traces, error) {
	rCfg := cfg.(*Config)
	var err error

	r := receivers.GetOrAdd(cfg, func() component.Component {
		var rcv component.Component
		rcv, err = newReceiver(set, rCfg)
		return rcv
	})
	if err != nil {
		return nil, err
	}

	r.Unwrap().(*githubActionsReceiver).tracesConsumer = consumer

	return r, nil
}

// newLogsReceiver creates a logs receiver based on provided config.
func newLogsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	rCfg := cfg.(*Config)
	var err error

	r := receivers.GetOrAdd(cfg, func() component.Component {
		var rcv component.Component
		rcv, err = newReceiver(set, rCfg)
		return rcv
	})
	if err != nil {
		return nil, err
	}

	r.Unwrap().(*githubActionsReceiver).logsConsumer = consumer

	return r, nil
}

// newMetricsReceiver creates a logs receiver based on provided config.
func newMetricsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	rCfg := cfg.(*Config)
	var err error

	r := receivers.GetOrAdd(cfg, func() component.Component {
		var rcv component.Component
		rcv, err = newReceiver(set, rCfg)
		return rcv
	})
	if err != nil {
		return nil, err
	}

	r.Unwrap().(*githubActionsReceiver).metricsConsumer = consumer

	return r, nil
}

func (gar *githubActionsReceiver) Start(ctx context.Context, host component.Host) error {
	endpoint := fmt.Sprintf("%s%s", gar.config.NetAddr.Endpoint, gar.config.Path)
	gar.logger.Info("Starting GithubActions server", zap.String("endpoint", endpoint))
	gar.server = &http.Server{
		Addr:              gar.config.NetAddr.Endpoint,
		Handler:           gar,
		ReadHeaderTimeout: 20 * time.Second,
	}

	gar.shutdownWG.Add(1)
	go func() {
		defer gar.shutdownWG.Done()

		if errHTTP := gar.server.ListenAndServe(); !errors.Is(errHTTP, http.ErrServerClosed) && errHTTP != nil {
			gar.createSettings.Logger.Error("Server closed with error", zap.Error(errHTTP))
		}
	}()

	return nil
}

func (gar *githubActionsReceiver) Shutdown(ctx context.Context) error {
	var err error
	if gar.server != nil {
		err = gar.server.Close()
	}
	gar.shutdownWG.Wait()
	return err
}

func (gar *githubActionsReceiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validate request path
	if r.URL.Path != gar.config.Path {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Validate the payload using the configured secret
	payload, err := github.ValidatePayload(r, []byte(gar.config.Secret))
	if err != nil {
		gar.logger.Debug("Payload validation failed", zap.Error(err))
		http.Error(w, "Invalid payload or signature", http.StatusBadRequest)
		return
	}

	// Determine the type of GitHub webhook event and ensure it's one we handle
	eventType := github.WebHookType(r)
	event, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		gar.logger.Debug("Webhook parsing failed", zap.Error(err))
		http.Error(w, "Failed to parse webhook", http.StatusBadRequest)
		return
	}

	// Handle events based on specific types and completion status
	switch e := event.(type) {
	case *github.WorkflowJobEvent:
		if gar.metricsConsumer != nil {
			err := gar.metricsConsumer.ConsumeMetrics(ctx, gar.metricsHandler.workflowJobEventToMetrics(e))

			if err != nil {
				gar.logger.Error("Failed to consume metrics", zap.Error(err))
			}
		}

		if e.GetWorkflowJob().GetStatus() != "completed" {
			gar.logger.Debug("Skipping non-completed WorkflowJobEvent", zap.String("status", e.GetWorkflowJob().GetStatus()))
			w.WriteHeader(http.StatusNoContent)
			return
		}
	case *github.WorkflowRunEvent:
		if gar.metricsConsumer != nil && e.GetWorkflowRun().GetEvent() == "push" {
			err := gar.metricsConsumer.ConsumeMetrics(ctx, gar.metricsHandler.workflowRunEventToMetrics(e))

			if err != nil {
				gar.logger.Error("Failed to consume metrics", zap.Error(err))
			}
		}

		if e.GetWorkflowRun().GetStatus() != "completed" {
			gar.logger.Debug("Skipping non-completed WorkflowRunEvent", zap.String("status", e.GetWorkflowRun().GetStatus()))
			w.WriteHeader(http.StatusNoContent)
			return
		}
	default:
		gar.logger.Debug("Skipping unsupported event type", zap.String("event", eventType))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	gar.logger.Debug("Received valid GitHub event", zap.String("type", eventType))
	traceErr := false

	// if a trace consumer is set, process the event into traces
	if gar.tracesConsumer != nil {
		td, err := eventToTraces(event, gar.config, gar.logger.Named("eventToTraces"))
		if err != nil {
			traceErr = true
			gar.logger.Debug("Failed to convert event to traces", zap.Error(err))
		}

		if td != nil {
			// Pass the traces to the nextConsumer
			consumerErr := gar.tracesConsumer.ConsumeTraces(ctx, *td)
			if consumerErr != nil {
				traceErr = true
				gar.logger.Debug("Failed to process traces", zap.Error(consumerErr))
			}
		}
	}

	// if a log consumer is set, process the event into logs
	if gar.logsConsumer != nil {
		if gar.ghClient == nil {
			gar.logger.Error("GitHub token not provided, but a logs consumer is set. Logs will not be processed. Please provide a GitHub token.")
		} else {
			withTraceInfo := gar.tracesConsumer != nil && !traceErr

			gar.logger.Debug("Calling eventToLogs")
			gar.logger.Debug("Event type being passed to eventToLogs", zap.String("event_type", fmt.Sprintf("%T", event)))

			ld, err := eventToLogs(event, gar.config, gar.ghClient, gar.logger.Named("eventToLogs"), withTraceInfo)
			if err != nil {
				gar.logger.Error("Failed to process logs", zap.Error(err))
			}

			if ld != nil {
				consumerErr := gar.logsConsumer.ConsumeLogs(ctx, *ld)
				if consumerErr != nil {
					gar.logger.Error("Failed to consume logs", zap.Error(consumerErr))
				}
			}
		}
	}

	w.WriteHeader(http.StatusAccepted)
}
