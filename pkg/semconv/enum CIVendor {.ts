enum CIVendor {
  Drone = 'drone',
}

interface CISystem {
  vendor: CIVendor;
  version: '1.8.0';

  // To check, all the info aboiut the running system
  url: string;
}

enum Status {
  Success = 'success',
}

enum DroneWorkflowKind {
  Build = 'build',
  Stage = 'stage',
  Step = 'step',
}

interface BaseWorkflow {
  vendor: CIVendor;
}

interface BaseDroneWorkflowItem extends BaseWorkflow {
  vendor: CIVendor.Drone;
  kind: DroneWorkflowKind;
  status: Status;
}

enum DroneTrigger {
  PR = 'pr',
  Sync = 'sync',
  Cron = 'cron',
}

interface DroneBuild extends BaseDroneWorkflowItem {
  kind: DroneWorkflowKind.Build;
  id: string;
  number: number;

  /** the build trigger */
  trigger: DroneTrigger;

  /** The VCS resource associated with the build */
  vcs?: {
    name: VCSName;
    source: string;
    target: string;
  };
  /** The User resource associated with the build */
  author?: {
    username: string;
    name: string;
    email: string;
    avatarURL: string;
  };
}

interface DroneStage extends BaseDroneWorkflowItem {
  kind: DroneWorkflowKind.Stage;
  name: string;
  number: number;
}

interface DroneStep extends BaseDroneWorkflowItem {
  kind: DroneWorkflowKind.Step;
  name: string;
  number: number;
}

enum VCSName {
  Git = 'git',
}

interface BaseVCS {
  name: VCSName;
}

interface GitVCS {
  name: VCSName.Git;
  repo: string; // grafana/grafana
  http_url: string; // github.com/...
}
