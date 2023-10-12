package semconv

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
)
