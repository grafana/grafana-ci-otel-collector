# grafana-ci-otel-collector

The Grafana CI OTel Collector is a distribution of the Open Telemetry collector tailored to CI/CD Observability.

## Components

The following is a list of components that are included in the Grafana CI OTel Collector.

<mark>**Highlighted**</mark> components are currently being developed by us.

### Receivers

- [otlpreceiver][otlpreceiver]
- <mark>**[dronereceiver][dronereceiver]**</mark>
- <mark>**[githubactionsreceiver][githubactionsreceiver]**</mark>

[otlpreceiver]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.99.0/receiver/otlpreceiver
[dronereceiver]: ./receiver/dronereceiver/README.md
[githubactionsreceiver]: ./receiver/githubactionsreceiver/README.md

### Processors

- [attributesprocessor][attributesprocessor]
- [batchprocessor][batchprocessor]
- [resourceprocessor][resourceprocessor]

[attributesprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.99.0/processor/attributesprocessor
[batchprocessor]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.99.0/processor/batchprocessor
[resourceprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.99.0/processor/resourceprocessor

### Connectors

- [routingconnector][routingconnector]
- [spanmetricsconnector][spanmetricsconnector]

[routingconnector]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.99.0/connector/routingconnector
[spanmetricsconnector]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.99.0/connector/spanmetricsconnector

### Exporters

- [debugexporter][debugexporter]
- [lokiexporter][lokiexporter]
- [otlpexporter][otlpexporter]
- [prometheusexporter][prometheusexporter]
- [prometheusremotewriteexporter][prometheusremotewriteexporter]

[debugexporter]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.99.0/exporter/debugexporter
[lokiexporter]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.99.0/exporter/lokiexporter
[otlpexporter]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.99.0/exporter/otlpexporter
[prometheusexporter]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.99.0/exporter/prometheusexporter
[prometheusremotewriteexporter]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.99.0/exporter/prometheusremotewriteexporter

### Extensions

- [basicauthextension][basicauthextension]
- [healthcheckextension][healthcheckextension]

[basicauthextension]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.99.0/extension/basicauthextension
[healthcheckextension]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.99.0/extension/healthcheckextension

## Configuration

The collector is configured using the `config.yaml` file in the root of the repository.
You can learn more about the OpenTelemetry Collector configuration [here][otel-configuration].

Refer to each component's documentation (linked above) for specific configuration options.

A barebones configuration file can be found in the `config.example.yaml` file in the root of the repository, with preconfigured exporters for Loki, Tempo, and Prometheus. You can use this file as a starting point for your own configuration.

[otel-configuration]: https://opentelemetry.io/docs/collector/configuration/

## Running Tempo, Loki, and Prometheus

[A basic Docker compose file](./docker-compose.yml) is provided in the root of the repository to run Tempo, Loki, and Prometheus.

To start the services, run:

```bash
docker compose up
```
