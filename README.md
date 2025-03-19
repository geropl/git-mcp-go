# Git MCP Server (Go)

A Model Context Protocol (MCP) server for Git repository interaction and automation, written in Go. This server provides tools to read, search, and manipulate Git repositories via Large Language Models.

## Features

This MCP server provides the following Git operations as tools:

- **git_status**: Shows the working tree status
- **git_diff_unstaged**: Shows changes in the working directory that are not yet staged
- **git_diff_staged**: Shows changes that are staged for commit
- **git_diff**: Shows differences between branches or commits
- **git_commit**: Records changes to the repository
- **git_add**: Adds file contents to the staging area
- **git_reset**: Unstages all staged changes
- **git_log**: Shows the commit logs
- **git_create_branch**: Creates a new branch from an optional base branch
- **git_checkout**: Switches branches
- **git_show**: Shows the contents of a commit
- **git_init**: Initialize a new Git repository
- **git_push**: Pushes local commits to a remote repository (requires `--write-access` flag)
- **git_list_repositories**: Lists all available Git repositories

## Installation

### Prerequisites

- Go 1.18 or higher
- Git installed on your system

### Download Prebuilt Binaries

You can download prebuilt binaries for your platform from the [GitHub Releases](https://github.com/geropl/git-mcp-go/releases) page.

### Building from Source

```bash
# Clone the repository
git clone https://github.com/geropl/git-mcp-go.git
cd git-mcp-go

# Build the server
go build -o git-mcp-go .
```

### Install with `go install`

```bash
go install github.com/geropl/git-mcp-go@latest
```

## Usage

### Command Line Structure

The Git MCP Server uses a command-line structure with subcommands:

```
git-mcp-go
├── serve [flags] [repository-paths...]
│   ├── --repository, -r <paths>                  # Repository paths (multiple ways to specify)
│   ├── --mode <shell|go-git>
│   ├── --write-access
│   └── --verbose, -v
└── setup [flags] [repository-paths...]
    ├── --repository, -r <paths>                  # Repository paths (multiple ways to specify)
    ├── --mode <shell|go-git>
    ├── --write-access
    ├── --auto-approve <tool-list|allow-read-only|allow-local-only>
    └── --tool <cline,roo-code>
```

### Multi-Repository Support

The Git MCP Server can now monitor and operate on multiple repositories simultaneously. You can specify repositories in several ways:

1. Using the `-r/--repository` flag:
   - With comma-separated paths: `-r=/path/to/repo1,/path/to/repo2`
   - With multiple flag instances: `-r=/path/to/repo1 -r=/path/to/repo2`
2. As positional arguments: `serve /path/to/repo1 /path/to/repo2`
3. A combination of both approaches

When using multiple repositories, the server will default to the first repository for operations where a specific repository is not specified.

### `serve` Command

The `serve` command starts the Git MCP server:

```bash
# Run with verbose logging
./git-mcp-go serve -v /path/to/repo1 /path/to/repo2 /path/to/repo3

# Run with go-git implementation
./git-mcp-go serve --mode go-git -r=/path/to/repo1,/path/to/repo2

# Enable write access for remote operations
./git-mcp-go serve -r=/path/to/repo1,/path/to/repo2 --write-access
```

The `--mode` flag allows you to choose between two different implementations:

- **shell**: Uses the Git CLI commands via shell execution (default)
- **go-git**: Uses the go-git library for Git operations where possible

The `--write-access` flag enables operations that modify remote state (currently only the push operation). By default, this is disabled for safety.

### `setup` Command

The `setup` command sets up the Git MCP server for use with an AI assistant. It copies itself to `~/mcp-servers/git-mcp-go` and modifies the tools config (cline: `cline_mcp_settings.json`) to use that binary.

```bash
# Set up for Cline with a single repository
./git-mcp-go setup -r /path/to/git/repository

# Set up with repositories as arguments
./git-mcp-go setup /path/to/repo1 /path/to/repo2 /path/to/repo3

# Set up with write access enabled
./git-mcp-go setup -r=/path/to/repo1,/path/to/repo2 --write-access

# Set up with auto-approval for read-only tools
./git-mcp-go setup -r=/path/to/repo1,/path/to/repo2 --auto-approve=allow-read-only

# Set up with specific tools auto-approved
./git-mcp-go setup -r=/path/to/repo1,/path/to/repo2 --auto-approve=git_status,git_log

# Set up with write access and auto-approval for read-only tools
./git-mcp-go setup -r=/path/to/repo1,/path/to/repo2 --write-access --auto-approve=allow-read-only
```

The `--auto-approve` flag allows you to specify which tools should be auto-approved (not require explicit user approval):

- **allow-read-only**: Auto-approve all read-only tools (git_status, git_diff_unstaged, git_diff_staged, git_log, git_show, git_diff)
- **allow-local-only**: Auto-approve all local-only tools (incl. git_commit, git_add, git_reset, but not git_push)
- **comma-separated list**: Auto-approve specific tools (e.g., git_status,git_log)

## Repository Management

### The `git_list_repositories` Tool

This tool lists all available Git repositories that the server is monitoring. It shows:

- The total number of repositories
- The path to each repository
- The repository name (derived from the directory name)

Example output:

```
Available repositories (3):

1. repo1 (/path/to/repo1)
2. repo2 (/path/to/repo2)
3. another-project (/path/to/another-project)
```

### Repository Selection

When running commands that require a repository path:

1. If a specific `repo_path` is provided in the command, it will be used.
2. If no `repo_path` is provided and multiple repositories are configured, the first repository will be used as the default.
3. Each command output will indicate which repository was used for the operation.

## Installation

### Automatic Installation and Configuration

The easiest way to install and register the Git MCP server with Cline is to use the setup command:

```bash
# Download linux binary for the latest release
RELEASE="$(curl -s https://api.github.com/repos/geropl/git-mcp-go/releases/latest)"
DOWNLOAD_URL="$(echo $RELEASE | jq -r '.assets[] | select(.name | contains("linux-amd64")) | .browser_download_url')"
curl -L -o ./git-mcp-go $DOWNLOAD_URL
chmod +x ./git-mcp-go

# Setup the mcp server with a single repository
./git-mcp-go setup -r /path/to/git/repository --tool=cline --auto-approve=allow-local-only

# Setup the mcp server with multiple repositories
./git-mcp-go setup -r=/path/to/repo1,/path/to/repo2 --tool=cline --auto-approve=allow-local-only

rm -f ./git-mcp-go
```

The setup command will:
1. Copy the executable to the Cline MCP directory
2. Create a registration script that configures Cline to use the Git MCP server

### Manual Configuration

Alternatively, you can manually add this to your `claude_desktop_config.json`:

```json
"mcpServers": {
  "git": {
    "command": "/path/to/git-mcp-go",
    "args": ["serve", "-r=/path/to/repo1,/path/to/repo2", "--mode", "shell"]
  }
}
```

```json
"mcpServers": {
  "git": {
    "command": "/path/to/git-mcp-go",
    "args": ["serve", "-r", "/path/to/git/repository"]
  }
}
```

## Implementation Details

This server is implemented using:

- [mcp-go](https://github.com/mark3labs/mcp-go): Go SDK for the Model Context Protocol
- [go-git](https://github.com/go-git/go-git): Pure Go implementation of Git (used for the `go-git` mode)

For operations not supported by go-git, the server falls back to using the Git CLI.

## Development

### Testing

The server includes comprehensive tests for all Git operations. The tests are designed to run against both implementation modes:

```bash
# Run all tests
go test ./pkg -v

# Run specific tests
go test ./pkg -v -run TestGitOperations/push
```

The test suite creates temporary repositories for each test case and verifies that the operations work correctly in both modes.

### Continuous Integration

This project uses GitHub Actions for continuous integration and deployment:

- Automated tests run on every pull request to the main branch
- Releases are created when a tag with the format `v*` is pushed
- Each release includes binaries for multiple platforms:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)

To create a new release:
```bash
# Tag the current commit
git tag v1.0.0

# Push the tag to GitHub
git push origin v1.0.0
```

## License

This project is licensed under the MIT License.
