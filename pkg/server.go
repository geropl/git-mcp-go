package pkg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/geropl/git-mcp-go/pkg/gitops"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GitServer represents the Git MCP server
type GitServer struct {
	server      *server.MCPServer
	repoPaths   []string // Changed from single string to array of strings
	gitOps      gitops.GitOperations
	writeAccess bool
}

// NewGitServer creates a new Git MCP server
func NewGitServer(repoPaths []string, gitOps gitops.GitOperations, writeAccess bool) *GitServer {
	s := server.NewMCPServer(
		"Git MCP Server",
		"1.0.0",
	)

	// Normalize repository paths
	normalizedPaths := make([]string, 0, len(repoPaths))
	for _, path := range repoPaths {
		if path == "" {
			continue
		}
		
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to resolve path %s: %v\n", path, err)
			continue
		}
		
		// Check if it's a git repository
		gitDirPath := filepath.Join(absPath, ".git")
		if info, err := os.Stat(gitDirPath); err == nil && info.IsDir() {
			normalizedPaths = append(normalizedPaths, absPath)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: not a git repository: %s\n", absPath)
		}
	}

	return &GitServer{
		server:      s,
		repoPaths:   normalizedPaths,
		gitOps:      gitOps,
		writeAccess: writeAccess,
	}
}

// isPathInAllowedRepos checks if a path is within any of the allowed repositories
func (s *GitServer) isPathInAllowedRepos(path string) bool {
	// Ensure path is absolute and clean
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Check if the path is within any of the allowed repositories
	for _, repoPath := range s.repoPaths {
		if strings.HasPrefix(absPath, repoPath) {
			return true
		}
	}
	return false
}

// validateRepoPath validates and normalizes a repository path
func (s *GitServer) validateRepoPath(requestedPath string) (string, error) {
	// If no specific path is provided, but we have repositories configured
	if requestedPath == "" {
		if len(s.repoPaths) > 0 {
			// Use the first repository as default
			return s.repoPaths[0], nil
		}
		return "", fmt.Errorf("no repository specified and no defaults configured")
	}

	// Always convert to absolute path first
	absPath, err := filepath.Abs(requestedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check if path is within allowed repositories
	if !s.isPathInAllowedRepos(absPath) {
		return "", fmt.Errorf(
			"access denied - path outside allowed repositories: %s",
			absPath,
		)
	}

	// Ensure it's a valid git repository
	gitDirPath := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitDirPath); os.IsNotExist(err) {
		return "", fmt.Errorf("not a git repository: %s", absPath)
	}

	return absPath, nil
}

// getRepoPathForOperation determines which repo path to use for an operation
func (s *GitServer) getRepoPathForOperation(requestedPath string) (string, error) {
	return s.validateRepoPath(requestedPath)
}

func GetReadOnlyToolNames() map[string]bool {
	return map[string]bool{
		"git_status":        true,
		"git_diff_unstaged": true,
		"git_diff_staged":   true,
		"git_diff":          true,
		"git_log":           true,
		"git_show":          true,
	}
}

func GetLocalOnlyToolNames() map[string]bool {
	// local tools that alter state, complementing the read-only tools
	result := map[string]bool{
		"git_init":          true,
		"git_create_branch": true,
		"git_checkout":      true,
		"git_commit":        true,
		"git_add":           true,
		"git_reset":         true,
	}

	for toolName := range GetReadOnlyToolNames() {
		result[toolName] = true
	}
	return result
}

