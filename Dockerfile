# STAGE 1 - build
FROM golang:1.27.3-alpine3.23@sha256:754d5200fc3edae76d3b6598a9a3fac7e74981ee6a5413ab955025f185753ef2 AS build
WORKDIR /src

COPY . .

RUN apk add --no-cache git make bash 'ca-certificates>=20261020-r0' \
    && update-ca-certificates

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
