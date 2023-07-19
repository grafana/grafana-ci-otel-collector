FROM golang:1.20-alpine3.17 as go-builder
WORKDIR /collector

COPY . .

RUN go install github.com/open-telemetry/opentelemetry-collector-contrib/cmd/mdatagen@v0.79.0
RUN go install go.opentelemetry.io/collector/cmd/builder@v0.79.0

RUN mdatagen ./pkg/dronereceiver/metadata.yaml
RUN builder --config config/builder-config.yml

ENTRYPOINT ["./collector/grafana-ci-otelcol"]
