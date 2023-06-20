local goImage = 'golang:1.20.4';

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

local pipeline(event) =
    {
      kind: 'pipeline',
      type: 'docker',
      name: '%s-pipeline' % event,
      platform: {
        os: 'linux',
        arch: 'amd64',
      },
      trigger: {
        event: [event],
      },
      steps: [
        buildStep(),
        testStep(),
      ],
    };

pipeline('pull_request') + pipeline('push')
