FROM golang:1.22.1-alpine3.19 as go-builder
WORKDIR /collector

COPY . .

RUN apk add --no-cache make

RUN make metadata
RUN make build

ENTRYPOINT ["./collector/grafana-ci-otelcol"]
