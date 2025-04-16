# STAGE 1 - build
FROM golang:1.24.2-alpine3.20 AS build
WORKDIR /src

COPY . .

RUN apk --update add --no-cache git make bash ca-certificates

RUN make build

# STAGE 2 - final image
FROM alpine:3.20

ARG BIN_PATH=/src/build/grafana-ci-otelcol

# Ensure /tmp directory exists and set permissions
RUN chmod 777 /tmp

ARG UID=10001
USER ${UID}

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build --chmod=755 ${BIN_PATH} /usr/bin/grafana-ci-otelcol

ENTRYPOINT ["/usr/bin/grafana-ci-otelcol"] 
CMD ["--config=/etc/grafana-ci-otelcol/config.yaml"]