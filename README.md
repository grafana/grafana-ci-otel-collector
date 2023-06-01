# grafana-ci-collector

## Development

Install the OpenTelemetry Collector Builder by following the instructios in [the official documentation](https://opentelemetry.io/docs/collector/custom-collector/#step-1---install-the-builder)

The project is currently using the `v0.78.2` version of the builder. You can download the binary relevant to your platform from https://github.com/open-telemetry/opentelemetry-collector/releases/tag/cmd%2Fbuilder%2Fv0.78.2

Once downloaded move it into your path and make it executable. on MacOS:

```bash
sudo mv <PATH_TO_DOWNLOADED_FILE> /usr/local/bin/ocb

sudo chmod 777 /usr/local/bin/ocb
```

### Configuring the collector

The collector is configured using the `config.yaml` file.
An example configuration can be found in `config.yaml.example`, copy the file to `config.yaml` and replace the values for the `dronereceiver` receiver with the ones relevant to your environment.

```bash
cp config.yaml.example config.yaml
```

### Building

```bash
make build
```

### Running

In the example config an exportper is configured to send data locally. A `docker-compose` file is provided to start Grafana Tempo.

```bash
docker-compose up -d
```

Then you can start the collector with:

```bash
make run
```

## Drone

### Generating traces

The receiver listens for Drone webhooks and generates trace data based on the information in the webhook payload.

Until a more complete data generator is available, you can simulate a webhook call you can manually send a request to the receiver:

```bash
curl -X POST -H "Content-Type: application/json" -d @./dronereceiver/testdata/build-completed.json http://localhost:3333/drone/webhook
```
