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
    drone:
      token: <YOUR TOKEN>
      host: http://${NETWORK_HOST}:8080
    webhook:
      endpoint: /drone/webhook
      port: 3333
      secret: bea26a2221fd8090ea38720fc445eca6
```

### Start the collector

```bash
make metadata && make build && make run
```

### Spin up the collector as a Docker image

Build the Docker image:

```bash
make docker-build-local
```

Run the Docker image:

```bash
make docker-run
```

Do both at once:

```bash
make docker
```

**NOTES:**

- When building/running the Docker image, we are specifying `$NETWORK_HOST` var to be `host.docker.internal`.

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

## Update package level binaries

See [Package Level Binaries](./PACKAGE_LEVEL_BINARIES.md)
