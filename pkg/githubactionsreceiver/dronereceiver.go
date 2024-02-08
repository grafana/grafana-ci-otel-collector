package githubactionsreceiver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/99designs/httpsignatures-go"
	"github.com/drone/drone-go/drone"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type githubactionsreceiver struct {
	cfg        *Config
	set        receiver.CreateSettings
	cancel     context.CancelFunc
	handler    *droneWebhookHandler
	httpServer *http.Server
}

func newgithubactionsreceiver(cfg *Config, set receiver.CreateSettings) (*githubactionsreceiver, error) {
	set.Logger.Info("creating the drone receiver")

	// Create a drone client
	oauthConfig := new(oauth2.Config)
	httpClient := oauthConfig.Client(
		context.Background(),
		&oauth2.Token{
			AccessToken: cfg.DroneConfig.Token,
		},
	)
	droneClient := drone.NewClient(cfg.DroneConfig.Host, httpClient)

	handler := droneWebhookHandler{
		reposConfig: cfg.ReposConfig,
		droneClient: droneClient,
		logger:      set.Logger.Named("handler"),
	}

	httpMux := http.NewServeMux()
	httpMux.HandleFunc(cfg.WebhookConfig.Endpoint, func(resp http.ResponseWriter, req *http.Request) {
		//TODO:  Maybe route based on the X-Drone-Event header?
		err := verifySignature(resp, req, cfg.WebhookConfig.Secret)
		if err != nil {
			set.Logger.Info("couldn't verify request signature", zap.Error(err))
			return
		}

		resp.WriteHeader(http.StatusOK)
		if err := handler.handler(resp, req); err != nil {
			set.Logger.Info("error handling the request", zap.Error(err))
			return
		}
	})

	httpServer := &http.Server{
		Handler: httpMux,
	}

	rcvr := &githubactionsreceiver{
		cfg:        cfg,
		set:        set,
		httpServer: httpServer,
		handler:    &handler,
	}

	return rcvr, nil
}

func verifySignature(resp http.ResponseWriter, req *http.Request, secret string) error {
	sig, err := httpsignatures.FromRequest(req)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("error parsing signature: %w", err)
	}

	if !sig.IsValid(secret, req) {
		resp.WriteHeader(http.StatusForbidden)
		return fmt.Errorf("signature is not valid")
	}

	return nil
}

func (r *githubactionsreceiver) Start(_ context.Context, host component.Host) error {
	r.set.Logger.Info("starting Drone receiver")

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