// RegisterTools registers all Git tools with the MCP server
func (s *GitServer) RegisterTools() {
	// Register git_status tool
	var repoPathDesc string

	if len(s.repoPaths) == 0 {
		repoPathDesc = "Path to Git repository"
		s.server.AddTool(mcp.NewTool("git_status",
			mcp.WithDescription("Shows the working tree status"),
			mcp.WithString("repo_path",
				mcp.Required(),
				mcp.Description(repoPathDesc),
			),
		), s.gitStatusHandler)
	} else {
		defaultRepo := s.repoPaths[0]
		if len(s.repoPaths) == 1 {
			repoPathDesc = fmt.Sprintf("Path to Git repository (default: %s)", defaultRepo)
		} else {
			repoPathDesc = fmt.Sprintf("Path to Git repository (default: %s, %d repositories available)", defaultRepo, len(s.repoPaths))
		}
		s.server.AddTool(mcp.NewTool("git_status",
			mcp.WithDescription("Shows the working tree status"),
			mcp.WithString("repo_path",
				mcp.Description(repoPathDesc),
			),
		), s.gitStatusHandler)
	}

	// Register git_diff_unstaged tool
	if len(s.repoPaths) == 0 {
		s.server.AddTool(mcp.NewTool("git_diff_unstaged",
			mcp.WithDescription("Shows changes in the working directory that are not yet staged"),
			mcp.WithString("repo_path",
				mcp.Required(),
				mcp.Description(repoPathDesc),
			),
		), s.gitDiffUnstagedHandler)
	} else {
		s.server.AddTool(mcp.NewTool("git_diff_unstaged",
			mcp.WithDescription("Shows changes in the working directory that are not yet staged"),
			mcp.WithString("repo_path",
				mcp.Description(repoPathDesc),
			),
		), s.gitDiffUnstagedHandler)
	}

	// Register git_diff_staged tool
	if len(s.repoPaths) == 0 {
		s.server.AddTool(mcp.NewTool("git_diff_staged",
			mcp.WithDescription("Shows changes that are staged for commit"),
			mcp.WithString("repo_path",
				mcp.Required(),
				mcp.Description(repoPathDesc),
			),
		), s.gitDiffStagedHandler)
	} else {
		s.server.AddTool(mcp.NewTool("git_diff_staged",
			mcp.WithDescription("Shows changes that are staged for commit"),
			mcp.WithString("repo_path",
				mcp.Description(repoPathDesc),
			),
		), s.gitDiffStagedHandler)
	}

	// Register git_diff tool
	diffTool := mcp.NewTool("git_diff",
		mcp.WithDescription("Shows differences between branches or commits"),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("Path to Git repository"),
		),
		mcp.WithString("target",
			mcp.Required(),
			mcp.Description("Target branch or commit to compare with"),
		),
	)
	s.server.AddTool(diffTool, s.gitDiffHandler)

	// Register git_commit tool
	commitTool := mcp.NewTool("git_commit",
		mcp.WithDescription("Records changes to the repository"),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("Path to Git repository"),
		),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("Commit message"),
		),
	)
	s.server.AddTool(commitTool, s.gitCommitHandler)

	// Register git_add tool
	addTool := mcp.NewTool("git_add",
		mcp.WithDescription("Adds file contents to the staging area"),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("Path to Git repository"),
		),
		// Note: mcp-go doesn't have WithStringArray, so we'll use a string and parse it
		mcp.WithString("files",
			mcp.Required(),
			mcp.Description("Comma-separated list of file paths to stage"),
		),
	)
	s.server.AddTool(addTool, s.gitAddHandler)

	// Register git_reset tool
	resetTool := mcp.NewTool("git_reset",
		mcp.WithDescription("Unstages all staged changes"),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("Path to Git repository"),
		),
	)
	s.server.AddTool(resetTool, s.gitResetHandler)

	// Register git_log tool
	logTool := mcp.NewTool("git_log",
		mcp.WithDescription("Shows the commit logs"),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("Path to Git repository"),
		),
		mcp.WithNumber("max_count",
			mcp.Description("Maximum number of commits to show (default: 10)"),
		),
	)
	s.server.AddTool(logTool, s.gitLogHandler)

	// Register git_create_branch tool
	createBranchTool := mcp.NewTool("git_create_branch",
		mcp.WithDescription("Creates a new branch from an optional base branch"),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("Path to Git repository"),
		),
		mcp.WithString("branch_name",
			mcp.Required(),
			mcp.Description("Name of the new branch"),
		),
		mcp.WithString("base_branch",
			mcp.Description("Starting point for the new branch"),
		),
	)
	s.server.AddTool(createBranchTool, s.gitCreateBranchHandler)

	// Register git_checkout tool
	checkoutTool := mcp.NewTool("git_checkout",
		mcp.WithDescription("Switches branches"),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("Path to Git repository"),
		),
		mcp.WithString("branch_name",
			mcp.Required(),
			mcp.Description("Name of branch to checkout"),
		),
	)
	s.server.AddTool(checkoutTool, s.gitCheckoutHandler)

	// Register git_show tool
	showTool := mcp.NewTool("git_show",
		mcp.WithDescription("Shows the contents of a commit"),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("Path to Git repository"),
		),
		mcp.WithString("revision",
			mcp.Required(),
			mcp.Description("The revision (commit hash, branch name, tag) to show"),
		),
	)
	s.server.AddTool(showTool, s.gitShowHandler)

	// Register git_init tool
	initTool := mcp.NewTool("git_init",
		mcp.WithDescription("Initialize a new Git repository"),
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("Path to directory to initialize git repo"),
		),
	)
	s.server.AddTool(initTool, s.gitInitHandler)

	// Register git_list_repositories tool
	s.server.AddTool(mcp.NewTool("git_list_repositories",
		mcp.WithDescription("Lists all available Git repositories"),
	), s.gitListRepositoriesHandler)

	if s.writeAccess {
		// Register git_push tool
		pushTool := mcp.NewTool("git_push",
			mcp.WithDescription("Pushes local commits to a remote repository (requires --write-access flag)"),
			mcp.WithString("repo_path",
				mcp.Required(),
				mcp.Description("Path to Git repository"),
			),
			mcp.WithString("remote",
				mcp.Description("Remote name (default: origin)"),
			),
			mcp.WithString("branch",
				mcp.Description("Branch name to push (default: current branch)"),
			),
		)
		s.server.AddTool(pushTool, s.gitPushHandler)
	}
}

