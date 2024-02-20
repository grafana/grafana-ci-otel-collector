package githubactionsreceiver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/cbrgm/githubevents/githubevents"
	"github.com/google/go-github/v58/github"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"go.uber.org/zap"
)

type githubactionsreceiver struct {
	cfg        *Config
	set        receiver.CreateSettings
	cancel     context.CancelFunc
	handler    *gitHubActionsWebhookHandler
	httpServer *http.Server
}

func newGitHubActionsReceiver(cfg *Config, set receiver.CreateSettings) (*githubactionsreceiver, error) {
	set.Logger.Info("creating githubactions receiver")
	// TODO: both of the following need auth from config
	// githubevents instantiates the same client under the hood, however it's not possible to acess it so we need to create a new one
	// Alternatively, we could avoid using githubevents and use the client directly also fot the webhooks.
	handle := githubevents.New("")
	client := github.NewClient(nil)

	handler := gitHubActionsWebhookHandler{
		logger:   set.Logger.Named("handler"),
		ghClient: client,
	}

	handle.OnWorkflowRunEventCompleted(handler.onWorkflowRunCompleted)
	handle.OnWorkflowJobEventCompleted(handler.onWorkflowJobCompleted)

	httpMux := http.NewServeMux()
	httpMux.HandleFunc(cfg.WebhookConfig.Endpoint, func(resp http.ResponseWriter, req *http.Request) {
		if err := handle.HandleEventRequest(req); err != nil {
			set.Logger.Info("error handling the request", zap.Error(err))
			return
		}

		resp.WriteHeader(http.StatusOK)
	})

	httpServer := &http.Server{
		Handler: httpMux,
	}

	receiver := &githubactionsreceiver{
		cfg:        cfg,
		set:        set,
		httpServer: httpServer,
		handler:    &handler,
	}

	// receiver, becaause we don't pay for vowels
	return receiver, nil
}

func (r *githubactionsreceiver) Start(_ context.Context, host component.Host) error {
	r.set.Logger.Info("starting GHActions receiver")

	go func() {
		r.set.Logger.Info("starting http server",
			zap.String("endpoint", r.cfg.WebhookConfig.Endpoint),
			zap.Int("port", r.cfg.WebhookConfig.Port),
		)

		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", r.cfg.WebhookConfig.Port))
		if err != nil {
			r.set.Logger.Error("error creating listener",
				zap.String("error", err.Error()),
			)
			host.ReportFatalError(err)
		}

		if errHTTP := r.httpServer.Serve(listener); errHTTP != nil && !errors.Is(errHTTP, http.ErrServerClosed) {
			r.set.Logger.Error("error starting server",
				zap.String("error", errHTTP.Error()),
			)
			host.ReportFatalError(errHTTP)
		}
	}()

	return nil
}

func (r *githubactionsreceiver) Shutdown(_ context.Context) error {
	err := r.httpServer.Shutdown(context.Background())
	if err != nil {
		return err
	}
	if r.cancel != nil {
		r.cancel()
	}

	return nil
}

func (r *githubactionsreceiver) enableLogs(consumer consumer.Logs) {
	r.handler.nextLogsConsumer = consumer
}

func (r *githubactionsreceiver) enableTraces(consumer consumer.Traces) {
	r.handler.nextTraceConsumer = consumer
}
