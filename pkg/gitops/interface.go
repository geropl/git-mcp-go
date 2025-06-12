package gitops

// GitOperations defines the interface for Git operations
type GitOperations interface {
	GetStatus(repoPath string) (string, error)
	GetDiffUnstaged(repoPath string) (string, error)
	GetDiffStaged(repoPath string) (string, error)
	GetDiff(repoPath string, target string) (string, error)
	CommitChanges(repoPath string, message string) (string, error)
	AddFiles(repoPath string, files []string) (string, error)
	ResetStaged(repoPath string) (string, error)
	GetLog(repoPath string, maxCount int) ([]string, error)
	CreateBranch(repoPath string, branchName string, baseBranch string) (string, error)
	CheckoutBranch(repoPath string, branchName string) (string, error)
	InitRepo(repoPath string) (string, error)
	ShowCommit(repoPath string, revision string) (string, error)
	PushChanges(repoPath string, remote string, branch string) (string, error)
	ApplyPatchFromString(repoPath string, patchString string) (string, error)
	ApplyPatchFromFile(repoPath string, patchFilePath string) (string, error)
}
