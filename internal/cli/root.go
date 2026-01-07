package cli

import (
	"fmt"
	"os"

	"github.com/Andrewy-gh/gwt/internal/output"
	"github.com/Andrewy-gh/gwt/internal/version"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verboseFlag bool
	quietFlag   bool
	configFlag  string
	noTUIFlag   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gwt",
	Short: "Git Worktree Manager - Simplify your Git worktree workflow",
	Long: `gwt (Git Worktree Manager) is a CLI tool that streamlines working with
Git worktrees, making it easy to manage multiple parallel branches.

Features:
  • Create and manage worktrees with ease
  • Interactive TUI for worktree selection
  • Docker and symlink support for databases
  • Branch management and cleanup utilities

Run 'gwt doctor' to check system prerequisites.`,
	Version: version.String(),
	// Silence usage on error to avoid cluttering output
	SilenceUsage:  true,
	SilenceErrors: true,
	// PersistentPreRun sets up the output package with the global flags
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		output.SetVerbose(verboseFlag)
		output.SetQuiet(quietFlag)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "suppress non-essential output")
	rootCmd.PersistentFlags().StringVarP(&configFlag, "config", "c", "", "path to config file")
	rootCmd.PersistentFlags().BoolVar(&noTUIFlag, "no-tui", false, "disable TUI, use simple prompts")

	// Mark flags as mutually exclusive
	rootCmd.MarkFlagsMutuallyExclusive("verbose", "quiet")
}

// GetVerbose returns the verbose flag value
func GetVerbose() bool {
	return verboseFlag
}

// GetQuiet returns the quiet flag value
func GetQuiet() bool {
	return quietFlag
}

// GetConfig returns the config flag value
func GetConfig() string {
	return configFlag
}

// GetNoTUI returns the no-tui flag value
func GetNoTUI() bool {
	return noTUIFlag
}
