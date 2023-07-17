# grafana-ci-collector

## Development

### Configuring the collector

The collector is configured using the `config.yaml` file.
An example configuration can be found in `config.yaml.example`, copy the file to `config.yaml` and replace the values for the `dronereceiver` receiver with the ones relevant to your environment.

```bash
cp config.example.yaml config.yaml
```

### Building

```bash
make metadata && make build
```

### Running

In the example config an exporter is configured to send data locally. A `docker-compose` file is provided to start Grafana Tempo.

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

Update the `dronereceiver` receiver in the `config.yaml` file to use the ngrok forwarding url as follows (example using the URL from above):

```yaml
receivers:
  dronereceiver:
    collection_interval: 15s
    drone:
      token: <YOUR TOKEN>
      host: http://localhost:8080
    endpoint: /drone/webhook
    port: 3333
```

### Start the collector

```bash
make metadata && make build && make run
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

## Setup Prometheus RW
### Install and use the Grafana Agent
Together with the Prometheus exporter which exports metrics to `localhost:9091` we can use 
Prometheus Remote Write (RW) and have the Grafana Agent collect the metrics on our behalf.

In order to make this work, we need to go to our instance and follow the instructions to install Grafana Agent 
(in this case https://gracie.grafana-dev.net/connections/infrastructure/grafana-agent).

Follow the instructions to install Grafana Agent on your local machine.

After the above is done, and once you are sure that the agent runs already as a service on your local machine,
running [a query example in Explore](https://gracie.grafana-dev.net/goto/mP1u6cC4R?orgId=1), should give you info 
about your local machine, discovered and scraped using the agent.

### Configure the Grafana Agent for the OTel Collector
In the `config.yaml` (provided that you've copied these bits from config.example.yaml):

```yaml
exporters:
    prometheusremotewrite:
      endpoint: "https://prometheus-dev-01-dev-us-central-0.grafana-dev.net/api/prom/push"
      auth:
        authenticator: basicauth/client

extensions:
  basicauth/client:
    client_auth:
      username: username
      password: password


service:
  pipelines:
    ...
    metrics:
      exporters: [prometheusremotewrite]
```

you need to replace the `username` and `password` with the ones that are specified in your `agent.yaml` file
(found in `$(brew --prefix)/etc/grafana-agent/config.yml` for MacOS users).

Re-run the collector to pick up the changes. After that you should be able to rerun [the same query example in Explore](https://gracie.grafana-dev.net/goto/mP1u6cC4R?orgId=1)
but this time being able to see the desired metrics such as `builds_metrics` etc.

As of July 14th 2023, both Prometheus exporter and Prometheus Remote Write exporter are used, for ease of local
development.

## Make CI/CD changes

See [Make CI/CD changes](.drone/README.md)
