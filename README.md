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

## Make CI/CD changes

See [Make CI/CD changes](.drone/README.md)

## Update package level binaries

See [Package Level Binaries](./PACKAGE_LEVEL_BINARIES.md)
