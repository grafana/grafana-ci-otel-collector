# Package level binaries

Package level binaries are managed using [bingo](https://github.com/bwplotka/bingo).

When updating to a newer otel version, `builder` and mdatagen binaries need to be updated as well to the same version.

In the project root, run (replace `X.XX.X` with the version you want to update to. This must match `otelcol_version` in [config/builder-config.yaml](./config/builder-config.yml#L5).

```
bingo get go.opentelemetry.io/collector/cmd/builder@vX.XX.X
bingo get github.com/open-telemetry/opentelemetry-collector-contrib/cmd/mdatagen@vX.X.XX
```
