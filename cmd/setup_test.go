package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Define expectation types
type fileExpectation struct {
	path     string
	content  string
	mustExist bool
}

// TestSetupCommand tests the setup command with various combinations of parameters
func TestSetupCommand(t *testing.T) {
	// Build the binary
	binaryPath, err := buildBinary()
	if err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.RemoveAll(filepath.Dir(binaryPath))

	type expectations struct {
		files       map[string]fileExpectation // map of tool to file expectation
		errors      []string
		exitCode    int
	}

	// Define test cases
	testCases := []struct {
		name        string
		toolParam   string
		repoPath    string
		writeAccess bool
		autoApprove string
		expect      expectations
	}{
		{
			name:        "Cline Only",
			toolParam:   "cline",
			repoPath:    "/mock/repo",
			writeAccess: true,
			autoApprove: "allow-read-only",
			expect: expectations{
				files: map[string]fileExpectation{
					"cline": {
						path:      "home/.vscode-server/data/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json",
						mustExist: true,
						content: `{
							"mcpServers": {
								"git": {
									"args": ["serve", "--repository=/mock/repo", "--write-access=true"],
									"autoApprove": ["git_status", "git_diff_unstaged", "git_diff_staged", "git_diff", "git_log", "git_show"],
									"disabled": false
								}
							}
						}`,
					},
				},
				exitCode: 0,
			},
		},
		{
			name:        "Roo Code Only",
			toolParam:   "roo-code",
			repoPath:    "/mock/repo",
			writeAccess: true,
			autoApprove: "allow-read-only",
			expect: expectations{
				files: map[string]fileExpectation{
					"roo-code": {
						path:      "home/.vscode-server/data/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json",
						mustExist: true,
						content: `{
							"mcpServers": {
								"git": {
									"args": ["serve", "--repository=/mock/repo", "--write-access=true"],
									"autoApprove": ["git_status", "git_diff_unstaged", "git_diff_staged", "git_diff", "git_log", "git_show"],
									"disabled": false
								}
							}
						}`,
					},
				},
				exitCode: 0,
			},
		},
		{
			name:        "Multiple Tools",
			toolParam:   "cline,roo-code",
			repoPath:    "/mock/repo",
			writeAccess: true,
			autoApprove: "allow-read-only",
			expect: expectations{
				files: map[string]fileExpectation{
					"cline": {
						path:      "home/.vscode-server/data/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json",
						mustExist: true,
						content: `{
							"mcpServers": {
								"git": {
									"args": ["serve", "--repository=/mock/repo", "--write-access=true"],
									"autoApprove": ["git_status", "git_diff_unstaged", "git_diff_staged", "git_diff", "git_log", "git_show"],
									"disabled": false
								}
							}
						}`,
					},
					"roo-code": {
						path:      "home/.vscode-server/data/User/globalStorage/rooveterinaryinc.roo-cline/settings/cline_mcp_settings.json",
						mustExist: true,
						content: `{
							"mcpServers": {
								"git": {
									"args": ["serve", "--repository=/mock/repo", "--write-access=true"],
									"autoApprove": ["git_status", "git_diff_unstaged", "git_diff_staged", "git_diff", "git_log", "git_show"],
									"disabled": false
								}
							}
						}`,
					},
				},
				exitCode: 0,
			},
		},
		{
			name:        "Invalid Tool",
			toolParam:   "invalid-tool,cline",
			repoPath:    "/mock/repo",
			writeAccess: true,
			autoApprove: "allow-read-only",
			expect: expectations{
				files: map[string]fileExpectation{
					"cline": {
						path:      "home/.vscode-server/data/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json",
						mustExist: true,
						content: `{
							"mcpServers": {
								"git": {
									"args": ["serve", "--repository=/mock/repo", "--write-access=true"],
									"autoApprove": ["git_status", "git_diff_unstaged", "git_diff_staged", "git_diff", "git_log", "git_show"],
									"disabled": false
								}
							}
						}`,
					},
				},
				errors:   []string{"Unsupported tool: invalid-tool"},
				exitCode: 1,
			},
		},
	}

	// Run each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory
			rootDir, err := os.MkdirTemp("", "git-mcp-go-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(rootDir)

			// Set up the directory structure
			homeDir := filepath.Join(rootDir, "home")

			// Copy the binary to the temp directory
			tempBinaryPath := filepath.Join(rootDir, "git-mcp-go")
			if err := copyFile(binaryPath, tempBinaryPath); err != nil {
				t.Fatalf("Failed to copy binary: %v", err)
			}
			if err := os.Chmod(tempBinaryPath, 0755); err != nil {
				t.Fatalf("Failed to make binary executable: %v", err)
			}

			// Set the HOME environment variable
			oldHome := os.Getenv("HOME")
			os.Setenv("HOME", homeDir)
			defer os.Setenv("HOME", oldHome)

			// Build the command - using -r instead of --repository for the new StringSlice flag format
			args := []string{"setup", "--tool=" + tc.toolParam}
			if tc.repoPath != "" {
				args = append(args, "-r="+tc.repoPath)
			}
			if tc.writeAccess {
				args = append(args, "--write-access=true")
			}
			if tc.autoApprove != "" {
				args = append(args, "--auto-approve="+tc.autoApprove)
			}

			t.Logf("Running command: %s %s", tempBinaryPath, strings.Join(args, " "))

			// Execute the command
			cmd := exec.Command(tempBinaryPath, args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err = cmd.Run()

			// Log output for debugging
			t.Logf("Command stdout: %s", stdout.String())
			t.Logf("Command stderr: %s", stderr.String())

			// Check exit code
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					t.Fatalf("Failed to run command: %v", err)
				}
			}

			// Verify exit code
			if exitCode != tc.expect.exitCode {
				t.Errorf("Expected exit code %d, got %d", tc.expect.exitCode, exitCode)
			}

			// Verify expected files
			verifyFileExpectations(t, rootDir, tc.expect.files)

			// Verify expected errors in output
			output := stdout.String() + stderr.String()
			for _, expectedError := range tc.expect.errors {
				if !strings.Contains(output, expectedError) {
					t.Errorf("Expected output to contain '%s', got: %s", expectedError, output)
				}
			}
		})
	}
}

