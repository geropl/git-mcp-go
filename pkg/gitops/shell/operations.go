package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/geropl/git-mcp-go/pkg/gitops"
)

// ShellGitOperations implements GitOperations using git CLI commands
type ShellGitOperations struct{}

// NewShellGitOperations creates a new ShellGitOperations instance
func NewShellGitOperations() *ShellGitOperations {
	return &ShellGitOperations{}
}

// GetStatus returns the status of the working tree
func (s *ShellGitOperations) GetStatus(repoPath string) (string, error) {
	return gitops.RunGitCommand(repoPath, "status")
}

// GetDiffUnstaged returns the diff of unstaged changes
func (s *ShellGitOperations) GetDiffUnstaged(repoPath string) (string, error) {
	return gitops.RunGitCommand(repoPath, "diff")
}

// GetDiffStaged returns the diff of staged changes
func (s *ShellGitOperations) GetDiffStaged(repoPath string) (string, error) {
	return gitops.RunGitCommand(repoPath, "diff", "--cached")
}

// GetDiff returns the diff between the current state and a target
func (s *ShellGitOperations) GetDiff(repoPath string, target string) (string, error) {
	return gitops.RunGitCommand(repoPath, "diff", target)
}

// CommitChanges commits the staged changes
func (s *ShellGitOperations) CommitChanges(repoPath string, message string) (string, error) {
	output, err := gitops.RunGitCommand(repoPath, "commit", "-m", message)
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}
	return output, nil
}

// AddFiles adds files to the staging area
func (s *ShellGitOperations) AddFiles(repoPath string, files []string) (string, error) {
	args := append([]string{"add"}, files...)
	_, err := gitops.RunGitCommand(repoPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to add files: %w", err)
	}
	return "Files staged successfully", nil
}

// ResetStaged unstages all staged changes
func (s *ShellGitOperations) ResetStaged(repoPath string) (string, error) {
	_, err := gitops.RunGitCommand(repoPath, "reset")
	if err != nil {
		return "", fmt.Errorf("failed to reset staged changes: %w", err)
	}
	return "All staged changes reset", nil
}

// GetLog returns the commit history
func (s *ShellGitOperations) GetLog(repoPath string, maxCount int) ([]string, error) {
	args := []string{"log", "--pretty=format:Commit: %H%nAuthor: %an <%ae>%nDate: %ad%nMessage: %s%n"}
	if maxCount > 0 {
		args = append(args, fmt.Sprintf("-n%d", maxCount))
	}

	output, err := gitops.RunGitCommand(repoPath, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}

	// Split the output into individual commit entries
	logs := strings.Split(strings.TrimSpace(output), "\n\n")
	return logs, nil
}

// CreateBranch creates a new branch
func (s *ShellGitOperations) CreateBranch(repoPath string, branchName string, baseBranch string) (string, error) {
	args := []string{"branch", branchName}
	if baseBranch != "" {
		args = append(args, baseBranch)
	}

	_, err := gitops.RunGitCommand(repoPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to create branch: %w", err)
	}

	baseRef := baseBranch
	if baseRef == "" {
		// Get the current branch name
		currentBranch, err := gitops.RunGitCommand(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			baseRef = "HEAD"
		} else {
			baseRef = strings.TrimSpace(currentBranch)
		}
	}

	return fmt.Sprintf("Created branch '%s' from '%s'", branchName, baseRef), nil
}

// CheckoutBranch switches to a branch
func (s *ShellGitOperations) CheckoutBranch(repoPath string, branchName string) (string, error) {
	_, err := gitops.RunGitCommand(repoPath, "checkout", branchName)
	if err != nil {
		return "", fmt.Errorf("failed to checkout branch: %w", err)
	}

	return fmt.Sprintf("Switched to branch '%s'", branchName), nil
}

// InitRepo initializes a new Git repository
func (s *ShellGitOperations) InitRepo(repoPath string) (string, error) {
	// Create directory if it doesn't exist
	err := os.MkdirAll(repoPath, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	_, err = gitops.RunGitCommand(repoPath, "init")
	if err != nil {
		return "", fmt.Errorf("failed to initialize repository: %w", err)
	}

	gitDir := filepath.Join(repoPath, ".git")
	return fmt.Sprintf("Initialized empty Git repository in %s", gitDir), nil
}

// ShowCommit shows the contents of a commit
func (s *ShellGitOperations) ShowCommit(repoPath string, revision string) (string, error) {
	return gitops.RunGitCommand(repoPath, "show", revision)
}

// PushChanges pushes local commits to a remote repository
func (s *ShellGitOperations) PushChanges(repoPath string, remote string, branch string) (string, error) {
	args := []string{"push"}
	if remote != "" {
		args = append(args, remote)
	}
	if branch != "" {
		args = append(args, branch)
	}

	output, err := gitops.RunGitCommand(repoPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to push changes: %w", err)
	}

	// Check if the output indicates that everything is up-to-date
	if strings.Contains(output, "up-to-date") {
		return output, nil
	}

	// Format the output to match the expected format
	return fmt.Sprintf("Successfully pushed to %s/%s\n%s",
		remote,
		branch,
		output), nil
}