// Serve starts the MCP server
func (s *GitServer) Serve() error {
	return server.ServeStdio(s.server)
}

// Tool handlers

func (s *GitServer) gitStatusHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	repoPath, err := s.getRepoPathForOperation(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
	}

	status, err := s.gitOps.GetStatus(repoPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get status: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Repository status for %s:\n%s", repoPath, status)), nil
}

func (s *GitServer) gitDiffUnstagedHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	repoPath, err := s.getRepoPathForOperation(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
	}

	diff, err := s.gitOps.GetDiffUnstaged(repoPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get unstaged diff: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Unstaged changes for %s:\n%s", repoPath, diff)), nil
}

func (s *GitServer) gitDiffStagedHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	repoPath, err := s.getRepoPathForOperation(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
	}

	diff, err := s.gitOps.GetDiffStaged(repoPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get staged diff: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Staged changes for %s:\n%s", repoPath, diff)), nil
}

func (s *GitServer) gitDiffHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	repoPath, err := s.getRepoPathForOperation(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
	}

	target, ok := request.Params.Arguments["target"].(string)
	if !ok {
		return mcp.NewToolResultError("target must be a string"), nil
	}

	diff, err := s.gitOps.GetDiff(repoPath, target)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get diff: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Diff with %s for %s:\n%s", target, repoPath, diff)), nil
}

func (s *GitServer) gitCommitHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	repoPath, err := s.getRepoPathForOperation(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
	}

	message, ok := request.Params.Arguments["message"].(string)
	if !ok {
		return mcp.NewToolResultError("message must be a string"), nil
	}

	result, err := s.gitOps.CommitChanges(repoPath, message)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to commit: %v", err)), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (s *GitServer) gitAddHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	repoPath, err := s.getRepoPathForOperation(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
	}

	filesStr, ok := request.Params.Arguments["files"].(string)
	if !ok {
		return mcp.NewToolResultError("files must be a string"), nil
	}

	// Split the comma-separated list of files
	files := strings.Split(filesStr, ",")
	// Trim spaces from each file path
	for i, file := range files {
		files[i] = strings.TrimSpace(file)
	}

	result, err := s.gitOps.AddFiles(repoPath, files)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add files: %v", err)), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (s *GitServer) gitResetHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	repoPath, err := s.getRepoPathForOperation(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
	}

	result, err := s.gitOps.ResetStaged(repoPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to reset: %v", err)), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (s *GitServer) gitLogHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	repoPath, err := s.getRepoPathForOperation(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
	}

	maxCount := 10
	if maxCountInterface, ok := request.Params.Arguments["max_count"]; ok {
		if maxCountFloat, ok := maxCountInterface.(float64); ok {
			maxCount = int(maxCountFloat)
		}
	}

	logs, err := s.gitOps.GetLog(repoPath, maxCount)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get log: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Commit history for %s:\n%s", repoPath, strings.Join(logs, "\n"))), nil
}