// Helper function to verify file expectations
func verifyFileExpectations(t *testing.T, rootDir string, fileExpects map[string]fileExpectation) {
	for tool, expect := range fileExpects {
		// Extract the important part after "globalStorage/"
		parts := strings.Split(expect.path, "globalStorage/")
		if len(parts) != 2 {
			t.Fatalf("Invalid test path format for %s: %s", tool, expect.path)
			continue
		}
		
		// The important part is what comes after "globalStorage/"
		importantPathSuffix := parts[1]
		
		// Find the file in any OS-specific path structure
		found := false
		var actualContent []byte
		var foundPath string
		
		err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			// Skip directories
			if info.IsDir() {
				return nil
			}
			
			// Check if this is our target file
			if strings.Contains(path, "globalStorage/"+importantPathSuffix) {
				found = true
				foundPath = path
				
				// Read the file content if needed
				if expect.content != "" {
					content, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					actualContent = content
				}
				
				// Stop the walk
				return filepath.SkipDir
			}
			
			return nil
		})
		
		if err != nil {
			t.Fatalf("Error walking directory tree: %v", err)
		}
		
		// Check if we found the file
		if !found {
			if expect.mustExist {
				t.Errorf("Expected file with suffix 'globalStorage/%s' was not created for %s", 
					importantPathSuffix, tool)
			}
			continue
		}
		
		t.Logf("Found matching file for %s: %s", tool, foundPath)
		
		// File exists, verify content if expected
		if expect.content != "" {
			// Parse both expected and actual content as JSON for comparison
			var expectedJSON, actualJSON map[string]interface{}
			
			if err := json.Unmarshal([]byte(expect.content), &expectedJSON); err != nil {
				t.Fatalf("Failed to parse expected JSON for %s: %v", tool, err)
			}
			
			if err := json.Unmarshal(actualContent, &actualJSON); err != nil {
				t.Fatalf("Failed to parse actual JSON in file %s: %v", foundPath, err)
			}
			
			// Process the JSON objects to make them comparable
			normalizeJSON(expectedJSON)
			normalizeJSON(actualJSON)
			
			// Compare the JSON objects
			if diff := cmp.Diff(expectedJSON, actualJSON); diff != "" {
				t.Errorf("File content mismatch for %s (-want +got):\n%s", tool, diff)
			}
		}
	}
}

// normalizeJSON processes a JSON object to make it comparable
// by removing fields that may vary and sorting arrays
func normalizeJSON(jsonObj map[string]interface{}) {
	if mcpServers, ok := jsonObj["mcpServers"].(map[string]interface{}); ok {
		if git, ok := mcpServers["git"].(map[string]interface{}); ok {
			// Remove the command field since it contains the full path
			delete(git, "command")
			
			// Sort the autoApprove array
			if autoApprove, ok := git["autoApprove"].([]interface{}); ok {
				// Convert to strings and sort
				strSlice := make([]string, len(autoApprove))
				for i, v := range autoApprove {
					strSlice[i] = v.(string)
				}
				
				// Sort the strings
				sort.Strings(strSlice)
				
				// Convert back to []interface{}
				sortedSlice := make([]interface{}, len(strSlice))
				for i, v := range strSlice {
					sortedSlice[i] = v
				}
				
				// Replace the original array with the sorted one
				git["autoApprove"] = sortedSlice
			}
		}
	}
}

// Helper function to build the binary
func buildBinary() (string, error) {
	// Create a temporary directory for the binary
	tempDir, err := os.MkdirTemp("", "git-mcp-go-build-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Get the project root directory (parent of cmd directory)
	currentDir, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	
	// Ensure we're building from the project root
	projectRoot := filepath.Dir(currentDir)
	if filepath.Base(currentDir) != "cmd" {
		// If we're already in the project root, use the current directory
		projectRoot = currentDir
	}
	
	fmt.Printf("Building binary from project root: %s\n", projectRoot)
	
	// Build the binary
	binaryPath := filepath.Join(tempDir, "git-mcp-go")
	cmd := exec.Command("go", "build", "-o", binaryPath)
	cmd.Dir = projectRoot // Set the working directory to the project root
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to build binary: %w\nstdout: %s\nstderr: %s",
			err, stdout.String(), stderr.String())
	}

	// Verify the binary exists and is executable
	info, err := os.Stat(binaryPath)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to stat binary: %w", err)
	}
	
	if info.Size() == 0 {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("binary file is empty")
	}

	// Make sure the binary is executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	fmt.Printf("Successfully built binary at %s (size: %d bytes)\n", binaryPath, info.Size())
	return binaryPath, nil
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
