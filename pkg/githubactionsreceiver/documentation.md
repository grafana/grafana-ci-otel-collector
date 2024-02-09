[comment]: <> (Code generated by mdatagen. DO NOT EDIT.)

# githubactionsreceiver

## Default Metrics

The following metrics are emitted by default. Each of them can be disabled by applying the following configuration:

```yaml
metrics:
  <metric_name>:
    enabled: false
```

### builds_number

Number of builds.

Currently there's no way to differentiate between restarted builds and manually triggered builds. This means builds started manually (i.e. via Drone UI or via APis) will count towards this metric should they run against a branch for which a build has already been executed.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| 1    | Sum         | Int        | Cumulative              | false     |

#### Attributes

| Name                    | Description     | Values                                                                                                                          |
| ----------------------- | --------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| ci.workflow_item.status | Build status    | Str: `skipped`, `blocked`, `declined`, `waiting_on_dependencies`, `pending`, `running`, `success`, `failure`, `killed`, `error` |
| git.repo.name           | Repository name | Any Str                                                                                                                         |
| git.branch.name         | Branch name     | Any Str                                                                                                                         |

### repo_info

Repo status.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| 1    | Sum         | Int        | Cumulative              | false     |

#### Attributes

| Name                    | Description     | Values                                                                                                                          |
| ----------------------- | --------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| ci.workflow_item.status | Build status    | Str: `skipped`, `blocked`, `declined`, `waiting_on_dependencies`, `pending`, `running`, `success`, `failure`, `killed`, `error` |
| git.repo.name           | Repository name | Any Str                                                                                                                         |
| git.branch.name         | Branch name     | Any Str                                                                                                                         |

### restarts_total

Total number build restarts.

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| 1    | Sum         | Int        | Cumulative              | true      |