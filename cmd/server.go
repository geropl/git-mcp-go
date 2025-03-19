package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/geropl/git-mcp-go/pkg"
	"github.com/geropl/git-mcp-go/pkg/gitops"
	"github.com/geropl/git-mcp-go/pkg/gitops/gogit"
	"github.com/geropl/git-mcp-go/pkg/gitops/shell"
	"github.com/spf13/cobra"
)

var (
	repoPaths   []string
	verbose     bool
	mode        string
	writeAccess bool
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve [repository-paths...]",
	Short: "Start the Git MCP server",
	Long: `Start the Git MCP server.

This command starts the Git MCP server, which provides tools for interacting with Git repositories through the MCP protocol.

You can specify multiple repositories using the -r/--repository flag (can be repeated or comma-separated) or by passing paths as arguments.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create the appropriate GitOperations implementation
		var gitOps gitops.GitOperations
		switch strings.ToLower(mode) {
		case "go-git":
			if verbose {
				fmt.Println("Using go-git implementation")
			}
			gitOps = gogit.NewGoGitOperations()
		case "shell":
			if verbose {
				fmt.Println("Using shell implementation")
			}
			gitOps = shell.NewShellGitOperations()
		default:
			if verbose {
				fmt.Println("Using shell implementation")
			}
			gitOps = shell.NewShellGitOperations()
		}

		// Collect all repository paths
		allRepoPaths := make([]string, 0)

		// Add repositories from the -r/--repository flag
		allRepoPaths = append(allRepoPaths, repoPaths...)

		// Add repositories from arguments
		allRepoPaths = append(allRepoPaths, args...)

		if len(allRepoPaths) == 0 {
			fmt.Fprintf(os.Stderr, "Error: No repositories specified. Use --repository or provide paths as arguments.\n")
			os.Exit(1)
		}

		if verbose {
			fmt.Printf("Monitoring %d repositories\n", len(allRepoPaths))
			for i, path := range allRepoPaths {
				fmt.Printf("  %d. %s\n", i+1, path)
			}
		}

		// Create and configure the Git MCP server
		gitServer := pkg.NewGitServer(allRepoPaths, gitOps, writeAccess)

		// Register all Git tools
		gitServer.RegisterTools()

		// Start the server
		if verbose {
			fmt.Println("Starting Git MCP Server...")
		}
		if err := gitServer.Serve(); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Add flags to the server command
	serveCmd.Flags().StringSliceVarP(&repoPaths, "repository", "r", []string{},
		"Git repository paths (can be specified multiple times, comma-separated, or as positional arguments)")
	serveCmd.Flags().StringVar(&mode, "mode", "shell", "Git operation mode: 'shell' or 'go-git'")
	serveCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	serveCmd.Flags().BoolVar(&writeAccess, "write-access", false, "Enable write access for remote operations (push)")
}
