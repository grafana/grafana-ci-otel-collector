package semconv

// A CI/CD System
const (
	// CI/CD system vendor.
	//
	// Type: Enum
	// Required: No
	// Stability: alpha
	AttributeCIVendor = "ci.vendor"

	// CI/CD system version string.
	//
	// Type: string
	// Required: No
	// Stability: alpha
	AttributeCIVersion = "ci.version"
)

// CI/CD system vendor enum
const (
	AttributeCIVendorDrone = "drone"
)

const (
	// Status of the Workflow.
	//
	// TODO: to change the wollowing in enum in the future
	// Type: string
	// Required: No
	// Stability: alpha
	AttributeCIWorkflowItemStatus = "ci.workflow_item.status"
)

const (
	// Drone workflow item kind.
	//
	// Type: Enum
	// Required: No
	// Stability: alpha
	AttributeDroneWorkflowItemKind = "ci.drone.workflow_item.kind" // build | stage | step
	// Drone workflow event Indicatates which event triggeed the workflow.
	//
	// TODO: to change the wollowing in enum in the future
	// Type: string
	// Required: No
	// Stability: alpha
	// Examples: push, pull_request, custom
	AttributeDroneWorkflowEvent = "ci.drone.workflow.event"
	// Drone workflow title.
	//
	// Type: string
	// Required: No
	// Stability: alpha
	AttributeDroneWorkflowTitle = "ci.drone.workflow.title"
)

// Drone workflow item kind enum.
const (
	AttributeDroneWorkflowItemKindBuild = "build"
	AttributeDroneWorkflowItemKindStage = "stage"
	AttributeDroneWorkflowItemKindStep  = "step"
)

// Drone build info
const (
	// Drone build id.
	//
	// Type: number
	// Required: No
	// Stability: alpha
	AttributeDroneBuildID = "ci.drone.build.id"
	// Drone build number.
	//
	// Type: number
	// Required: No
	// Stability: alpha
	AttributeDroneBuildNumber = "ci.drone.build.number"
	// Drone build message.
	//
	// Type: number
	// Required: No
	// Stability: alpha
	AttributeDroneBuildMessage = "ci.drone.build.message"

	// Experimental attributes
	AttributesDroneBuildBefore = "ci.drone.build.before"
	AttributesDroneBuildAfter  = "ci.drone.build.after"
	AttributesDroneBuildSource = "ci.drone.build.source"
	AttributesDroneBuildTarget = "ci.drone.build.target"
	AttributesDroneBuildRef    = "ci.drone.build.ref"
	AttributesDroneBuildLink   = "ci.drone.build.link"
	AttributesDroneBuildParent = "ci.drone.build.parent"
)

// Drone stage info
const (
	// Drone stage id.
	//
	// Type: number
	// Required: No
	// Stability: alpha
	AttributeDroneStageID = "ci.drone.stage.id"
	// Drone stage number.
	//
	// Type: number
	// Required: No
	// Stability: alpha
	AttributeDroneStageNumber = "ci.drone.stage.number"
	// Drone stage name.
	//
	// Type: number
	// Required: No
	// Stability: alpha
	AttributeDroneStageName = "ci.drone.stage.name"
)

// Drone step info
const (
	// Drone stage id.
	//
	// Type: number
	// Required: No
	// Stability: alpha
	AttributeDroneStepID = "ci.drone.step.id"
	// Drone build number.
	//
	// Type: number
	// Required: No
	// Stability: alpha
	AttributeDroneStepNumber = "ci.drone.step.number"
	// Drone build message.
	//
	// Type: number
	// Required: No
	// Stability: alpha
	AttributeDroneStepName = "ci.drone.step.name"
)

// VCS Info
const (
	// VCS Type
	//
	// Type: Enum
	// Requirement Level: Optional
	// Stability: alpha
	AttributeVCSType = "vcs.type"
)

// VCS Type Enum
const (
	AttributeVCSTypeGit = "git"
)

// Git repository info
const (
	// Git HTTP URL
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: alpha
	AttributeGitHTTPURL = "git.url.http"
	// Git SSH URL
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: alpha
	AttributeGitSSHURL = "git.url.ssh"
	// Git web URL
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: alpha
	AttributeGitWWWURL = "git.url.www"
	// Git repository name (org/slug)
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: alpha
	AttributeGitRepoName = "git.repo.name"
	// Git repository branch name
	//
	// Type: string
	// Requirement Level: Optional
	// Stability: alpha
	AttributeGitBranchName = "git.branch.name"
)
