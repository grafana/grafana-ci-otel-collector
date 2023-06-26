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

local verifyGenStep() =
    {
        name: 'verify-gen',
        image: goImage,
        commands: [
            'make metadata',
            'if [ -n "$(git status --porcelain)" ]; then echo "ERROR: Please run make metadata and commit your changes." && git diff --exit-code; fi',
        ],
    };

local verifyGenPipeline() =
    [{
      kind: 'pipeline',
      type: 'docker',
      name: 'verify-gen-pipeline' ,
      platform: {
        os: 'linux',
        arch: 'amd64',
      },
      paths: {
        include: ['pkg/dronereceiver/metadata.yaml'],
      },
      trigger: prTrigger,
      steps: [
        verifyGenStep(),
      ],
    }];

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

pipeline(prTrigger) + pipeline(mainTrigger) + pipeline(customTrigger) + verifyGenPipeline()
