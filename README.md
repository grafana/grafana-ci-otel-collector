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
[githubactionsreceiver]: https://github.com/grafana/opentelemetry-collector-contrib/tree/feat-add-githubactionseventreceiver-2/receiver/githubactionsreceiver

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

The collector is configured using the `config.yaml` file.
You can learn more about the OpenTelemetry Collector configuration [here][otel-configuration].

Refer to each component's documentation (linked above) for specific configuration options.

[otel-configuration]: https://opentelemetry.io/docs/collector/configuration/

## Development

After having configured the collector, you can start it by running:

```bash
make run
```

## Local Drone instance

It is possible to use a local Drone instance for easier development.

Note that by default no webhook events are sent to the receiver from GitHub (i.e. when you push to a branch to trigger a build), you need to manually trigger builds on Drone.
If you need to get those webhooks, you can configure it your repository settings on GitHub.

### Environment variables

The `docker-compose.localdrone.yml` file expects the following environment variables to be set:

```bash
DRONE_SERVER_PROXY_HOST=
DRONE_GITHUB_CLIENT_ID=
DRONE_GITHUB_CLIENT_SECRET=
GH_HANDLE=
```

you can copy the example env vars file and replace the values:

```bash
cp .env.example .env
```

### ngrok

First, [install ngrok](https://ngrok.com/download) to expose a tunnel to your local drone instance.

Once installed, start ngrok with:

```bash
ngrok http 8080
```

the output should look something like this:

```bash
Session Status                online
Account                       you@example.com
Version                       3.3.1
Region                        Europe (eu)
Latency                       44ms
Web Interface                 http://127.0.0.1:4040
Forwarding                    https://3dfc-2001-818-d8d9-a00-e5-c197-b7d2-3551.ngrok-free.app -> http://localhost:8080
```

Copy the forwarding url (in this case `https://3dfc-2001-818-d8d9-a00-e5-c197-b7d2-3551.ngrok-free.app`) and use it to configure the `DRONE_SERVER_PROXY_HOST` environment variable in the `.env` file.

### GitHub OAuth App

We then need to create a GitHub OAuth App to use for authentication with Drone.
Go to **Settings -> Developer settings -> OAuth Apps** and click on "New OAuth App".

Pick whatever you want for the name and description, and use the ngrok forwarding url for the `Homepage URL` and `Authorization callback URL` fields as follows (example using the URL from above):

```
Homepage URL:
https://3dfc-2001-818-d8d9-a00-e5-c197-b7d2-3551.ngrok-free.app


Authorization callback URL:
https://3dfc-2001-818-d8d9-a00-e5-c197-b7d2-3551.ngrok-free.app/login
```

Click on "Register application".

After the application is registered, generate a `Client secret`.

Take note of the `Client ID` and `Client secret` values and use them to configure the `DRONE_GITHUB_CLIENT_ID` and `DRONE_GITHUB_CLIENT_SECRET` environment variables in the `.env` file.

### Run Drone

You can now start Drone with:

```bash
docker compose -f docker-compose.localdrone.yaml up -d
```

And use the ngrok forwarding url to access the Drone UI.
Navigate to the repository you want to start monitoring and click on "Activate repository".

### Get your drone token

If you filled in the `GH_HANDLE` environment variable in the `.env` file, your user has admin privileges. You can get your drone token by navigating to https://3dfc-2001-818-d8d9-a00-e5-c197-b7d2-3551.ngrok-free.app/account (replace the url with your ngrok forwarding url) and copy the token.

### Configure the collector

Update the `dronereceiver` receiver in the `config.yaml` file to use the [drone token](#get-your-drone-token) from above:

```yaml
receivers:
  dronereceiver:
    collection_interval: 15s
    endpoint: localhost:3333
    path: /drone/webhook
    secret: bea26a2221fd8090ea38720fc445eca6
    drone:
      token: <YOUR TOKEN>
      host: http://${NETWORK_HOST}:8080
```

## Spin up Grafana as a Docker image locally

### Run Docker image

Choose your Grafana image version. In this example we'll use `10.0.0`. Make sure to add `--add-host=host.docker.internal:host-gateway`
for the image to be able to have access to your personal machine's localhost.

```bash
docker run --add-host=host.docker.internal:host-gateway --rm -p 3000:3000 grafana/grafana:10.0.0
```

### Set up Tempo datasource

As of Grafana 10:

`Toggle menu` -> `Connections` -> `Data sources` -> `Search for Tempo` -> `+ Add new data source`

Under `HTTP`, in the `URL` field, provided that you still use the default app's port for Tempo (`3200`), add:

```
http://host.docker.internal:3200
```

Click `Save & Test`

You are now ready to see your traces collector and play around with it using Tempo in Explore, or while building a new
dashboard!

## Make CI/CD changes

See [Make CI/CD changes](.drone/README.md)
