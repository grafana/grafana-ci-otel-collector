package semconv

// A CI/CD System
const (
	AttributeCIVendor  = "ci.vendor"
	AttributeCIVersion = "ci.version"
)

const (
	// AttributeCIVendorGHAction = "GitHub Actions"
	// AttributeCIVendorJenkins  = "Jenkins"
	AttributeCIVendorDrone = "Drone"
)

const (
	// Status of the Workflow.
	//
	// Type: Enum
	// Required: No
	// Stability: alpha
	AttributeCIStatus = "ci.component.status"
)

const (
	AttributeDroneWorkflowKind = "ci.drone.workflow.kind" // build | stage | step
	// AttributeGHActionsWorkflowKind = "ci.ghactions.workflow.kind"  build | workflow | job | step // maybe it's not build, but it's the group of all workflows
)

const (
	AttributeCIStatusSkipped               = "skipped"
	AttributeCIStatusBlocked               = "blocked"
	AttributeCIStatusDeclined              = "declined"
	AttributeCIStatusWaitingOnDependencies = "waiting_on_dependencies"
	AttributeCIStatusPending               = "pending"
	AttributeCIStatusRunning               = "running"
	AttributeCIStatusSuccess               = "success"
	AttributeCIStatusFailure               = "failure"
	AttributeCIStatusKilled                = "killed"
	AttributeCIStatusError                 = "error"
)

// VCS Info version control system
const (
	AttributeVCSName = "vcs.name"
)

const (
	AttributeCISNameGit = "git"
	AttributeCISNameSVN = "svn"
)

func GetResourceSemanticConventionAttributeNames() []string {
	return []string{
		AttributeCIStatus,
	}
}
