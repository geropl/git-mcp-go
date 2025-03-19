package pkg

import (
	"context"
	"strings"
	"testing"

	"github.com/geropl/git-mcp-go/pkg/gitops/shell"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiRepositorySupport(t *testing.T) {
	// Create multiple test repositories
	repo1Dir := t.TempDir()
	repo2Dir := t.TempDir()
	repo3Dir := t.TempDir()

	// Initialize the repositories
	gitOps := shell.NewShellGitOperations()

	// Initialize the repos
	_, err := gitOps.InitRepo(repo1Dir)
	require.NoError(t, err, "Failed to initialize repo1")

	_, err = gitOps.InitRepo(repo2Dir)
	require.NoError(t, err, "Failed to initialize repo2")

	_, err = gitOps.InitRepo(repo3Dir)
	require.NoError(t, err, "Failed to initialize repo3")

	t.Run("TestGitListRepositories", func(t *testing.T) {
		// Create a server with multiple repositories
		repoPaths := []string{repo1Dir, repo2Dir, repo3Dir}
		server := NewGitServer(repoPaths, gitOps, false)
		server.RegisterTools()

		// Call the git_list_repositories tool
		request := mcp.CallToolRequest{}
		request.Params.Name = "git_list_repositories"
		request.Params.Arguments = map[string]interface{}{}

		result, err := server.gitListRepositoriesHandler(context.Background(), request)
		require.NoError(t, err, "List repositories handler should not return error")
		require.NotNil(t, result, "Result should not be nil")

		// Verify the result
		if len(result.Content) > 0 {
			text := ""
			if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
				text = textContent.Text
			}

			// Check that the output contains information about all repositories
			assert.Contains(t, text, "Available repositories (3)", "Should show 3 repositories")
			for _, repoPath := range repoPaths {
				assert.Contains(t, text, repoPath, "Output should contain the repository path")
			}
		} else {
			t.Fatalf("No content in result")
		}
	})

	t.Run("TestRepositorySelection", func(t *testing.T) {
		// Create a server with multiple repositories
		repoPaths := []string{repo1Dir, repo2Dir, repo3Dir}
		server := NewGitServer(repoPaths, gitOps, false)

		// Test default repository selection (first repository)
		selectedPath, err := server.getRepoPathForOperation("")
		require.NoError(t, err, "Default repository selection should not error")
		assert.Equal(t, repo1Dir, selectedPath, "Default should be the first repository")

		// Test specific repository selection
		selectedPath, err = server.getRepoPathForOperation(repo2Dir)
		require.NoError(t, err, "Specific repository selection should not error")
		assert.Equal(t, repo2Dir, selectedPath, "Should select the specified repository")

		// Test invalid repository selection
		_, err = server.getRepoPathForOperation("/invalid/path")
		require.Error(t, err, "Invalid repository selection should error")
		assert.Contains(t, err.Error(), "access denied", "Error should mention access denied")
	})

	t.Run("TestOperationsAcrossMultipleRepositories", func(t *testing.T) {
		// Create a server with multiple repositories
		repoPaths := []string{repo1Dir, repo2Dir, repo3Dir}
		server := NewGitServer(repoPaths, gitOps, false)
		server.RegisterTools()

		// Test git_status on different repositories
		// First repository (default)
		request := mcp.CallToolRequest{}
		request.Params.Name = "git_status"
		request.Params.Arguments = map[string]interface{}{}

		result, err := server.gitStatusHandler(context.Background(), request)
		require.NoError(t, err, "Status on default repository should not error")
		if len(result.Content) > 0 {
			text := ""
			if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
				text = textContent.Text
			}
			assert.Contains(t, text, repo1Dir, "Output should reference the first repository")
		}

		// Second repository (explicit)
		request = mcp.CallToolRequest{}
		request.Params.Name = "git_status"
		request.Params.Arguments = map[string]interface{}{
			"repo_path": repo2Dir,
		}

		result, err = server.gitStatusHandler(context.Background(), request)
		require.NoError(t, err, "Status on second repository should not error")
		if len(result.Content) > 0 {
			text := ""
			if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
				text = textContent.Text
			}
			assert.Contains(t, text, repo2Dir, "Output should reference the second repository")
		}

		// Invalid repository (should error)
		request = mcp.CallToolRequest{}
		request.Params.Name = "git_status"
		request.Params.Arguments = map[string]interface{}{
			"repo_path": "/invalid/path",
		}

		result, err = server.gitStatusHandler(context.Background(), request)
		require.NoError(t, err, "Handler should not return error, but result should indicate error")
		if len(result.Content) > 0 {
			text := ""
			if textContent, ok := mcp.AsTextContent(result.Content[0]); ok {
				text = textContent.Text
			}
			assert.True(t, strings.Contains(text, "Repository path error") ||
				strings.Contains(text, "access denied"),
				"Output should indicate repository path error")
		}
	})
}
