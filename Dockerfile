FROM golang:1.20-alpine3.17 as go-builder
WORKDIR /collector

ARG network_host
ENV NETWORK_HOST $network_host

COPY . .

RUN apk add --no-cache make

RUN make metadata
RUN make build

ENTRYPOINT ["./collector/grafana-ci-otelcol"]
