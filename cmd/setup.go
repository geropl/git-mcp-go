package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/geropl/git-mcp-go/pkg"
	"github.com/spf13/cobra"
)

var (
	tool        string
	autoApprove string
)

func init() {
	rootCmd.AddCommand(setupCmd)

	// Add flags to the setup command
	setupCmd.Flags().StringVarP(&repoPath, "repository", "r", "", "Git repository path")
	setupCmd.Flags().StringVar(&mode, "mode", "shell", "Git operation mode: 'shell' or 'go-git'")
	setupCmd.Flags().BoolVar(&writeAccess, "write-access", false, "Enable write access for remote operations (push)")
	setupCmd.Flags().StringVar(&tool, "tool", "cline", "The AI assistant tool to set up for")
	setupCmd.Flags().StringVar(&autoApprove, "auto-approve", "", "Comma-separated list of tools to auto-approve, or 'allow-read-only' to auto-approve all read-only tools, or 'allow-local-only' to auto-approve all local-only tools")
}

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up the Git MCP server for use with an AI assistant",
	Long: `Set up the Git MCP server for use with an AI assistant.

This command sets up the Git MCP server for use with an AI assistant by installing the binary and configuring the AI assistant to use it.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create the MCP servers directory if it doesn't exist
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting user home directory: %v\n", err)
			os.Exit(1)
		}

		mcpServersDir := filepath.Join(homeDir, "mcp-servers")
		if err := os.MkdirAll(mcpServersDir, 0755); err != nil {
			fmt.Printf("Error creating MCP servers directory: %v\n", err)
			os.Exit(1)
		}

		// Check if the git-mcp-go binary is already on the path
		binaryPath, found := checkBinary(mcpServersDir)
		if !found {
			fmt.Printf("git-mcp-go binary not found on path, copying current binary to '%s'...\n", binaryPath)
			err := copySelfToBinaryPath(binaryPath)
			if err != nil {
				fmt.Printf("Error copying git-mcp-go binary: %v\n", err)
				os.Exit(1)
			}
		}

		// Set up the tool-specific configuration
		switch strings.ToLower(tool) {
		case "cline":
			if err := setupCline(binaryPath, repoPath, writeAccess, autoApprove); err != nil {
				fmt.Printf("Error setting up Cline: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Printf("Unsupported tool: %s\n", tool)
			fmt.Println("Currently supported tools: cline")
			os.Exit(1)
		}

		fmt.Printf("git-mcp-go binary successfully set up for %s\n", tool)
	},
}

// checkBinary checks if the git-mcp-go binary is already on the path
func checkBinary(mcpServersDir string) (string, bool) {
	// Try to find the binary on the path
	path, err := exec.LookPath("git-mcp-go")
	if err == nil {
		fmt.Printf("Found git-mcp-go binary at %s\n", path)
		return path, true
	}

	binaryPath := filepath.Join(mcpServersDir, "git-mcp-go")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	if _, err := os.Stat(binaryPath); err == nil {
		fmt.Printf("Found git-mcp-go binary at %s\n", binaryPath)
		return binaryPath, true
	}

	return binaryPath, false
}

// copySelfToBinaryPath copies the current executable to the specified path
func copySelfToBinaryPath(binaryPath string) error {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Check if the destination is the same as the source
	absExecPath, _ := filepath.Abs(execPath)
	absDestPath, _ := filepath.Abs(binaryPath)
	if absExecPath == absDestPath {
		return nil // Already in the right place
	}

	// Copy the file
	sourceFile, err := os.Open(execPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	err = os.MkdirAll(filepath.Dir(binaryPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Make the binary executable
	if runtime.GOOS != "windows" {
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	fmt.Printf("git-mcp-go binary installed successfully at %s\n", binaryPath)
	return nil
}

// setupCline sets up the git-mcp-go server for Cline
func setupCline(binaryPath string, repoPath string, writeAccess bool, autoApprove string) error {
	// Determine the Cline config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	var configDir string
	switch runtime.GOOS {
	case "darwin":
		configDir = filepath.Join(homeDir, "Library", "Application Support", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings")
	case "linux":
		configDir = filepath.Join(homeDir, ".vscode-server", "data", "User", "globalStorage", "saoudrizwan.claude-dev", "settings")
	case "windows":
		configDir = filepath.Join(homeDir, "AppData", "Roaming", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings")
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Create the config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	serverArgs := []string{"serve"}
	if repoPath != "" {
		serverArgs = append(serverArgs, "--repository="+repoPath)
	}
	if writeAccess {
		serverArgs = append(serverArgs, "--write-access=true")
	}

	// Process auto-approve flag
	autoApproveTools := []string{}
	if autoApprove != "" {
		if autoApprove == "allow-read-only" {
			// Get the list of read-only tools
			for k := range pkg.GetReadOnlyToolNames() {
				autoApproveTools = append(autoApproveTools, k)
			}
		} else if autoApprove == "allow-local-only" {
			// Get the list of local-only tools
			for k := range pkg.GetLocalOnlyToolNames() {
				autoApproveTools = append(autoApproveTools, k)
			}
		} else {
			// Split comma-separated list
			for _, tool := range strings.Split(autoApprove, ",") {
				trimmedTool := strings.TrimSpace(tool)
				if trimmedTool != "" {
					autoApproveTools = append(autoApproveTools, trimmedTool)
				}
			}
		}
	}

	// Create the MCP settings file
	settingsPath := filepath.Join(configDir, "cline_mcp_settings.json")
	newSettings := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"git": map[string]interface{}{
				"command":     binaryPath,
				"args":        serverArgs,
				"disabled":    false,
				"autoApprove": autoApproveTools,
			},
		},
	}

	// Check if the settings file already exists
	var settings map[string]interface{}
	if _, err := os.Stat(settingsPath); err == nil {
		// Read the existing settings
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			return fmt.Errorf("failed to read existing settings: %w", err)
		}

		// Parse the existing settings
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("failed to parse existing settings: %w", err)
		}

		// Merge the new settings with the existing settings
		if mcpServers, ok := settings["mcpServers"].(map[string]interface{}); ok {
			mcpServers["git"] = newSettings["mcpServers"].(map[string]interface{})["git"]
		} else {
			settings["mcpServers"] = newSettings["mcpServers"]
		}
	} else {
		// Use the new settings
		settings = newSettings
	}

	// Write the settings to the file
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	fmt.Printf("Cline MCP settings updated at %s\n", settingsPath)
	return nil
}
