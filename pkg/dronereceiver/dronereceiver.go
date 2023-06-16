package dronereceiver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	drone "github.com/drone/drone-go/drone"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type droneReceiver struct {
	cfg *Config
	set receiver.CreateSettings

	cancel context.CancelFunc

	handler *droneWebhookHandler

	httpServer *http.Server
}

func newDroneReceiver(cfg *Config, set receiver.CreateSettings) (*droneReceiver, error) {
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

	httpMux := http.NewServeMux()

	handler := droneWebhookHandler{
		droneClient: droneClient,
		logger:      set.Logger.Named("handler"),
	}

	httpMux.HandleFunc(cfg.Endpoint, func(resp http.ResponseWriter, req *http.Request) {
		// TODO: write the handler for the webhook endpoint
		// Maybe route based on the X-Drone-Event header?
		set.Logger.Info("Got request")
		resp.WriteHeader(http.StatusOK)
		handler.handler(resp, req)
	})

	httpServer := &http.Server{
		Handler: httpMux,
	}

	receiver := &droneReceiver{
		cfg: cfg,
		set: set,

		httpServer: httpServer,
		handler:    &handler,
	}

	return receiver, nil
}

func (r *droneReceiver) Start(_ context.Context, host component.Host) error {
	r.set.Logger.Info("starting Drone receiver")

	go func() {
		r.set.Logger.Info("starting http server",
			zap.String("endpoint", r.cfg.Endpoint),
			zap.Int("port", r.cfg.Port),
		)

		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", r.cfg.Port))
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

func (r *droneReceiver) Shutdown(_ context.Context) error {
	r.httpServer.Shutdown(context.Background())
	if r.cancel != nil {
		r.cancel()
	}

	return nil
}

func (r *droneReceiver) enableLogs(consumer consumer.Logs) {
	r.handler.nextLogsConsumer = consumer
}

func (r *droneReceiver) enableTraces(consumer consumer.Traces) {
	r.handler.nextTraceConsumer = consumer
}
