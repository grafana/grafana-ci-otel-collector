local goImage = 'golang:1.20.4';

local prTrigger = {
  event: [
    'pull_request',
  ],
};

local mainTrigger = {
  branch: 'main',
  event: [
    'push',
  ],
};

local buildStep() =
    {
        name: 'build',
        image: goImage,
        commands: [
            'make build'
        ],
    };

local testStep() =
    {
        name: 'test',
        image: goImage,
        commands: [
            'go test ./pkg/dronereceiver'
        ],
    };

local pipeline(trigger) =
    [{
      kind: 'pipeline',
      type: 'docker',
      name: '%s-pipeline' % trigger.event,
      platform: {
        os: 'linux',
        arch: 'amd64',
      },
      trigger: trigger,
      steps: [
        buildStep(),
        testStep(),
      ],
    }];

pipeline(prTrigger) + pipeline(mainTrigger)
