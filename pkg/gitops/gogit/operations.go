package gogit

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/geropl/git-mcp-go/pkg/gitops"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GoGitOperations implements GitOperations using the go-git library
type GoGitOperations struct{}

// NewGoGitOperations creates a new GoGitOperations instance
func NewGoGitOperations() *GoGitOperations {
	return &GoGitOperations{}
}

// GetStatus returns the status of the working tree
func (g *GoGitOperations) GetStatus(repoPath string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	return status.String(), nil
}

// GetDiffUnstaged returns the diff of unstaged changes
func (g *GoGitOperations) GetDiffUnstaged(repoPath string) (string, error) {
	// go-git doesn't have a direct equivalent to git diff
	// We'll use git command for this operation
	return gitops.RunGitCommand(repoPath, "diff")
}

// GetDiffStaged returns the diff of staged changes
func (g *GoGitOperations) GetDiffStaged(repoPath string) (string, error) {
	// go-git doesn't have a direct equivalent to git diff --cached
	// We'll use git command for this operation
	return gitops.RunGitCommand(repoPath, "diff", "--cached")
}

// GetDiff returns the diff between the current state and a target
func (g *GoGitOperations) GetDiff(repoPath string, target string) (string, error) {
	// go-git doesn't have a direct equivalent to git diff with target
	// We'll use git command for this operation
	return gitops.RunGitCommand(repoPath, "diff", target)
}

// CommitChanges commits the staged changes
func (g *GoGitOperations) CommitChanges(repoPath string, message string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	commit, err := wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "MCP Git Server",
			Email: "mcp-git@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	return fmt.Sprintf("Changes committed successfully with hash %s", commit.String()), nil
}

// AddFiles adds files to the staging area
func (g *GoGitOperations) AddFiles(repoPath string, files []string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	for _, file := range files {
		_, err := wt.Add(file)
		if err != nil {
			return "", fmt.Errorf("failed to add file %s: %w", file, err)
		}
	}

	return "Files staged successfully", nil
}

// ResetStaged unstages all staged changes
func (g *GoGitOperations) ResetStaged(repoPath string) (string, error) {
	// go-git doesn't have a direct equivalent to git reset
	// We'll use git command for this operation
	_, err := gitops.RunGitCommand(repoPath, "reset")
	if err != nil {
		return "", fmt.Errorf("failed to reset staged changes: %w", err)
	}
	return "All staged changes reset", nil
}

// GetLog returns the commit history
func (g *GoGitOperations) GetLog(repoPath string, maxCount int) ([]string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get the HEAD reference
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get the commit object for the HEAD reference
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	// Create a commit iterator
	commitIter, err := repo.Log(&git.LogOptions{From: commit.Hash})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit iterator: %w", err)
	}

	// Collect commits
	var logs []string
	count := 0
	err = commitIter.ForEach(func(c *object.Commit) error {
		if maxCount > 0 && count >= maxCount {
			return fmt.Errorf("stop iteration")
		}

		log := fmt.Sprintf("Commit: %s\nAuthor: %s\nDate: %s\nMessage: %s\n",
			c.Hash.String(),
			c.Author.String(),
			c.Author.When.String(),
			c.Message)

		logs = append(logs, log)
		count++
		return nil
	})

	if err != nil && err.Error() != "stop iteration" {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return logs, nil
}

// CreateBranch creates a new branch
func (g *GoGitOperations) CreateBranch(repoPath string, branchName string, baseBranch string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	var baseRef *plumbing.Reference
	if baseBranch != "" {
		// Get the base branch reference
		baseRef, err = repo.Reference(plumbing.NewBranchReferenceName(baseBranch), true)
		if err != nil {
			return "", fmt.Errorf("failed to get base branch reference: %w", err)
		}
	} else {
		// Use HEAD as base
		baseRef, err = repo.Head()
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD: %w", err)
		}
	}

	// Create the new branch
	branchRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branchName), baseRef.Hash())
	err = repo.Storer.SetReference(branchRef)
	if err != nil {
		return "", fmt.Errorf("failed to create branch: %w", err)
	}

	baseName := baseBranch
	if baseName == "" {
		if baseRef.Name().IsBranch() {
			baseName = baseRef.Name().Short()
		} else {
			baseName = baseRef.Hash().String()
		}
	}

	return fmt.Sprintf("Created branch '%s' from '%s'", branchName, baseName), nil
}

// CheckoutBranch switches to a branch
func (g *GoGitOperations) CheckoutBranch(repoPath string, branchName string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to checkout branch: %w", err)
	}

	return fmt.Sprintf("Switched to branch '%s'", branchName), nil
}

// InitRepo initializes a new Git repository
func (g *GoGitOperations) InitRepo(repoPath string) (string, error) {
	// Create directory if it doesn't exist
	err := os.MkdirAll(repoPath, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Initialize repository
	_, err = git.PlainInit(repoPath, false)
	if err != nil {
		return "", fmt.Errorf("failed to initialize repository: %w", err)
	}

	gitDir := filepath.Join(repoPath, ".git")
	return fmt.Sprintf("Initialized empty Git repository in %s", gitDir), nil
}

// ShowCommit shows the contents of a commit
func (g *GoGitOperations) ShowCommit(repoPath string, revision string) (string, error) {
	// go-git doesn't have a direct equivalent to git show
	// We'll use git command for this operation
	return gitops.RunGitCommand(repoPath, "show", revision)
}

// PushChanges pushes local commits to a remote repository
func (g *GoGitOperations) PushChanges(repoPath string, remote string, branch string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Use "origin" as default remote if not specified
	if remote == "" {
		remote = "origin"
	}

	// Determine refspec based on branch
	var refspec string
	var branchName string
	if branch == "" {
		// Get current branch
		head, err := repo.Head()
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD: %w", err)
		}
		if !head.Name().IsBranch() {
			return "", fmt.Errorf("HEAD is not a branch")
		}
		refspec = head.Name().String()
		branchName = head.Name().Short()
	} else {
		refspec = plumbing.NewBranchReferenceName(branch).String()
		branchName = branch
	}

	// Push to remote
	err = repo.Push(&git.PushOptions{
		RemoteName: remote,
		RefSpecs:   []config.RefSpec{config.RefSpec(refspec + ":" + refspec)},
	})

	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			return "Everything up-to-date", nil
		}
		return "", fmt.Errorf("failed to push: %w", err)
	}

	return fmt.Sprintf("Successfully pushed to %s/%s", remote, branchName), nil
}
