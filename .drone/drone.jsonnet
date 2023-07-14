local drone = import 'github.com/Duologic/drone-libsonnet/main.libsonnet';
local pl = drone.pipeline.docker;
local step = pl.step;

local goImage = 'golang:1.20.4';
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

[
  pl.new('pr')
  + pl.withImagePullSecrets(['dockerconfigjson'])
  + pl.withTrigger(prTrigger)
  + pl.withSteps([
    step.new('build', image=goImage)
    + step.withCommands([
      'make build',
    ]),
    step.new('test', image=goImage)
    + step.withCommands([
      'go test ./pkg/dronereceiver',
    ]),
  ]),
  pl.new('custom')
  + pl.withImagePullSecrets(['dockerconfigjson'])
  + pl.withTrigger(customTrigger)
  + pl.withSteps([
    step.new('step1', image=goImage)
    + step.withCommands([
      'echo step1',
    ]),
    step.new('step2', image=goImage)
    + step.withCommands([
      'echo step2',
    ]),
  ]),
  pl.new('main')
  + pl.withImagePullSecrets(['dockerconfigjson'])
  + pl.withTrigger(mainTrigger)
  + pl.withSteps([
    step.new('build', image=goImage)
    + step.withCommands([
      'make build',
    ]),
    step.new('test', image=goImage)
    + step.withCommands([
      'go test ./pkg/dronereceiver',
    ]),
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
]
