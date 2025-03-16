package pkg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geropl/git-mcp-go/pkg/gitops"
	"github.com/geropl/git-mcp-go/pkg/gitops/gogit"
	"github.com/geropl/git-mcp-go/pkg/gitops/shell"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// initRepos initializes a remote repo and creates a local clone
func initRepos(t *testing.T, remoteDir, localDir string) {
	// Initialize bare repository in remoteDir
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	require.NoError(t, cmd.Run())

	// Clone the remote repository to localDir
	cmd = exec.Command("git", "clone", remoteDir, localDir)
	require.NoError(t, cmd.Run())

	// Set up git config for the test
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = localDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = localDir
	require.NoError(t, cmd.Run())
}

// createCommit creates a file and commits it
func createCommit(t *testing.T, repoDir, filename, content, message string) {
	filePath := filepath.Join(repoDir, filename)
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	cmd := exec.Command("git", "add", filename)
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoDir
	require.NoError(t, cmd.Run())
}

func TestGitOperations(t *testing.T) {
	// Test cases table
	testCases := []struct {
		name           string
		setupFunc      func(t *testing.T, remoteRepo, localRepo string) // Setup repositories
		action         string                                           // MCP action to run
		params         map[string]interface{}                           // Parameters for the action
		expectedResult func(t *testing.T, result string, remoteDir string, err error) // Validation function
	}{
		{
			name: "basic_push",
			setupFunc: func(t *testing.T, remoteRepo, localRepo string) {
				initRepos(t, remoteRepo, localRepo)
				createCommit(t, localRepo, "test.txt", "test content", "Initial commit")
				
				// Get the current branch name
				cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
				cmd.Dir = localRepo
				output, err := cmd.Output()
				require.NoError(t, err)
				branch := strings.TrimSpace(string(output))
				
				// Set the branch parameter
				t.Logf("Current branch: %s", branch)
			},
			action: "git_push",
			params: map[string]interface{}{
				"remote": "origin",
				// We'll use the current branch name from the repository
				// This will be determined at runtime
			},
			expectedResult: func(t *testing.T, result string, remoteDir string, err error) {
				require.NoError(t, err)
				require.Contains(t, result, "Successfully pushed")

				// Verify the commit exists in the remote repository
				cmd := exec.Command("git", "log", "--oneline")
				cmd.Dir = remoteDir
				output, err := cmd.Output()
				require.NoError(t, err)
				require.Contains(t, string(output), "Initial commit")
			},
		},
		{
			name: "push_multiple_commits",
			setupFunc: func(t *testing.T, remoteRepo, localRepo string) {
				initRepos(t, remoteRepo, localRepo)
				createCommit(t, localRepo, "file1.txt", "content 1", "First commit")
				createCommit(t, localRepo, "file2.txt", "content 2", "Second commit")
				createCommit(t, localRepo, "file3.txt", "content 3", "Third commit")
			},
			action: "git_push",
			params: map[string]interface{}{},
			expectedResult: func(t *testing.T, result string, remoteDir string, err error) {
				require.NoError(t, err)
				require.Contains(t, result, "Successfully pushed")

				// Verify all commits exist in the remote repository
				cmd := exec.Command("git", "log", "--oneline")
				cmd.Dir = remoteDir
				output, err := cmd.Output()
				require.NoError(t, err)
				require.Contains(t, string(output), "First commit")
				require.Contains(t, string(output), "Second commit")
				require.Contains(t, string(output), "Third commit")
			},
		},
		{
			name: "push_different_branch",
			setupFunc: func(t *testing.T, remoteRepo, localRepo string) {
				initRepos(t, remoteRepo, localRepo)
				createCommit(t, localRepo, "main.txt", "main content", "Main branch commit")

				// Create and switch to a new branch
				cmd := exec.Command("git", "checkout", "-b", "feature")
				cmd.Dir = localRepo
				require.NoError(t, cmd.Run())

				createCommit(t, localRepo, "feature.txt", "feature content", "Feature branch commit")
			},
			action: "git_push",
			params: map[string]interface{}{
				"remote": "origin",
				"branch": "feature",
			},
			expectedResult: func(t *testing.T, result string, remoteDir string, err error) {
				require.NoError(t, err)
				require.Contains(t, result, "Successfully pushed")

				// Verify the feature branch exists in the remote repository
				cmd := exec.Command("git", "ls-remote", "--heads", remoteDir)
				output, err := cmd.Output()
				require.NoError(t, err)
				require.Contains(t, string(output), "refs/heads/feature")

				// Create a temporary directory to check the remote branch
				tempDir := t.TempDir()
				
				// Clone the remote repository to the temp directory
				cmd = exec.Command("git", "clone", "--branch", "feature", remoteDir, tempDir)
				require.NoError(t, cmd.Run())
				
				// Verify the commit exists in the feature branch
				cmd = exec.Command("git", "log", "--oneline")
				cmd.Dir = tempDir
				output, err = cmd.Output()
				require.NoError(t, err)
				require.Contains(t, string(output), "Feature branch commit")
			},
		},
		{
			name: "push_no_changes",
			setupFunc: func(t *testing.T, remoteRepo, localRepo string) {
				initRepos(t, remoteRepo, localRepo)
				createCommit(t, localRepo, "test.txt", "test content", "Initial commit")

				// Get the current branch name
				cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
				cmd.Dir = localRepo
				output, err := cmd.Output()
				require.NoError(t, err)
				branch := strings.TrimSpace(string(output))

				// Push the commit
				cmd = exec.Command("git", "push", "origin", branch)
				cmd.Dir = localRepo
				require.NoError(t, cmd.Run())
			},
			action: "git_push",
			params: map[string]interface{}{},
			expectedResult: func(t *testing.T, result string, remoteDir string, err error) {
				require.NoError(t, err)
				require.Contains(t, result, "up-to-date")
			},
		},
	}

	// Run each test case in both modes
	modes := []string{"shell", "go-git"}

	for _, mode := range modes {
		for _, tc := range testCases {
			t.Run(fmt.Sprintf("%s_%s", tc.name, mode), func(t *testing.T) {
				// Create temporary directories for repositories
				remoteDir := t.TempDir()
				localDir := t.TempDir()

				// Setup repositories
				tc.setupFunc(t, remoteDir, localDir)

				// Create appropriate GitOperations implementation based on mode
				var gitOps gitops.GitOperations
				if mode == "shell" {
					gitOps = shell.NewShellGitOperations()
				} else {
					gitOps = gogit.NewGoGitOperations()
				}

				// Create server with local repository
				server := NewGitServer([]string{localDir}, gitOps, true) // Enable write access for tests
				server.RegisterTools()

				// Execute the action and validate results
				var result *mcp.CallToolResult
				var err error

				// Copy the parameters and add the repo_path if not present
				params := make(map[string]interface{})
				for k, v := range tc.params {
					params[k] = v
				}
				if _, ok := params["repo_path"]; !ok {
					params["repo_path"] = localDir
				}

				switch tc.action {
				case "git_push":
					request := mcp.CallToolRequest{}
					request.Params.Name = "git_push"
					request.Params.Arguments = params
					result, err = server.gitPushHandler(context.Background(), request)
				// Add other actions as needed
				default:
					t.Fatalf("Unknown action: %s", tc.action)
				}

				// Validate the results
				if result != nil && len(result.Content) > 0 {
					text := ""
					if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
						text = textContent.Text
					}
					tc.expectedResult(t, text, remoteDir, err)
				} else {
					tc.expectedResult(t, "", remoteDir, err)
				}
			})
		}
	}
}
