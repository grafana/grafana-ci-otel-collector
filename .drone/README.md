# Make CI/CD changes

## crdsonnet/drone-libsonnet

`drone-libsonnet` is a library written in `libsonnet` which is responsible for creating Drone pipelines.

[Full documentation](https://github.com/crdsonnet/drone-libsonnet/blob/master/docs/README.md)

You need to run:

```bash
jb install github.com/crdsonnet/drone-libsonnet@master
```

This will generate a `vendor` folder in your `.drone/` folder which has been already added in the `.gitignore` file,
containing all the necessary dependencies for the library to operate.

## Make pipeline(s) changes

`.drone/drone.jsonnet` is the file you are looking for in order to make changes. Have a look at the above `drone-libsonnet`
documentation to get more options about your CI/CD needs.

`.drone/jsonnet*` files are associated with the `drone-libsonnet` library version.

## Regenerate `.drone.yml`

> [!NOTE]
> Make sure you have the `DRONE_SERVER` and `DRONE_TOKEN` environment variables set in your shell. You can find them in https://drone.grafana.net/account.

After you are happy with the changes you've made in `.drone/drone.jsonnet` from your root project directory, run:

```bash
make drone
```

or

```bash
drone jsonnet --stream \
              --format \
              --source <(jsonnet -J .drone/vendor/ .drone/drone.jsonnet) \
              --target .drone.yml
```