func (s *GitServer) gitCreateBranchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	repoPath, err := s.getRepoPathForOperation(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
	}

	branchName, ok := request.Params.Arguments["branch_name"].(string)
	if !ok {
		return mcp.NewToolResultError("branch_name must be a string"), nil
	}

	baseBranch := ""
	if baseBranchInterface, ok := request.Params.Arguments["base_branch"]; ok {
		if baseBranchStr, ok := baseBranchInterface.(string); ok {
			baseBranch = baseBranchStr
		}
	}

	result, err := s.gitOps.CreateBranch(repoPath, branchName, baseBranch)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create branch: %v", err)), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (s *GitServer) gitCheckoutHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	repoPath, err := s.getRepoPathForOperation(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
	}

	branchName, ok := request.Params.Arguments["branch_name"].(string)
	if !ok {
		return mcp.NewToolResultError("branch_name must be a string"), nil
	}

	result, err := s.gitOps.CheckoutBranch(repoPath, branchName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to checkout branch: %v", err)), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (s *GitServer) gitShowHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	repoPath, err := s.getRepoPathForOperation(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
	}

	revision, ok := request.Params.Arguments["revision"].(string)
	if !ok {
		return mcp.NewToolResultError("revision must be a string"), nil
	}

	result, err := s.gitOps.ShowCommit(repoPath, revision)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to show commit: %v", err)), nil
	}

	return mcp.NewToolResultText(result), nil
}

func (s *GitServer) gitInitHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	// For init, we don't validate through getRepoPathForOperation since we're creating a new repo
	if requestedPath == "" {
		return mcp.NewToolResultError("repo_path must be specified for initialization"), nil
	}

	// Ensure the path is absolute
	absPath, err := filepath.Abs(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get absolute path: %v", err)), nil
	}

	result, err := s.gitOps.InitRepo(absPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to initialize repository: %v", err)), nil
	}

	// Add the new repository to our list of managed repositories
	s.repoPaths = append(s.repoPaths, absPath)

	return mcp.NewToolResultText(result), nil
}

func (s *GitServer) gitPushHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if write access is enabled
	if !s.writeAccess {
		return mcp.NewToolResultError("Write access is disabled. Use --write-access flag to enable remote operations."), nil
	}

	requestedPath, _ := request.Params.Arguments["repo_path"].(string)
	
	repoPath, err := s.getRepoPathForOperation(requestedPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Repository path error: %v", err)), nil
	}

	remote := ""
	if remoteInterface, ok := request.Params.Arguments["remote"]; ok {
		if remoteStr, ok := remoteInterface.(string); ok {
			remote = remoteStr
		}
	}

	branch := ""
	if branchInterface, ok := request.Params.Arguments["branch"]; ok {
		if branchStr, ok := branchInterface.(string); ok {
			branch = branchStr
		}
	}

	result, err := s.gitOps.PushChanges(repoPath, remote, branch)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to push changes: %v", err)), nil
	}

	return mcp.NewToolResultText(result), nil
}

// gitListRepositoriesHandler lists all available repositories
func (s *GitServer) gitListRepositoriesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if len(s.repoPaths) == 0 {
		return mcp.NewToolResultText("No repositories configured"), nil
	}
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Available repositories (%d):\n\n", len(s.repoPaths)))
	
	for i, repoPath := range s.repoPaths {
		// Get the repository name (last part of the path)
		repoName := filepath.Base(repoPath)
		result.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, repoName, repoPath))
	}
	
	return mcp.NewToolResultText(result.String()), nil
}
