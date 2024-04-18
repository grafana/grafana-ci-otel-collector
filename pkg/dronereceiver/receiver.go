package dronereceiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/99designs/httpsignatures-go"
	"github.com/drone/drone-go/drone"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

var errSignatureNotValid = errors.New("signature is not valid")
var errParsingSignature = errors.New("error parsing signature")

type droneReceiver struct {
	cfg         *Config
	set         receiver.CreateSettings
	httpServer  *http.Server
	shutdownWG  sync.WaitGroup
	droneClient drone.Client
	obsrecv     *receiverhelper.ObsReport
	logger      *zap.Logger

	logsConsumer   consumer.Logs
	tracesConsumer consumer.Traces
}

func newReceiver(params receiver.CreateSettings,
	config *Config) (*droneReceiver, error) {

	transport := "http"
	if config.TLSSetting != nil {
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

	// Create a drone client
	oauthConfig := new(oauth2.Config)
	httpClient := oauthConfig.Client(
		context.Background(),
		&oauth2.Token{
			AccessToken: config.DroneConfig.Token,
		},
	)
	droneClient := drone.NewClient(config.DroneConfig.Host, httpClient)

	receiver := &droneReceiver{
		cfg:         config,
		set:         params,
		droneClient: droneClient,
		obsrecv:     obsrecv,
		logger:      params.Logger,
	}

	return receiver, nil
}

func verifySignature(resp http.ResponseWriter, req *http.Request, secret string) error {
	sig, err := httpsignatures.FromRequest(req)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("%w: %w", errParsingSignature, err)
	}

	if !sig.IsValid(secret, req) {
		resp.WriteHeader(http.StatusForbidden)
		return errSignatureNotValid
	}

	return nil
}

func (r *droneReceiver) Start(_ context.Context, host component.Host) error {
	endpoint := fmt.Sprintf("%s%s", r.cfg.Endpoint, r.cfg.Path)
	r.logger.Info("Starting Drone webhook server", zap.String("endpoint", endpoint))

	r.httpServer = &http.Server{
		Addr:              r.cfg.ServerConfig.Endpoint,
		Handler:           r,
		ReadHeaderTimeout: 20 * time.Second,
	}

	r.shutdownWG.Add(1)
	go func() {
		defer r.shutdownWG.Done()

		if errHTTP := r.httpServer.ListenAndServe(); !errors.Is(errHTTP, http.ErrServerClosed) && errHTTP != nil {
			r.set.TelemetrySettings.Logger.Error("Server closed with error", zap.Error(errHTTP))
		}
	}()

	return nil
}

func (r *droneReceiver) Shutdown(_ context.Context) error {
	var err error
	if r.httpServer != nil {
		err = r.httpServer.Close()
	}
	r.shutdownWG.Wait()
	return err
}

func (r *droneReceiver) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	err := verifySignature(resp, req, r.cfg.Secret)
	if err != nil {
		r.logger.Info("couldn't verify request signature", zap.Error(err))
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		r.logger.Error("error reading the request body", zap.Error(err))
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	var evt WebhookEvent
	err = json.Unmarshal(body, &evt)
	if err != nil {
		// TODO: handle this
		r.logger.Error("error unmarshalling the request body", zap.Error(err))
		http.Error(resp, err.Error(), http.StatusBadRequest)
		return
	}

	traces, logs := handleEvent(evt, r.cfg, r.droneClient, r.logger.Named("handler"))

	if r.tracesConsumer != nil && traces != nil {
		err := r.tracesConsumer.ConsumeTraces(req.Context(), *traces)
		if err != nil {
			r.logger.Error("Failed to consume traces", zap.Error(err))
		}
	}
	if r.logsConsumer != nil && logs != nil {
		err := r.logsConsumer.ConsumeLogs(req.Context(), *logs)
		if err != nil {
			r.logger.Error("Failed to consume logs", zap.Error(err))
		}
	}
}
