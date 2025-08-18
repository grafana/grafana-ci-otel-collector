# STAGE 1 - build
FROM golang:1.25.0-alpine3.21@sha256:c8e1680f8002c64ddfba276a3c1f763097cb182402673143a89dcca4c107cf17 AS build
WORKDIR /src

COPY . .

RUN apk --update add --no-cache git make bash ca-certificates

RUN make build

# STAGE 2 - final image
FROM scratch

ARG BIN_PATH=/src/build/grafana-ci-otelcol

ARG UID=10001
USER ${UID}

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build --chmod=755 ${BIN_PATH} /usr/bin/grafana-ci-otelcol

ENTRYPOINT ["/usr/bin/grafana-ci-otelcol"] 
CMD ["--config=/etc/grafana-ci-otelcol/config.yaml"]