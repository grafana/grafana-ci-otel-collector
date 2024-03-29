local drone = import 'github.com/Duologic/drone-libsonnet/main.libsonnet';
local pl = drone.pipeline.docker;
local step = pl.step;
local secret = drone.secret;

local goImage = 'golang:1.20.4';
local dockerDINDImage = 'docker:dind';
local updaterImage = 'us.gcr.io/kubernetes-dev/drone/plugins/updater';
local dockerVolume = {
  name: 'docker',
  host: {
    path: '/var/run/docker.sock',
  },
};
local dockerDindVolume = {
  name: 'dockerDind',
  temp: {},
};

local prTrigger = {
  event: [
    'pull_request',
  ],
};

local customTrigger = {
  event: [
    'custom',
  ],
};

local mainTrigger = {
  branch: 'main',
  event: [
    'push',
  ],
};

local hackathonTheodoraTrigger = {
  branch: 'hackathon-theodora',
  event: [
    'push',
  ],
};

local verifyGenTrigger = {
  event: [
    'pull_request',
  ],
  paths: {
    include: ['pkg/dronereceiver/metadata.yaml'],
  },
};

[
  pl.new('pr')
  + pl.withImagePullSecrets(['dockerconfigjson'])
  + pl.withTrigger(prTrigger)
  + pl.withVolumes([
    dockerVolume,
  ])
  + pl.withSteps([
    step.new('build', image=goImage)
    + step.withCommands([
      'make build',
    ]),
    step.new('test', image=goImage)
    + step.withDependsOn(['build'])
    + step.withCommands([
      'go test ./pkg/dronereceiver/...',
    ]),
    step.new('build-docker-image', image=dockerDINDImage)
    + step.withCommands([
        'docker build .',
    ])
    + step.withVolumes([
        {
            name: 'docker',
            path: '/var/run/docker.sock',
        },
    ]),
  ]),
  pl.new('custom')
  + pl.withImagePullSecrets(['dockerconfigjson'])
  + pl.withTrigger(customTrigger)
  + pl.withSteps([
    step.new('build', image=goImage)
    + step.withCommands([
      'make build',
    ]),
    step.new('test', image=goImage)
    + step.withDependsOn(['build'])
    + step.withCommands([
      'go test ./pkg/dronereceiver/...',
    ]),
  ]),
  pl.new('main')
  + pl.withImagePullSecrets(['dockerconfigjson'])
  + pl.withTrigger(mainTrigger)
  + pl.withVolumes([
    dockerVolume,
    dockerDindVolume,
  ])
  + pl.withSteps([
    step.new('build', image=goImage)
    + step.withCommands([
      'make build',
    ]),
    step.new('test', image=goImage)
    + step.withDependsOn(['build'])
    + step.withCommands([
      'go test ./pkg/dronereceiver/...',
    ]),
    step.new('build-docker-image', image=dockerDINDImage)
    + step.withCommands([
        'docker build --tag us.gcr.io/kubernetes-dev/grafana-ci-otel-collector:${DRONE_COMMIT} .',
    ])
    + step.withVolumes([
        {
            name: 'dockerDind',
            path: '/var/run',
        },
        {
            name: 'docker',
            path: '/var/run/docker.sock',
        },
    ]),
    step.new('publish-to-gcr', image=dockerDINDImage)
    + step.withDependsOn(['build-docker-image'])
    + step.withCommands([
        'echo $${GCR_CREDENTIALS} | docker login -u _json_key --password-stdin https://us.gcr.io',
        'docker push us.gcr.io/kubernetes-dev/grafana-ci-otel-collector:${DRONE_COMMIT}',
    ])
    + step.withEnvironment({
        GCR_CREDENTIALS: {
            from_secret: 'gcr_credentials',
        },
    })
    + step.withVolumes([
        {
            name: 'dockerDind',
            path: '/var/run',
        },
        {
            name: 'docker',
            path: '/var/run/docker.sock',
        },
    ]),
    step.new('update-deployment-tools-dev', image=updaterImage)
    + step.withDependsOn(['publish-to-gcr'])
    + step.withSettings({
      config_json: |||
        {
          "destination_branch": "master",
          "pull_request_branch_prefix": "auto-merge/grafana-ci-otel-collector/",
          "pull_request_enabled": true,
          "pull_request_team_reviewers": [],
          "pull_request_title": "Dev: Update grafana-ci-otel-collector",
          "repo_name": "deployment_tools",
          "update_jsonnet_attribute_configs": [
            {
              "file_path": "ksonnet/environments/grafana-ci-otel-collector/image-dev.libsonnet",
              "jsonnet_key": "dev",
              "jsonnet_value": "us.gcr.io/kubernetes-dev/grafana-ci-otel-collector:${DRONE_COMMIT}"
            }
          ]
        }
      |||,
      github_app_id: {
        from_secret: 'gh_app_id',
      },
      github_app_installation_id: {
        from_secret: 'gh_app_installation_id',
      },
      github_app_private_key: {
        from_secret: 'gh_app_private_key',
      },
    }),
    step.new('update-deployment-tools-ops', image=updaterImage)
    + step.withDependsOn(['publish-to-gcr'])
    + step.withSettings({
      config_json: |||
        {
          "destination_branch": "master",
          "pull_request_branch_prefix": "grafana-ci-otel-collector/",
          "pull_request_enabled": true,
          "pull_request_team_reviewers": [],
          "pull_request_title": "Ops: Update grafana-ci-otel-collector ",
          "repo_name": "deployment_tools",
          "update_jsonnet_attribute_configs": [
            {
              "file_path": "ksonnet/environments/grafana-ci-otel-collector/image-ops.libsonnet",
              "jsonnet_key": "ops",
              "jsonnet_value": "us.gcr.io/kubernetes-dev/grafana-ci-otel-collector:${DRONE_COMMIT}"
            }
          ]
        }
      |||,
      github_app_id: {
        from_secret: 'gh_app_id',
      },
      github_app_installation_id: {
        from_secret: 'gh_app_installation_id',
      },
      github_app_private_key: {
        from_secret: 'gh_app_private_key',
      },
    }),
  ]),
  pl.new('hackathon-theodora')
  + pl.withImagePullSecrets(['dockerconfigjson'])
  + pl.withTrigger(hackathonTheodoraTrigger)
  + pl.withVolumes([
    dockerVolume,
    dockerDindVolume,
  ])
  + pl.withSteps([
    step.new('build', image=goImage)
    + step.withCommands([
      'make build',
    ]),
    step.new('test', image=goImage)
    + step.withDependsOn(['build'])
    + step.withCommands([
      'go test ./pkg/dronereceiver/...',
    ]),
    step.new('build-docker-image', image=dockerDINDImage)
    + step.withCommands([
        'docker build --tag us.gcr.io/kubernetes-dev/grafana-ci-otel-collector-hackathon-theodora:${DRONE_COMMIT} .',
    ])
    + step.withVolumes([
        {
            name: 'dockerDind',
            path: '/var/run',
        },
        {
            name: 'docker',
            path: '/var/run/docker.sock',
        },
    ]),
    step.new('publish-to-gcr', image=dockerDINDImage)
    + step.withDependsOn(['build-docker-image'])
    + step.withCommands([
        'echo $${GCR_CREDENTIALS} | docker login -u _json_key --password-stdin https://us.gcr.io',
        'docker push us.gcr.io/kubernetes-dev/grafana-ci-otel-collector-hackathon-theodora:${DRONE_COMMIT}',
    ])
    + step.withEnvironment({
        GCR_CREDENTIALS: {
            from_secret: 'gcr_credentials',
        },
    })
    + step.withVolumes([
        {
            name: 'dockerDind',
            path: '/var/run',
        },
        {
            name: 'docker',
            path: '/var/run/docker.sock',
        },
    ]),
    step.new('update-deployment-tools-dev', image=updaterImage)
    + step.withDependsOn(['publish-to-gcr'])
    + step.withSettings({
      config_json: |||
        {
          "destination_branch": "master",
          "pull_request_branch_prefix": "auto-merge/grafana-ci-otel-collector/",
          "pull_request_enabled": true,
          "pull_request_team_reviewers": [],
          "pull_request_title": "Dev: Update grafana-ci-otel-collector-hackathon-theodora",
          "repo_name": "deployment_tools",
          "update_jsonnet_attribute_configs": [
            {
              "file_path": "ksonnet/environments/grafana-ci-otel-collector/image-dev.libsonnet",
              "jsonnet_key": "hackathon_theodora",
              "jsonnet_value": "us.gcr.io/kubernetes-dev/grafana-ci-otel-collector-hackathon-theodora:${DRONE_COMMIT}"
            }
          ]
        }
      |||,
      github_app_id: {
        from_secret: 'gh_app_id',
      },
      github_app_installation_id: {
        from_secret: 'gh_app_installation_id',
      },
      github_app_private_key: {
        from_secret: 'gh_app_private_key',
      },
    }),
  ]),
  pl.new('verify-gen-pipeline')
  + pl.withTrigger(verifyGenTrigger)
  + pl.withSteps([
    step.new('verify-gen', image=goImage)
    + step.withCommands([
      'make metadata',
      'if [ -n "$(git status --porcelain)" ]; then echo "ERROR: Please run make metadata and commit your changes." && git diff --exit-code; fi',
    ]),
  ]),
  secret.new('gcr_credentials', 'infra/data/ci/gcr-admin', 'service-account'),
  secret.new('gh_app_id', 'infra/data/ci/grafana-release-eng/grafana-delivery-bot', 'app-id'),
  secret.new('gh_app_installation_id', 'infra/data/ci/grafana-release-eng/grafana-delivery-bot', 'app-installation-id'),
  secret.new('gh_app_private_key', 'infra/data/ci/grafana-release-eng/grafana-delivery-bot', 'app-private-key'),
  secret.new('dockerconfigjson', 'secret/data/common/gcr', '.dockerconfigjson'),

]
