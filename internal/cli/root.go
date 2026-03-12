package cli

import (
	"fmt"
	"os"
	"strings"

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
	args := normalizeCLIArgs(os.Args[1:])
	if len(args) > 0 {
		rootCmd.SetArgs(args)
	}

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

func normalizeCLIArgs(args []string) []string {
	commandIndex, commandName := firstCommandToken(args)
	if commandIndex == -1 || commandName == "" {
		return args
	}

	if isKnownRootSubcommand(commandName) {
		return args
	}

	normalized := make([]string, 0, len(args)+1)
	normalized = append(normalized, args[:commandIndex]...)
	normalized = append(normalized, "create")
	normalized = append(normalized, args[commandIndex:]...)

	return normalized
}

func firstCommandToken(args []string) (int, string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return -1, ""
		}

		if arg == "-c" || arg == "--config" {
			i++
			continue
		}

		if strings.HasPrefix(arg, "--config=") {
			continue
		}

		if strings.HasPrefix(arg, "-") {
			continue
		}

		return i, arg
	}

	return -1, ""
}

func isKnownRootSubcommand(name string) bool {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == name {
			return true
		}

		for _, alias := range cmd.Aliases {
			if alias == name {
				return true
			}
		}
	}

	return false
}
