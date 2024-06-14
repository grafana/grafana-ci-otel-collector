FROM golang:1.22.1-alpine3.19 as go-builder
WORKDIR /build

COPY . .

RUN apk add --no-cache make

RUN make metadata
RUN make build

ENTRYPOINT ["./build/grafana-ci-otelcol"]
