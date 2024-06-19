local drone = import 'github.com/Duologic/drone-libsonnet/main.libsonnet';
local pl = drone.pipeline.docker;
local step = pl.step;
local secret = drone.secret;

local goImage = 'golang:1.22.2';
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

local verifyGenTrigger = {
  event: [
    'pull_request',
  ],
  paths: {
    include: ['pkg/dronereceiver/metadata.yaml'],
  },
};

local checkPackages() = step.new('check-packages', image=goImage)
  + step.withCommands([
      'make tidy-all',
      'git diff -s --exit-code || (echo "Packages are out of date. Run make tidy-all and commit the changes" && git --no-pager diff && exit 1)',
    ]);

local installTools() = step.new('install-tools', image=goImage)
  + step.withDependsOn(['check-packages'])
  + step.withCommands([
      'make install-tools',
    ]);

local codeGen() = step.new('check-codegen', image=goImage)
  + step.withDependsOn(['install-tools'])
  + step.withCommands([
      'make generate',
      'git diff -s --exit-code || (echo "Generated code is out of date. Run make generate and commit the changes" && git --no-pager diff && exit 1)',
    ]);

local crosslinkRun() = step.new('check-crosslink', image=goImage)
  + step.withDependsOn(['install-tools'])
  + step.withCommands([
      'make crosslink',
      'git diff -s --exit-code || (echo "Replace statements not updated. Run make crosslink and commit the changes" && git --no-pager diff && exit 1)'
    ]);

local lint() = step.new('lint', image=goImage)
  + step.withDependsOn(['install-tools'])
  + step.withCommands([
      'make lint-all',
    ]);

local test() = step.new('test', image=goImage)
  + step.withDependsOn(['install-tools'])
  + step.withCommands([
      'make test-all',
    ]);

local build(tag=false) = step.new('build', image=dockerDINDImage)
  + step.withDependsOn(['test'])
  + step.withCommands([
      'docker build'+ (if tag then " --tag us.gcr.io/kubernetes-dev/grafana-ci-otel-collector:${DRONE_COMMIT}" else "") + ' .',
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
  ]);

[
  pl.new('pr')
  + pl.withImagePullSecrets(['dockerconfigjson'])
  + pl.withTrigger(prTrigger)
  + pl.withVolumes([
    dockerVolume,
  ])
  + pl.withSteps([
    checkPackages(),
    installTools(),
    codeGen(),
    crosslinkRun(),
    lint(),
    test(),
    build(),
  ]),

  pl.new('main')
  + pl.withImagePullSecrets(['dockerconfigjson'])
  + pl.withTrigger(mainTrigger)
  + pl.withVolumes([
    dockerVolume,
    dockerDindVolume,
  ])
  + pl.withSteps([
    checkPackages(),
    installTools(),
    codeGen(),
    crosslinkRun(),
    lint(),
    test(),
    build(tag=true),
    step.new('publish-to-gcr', image=dockerDINDImage)
    + step.withDependsOn(['build'])
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
