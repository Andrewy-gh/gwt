package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Andrewy-gh/gwt/internal/config"
	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	configInitForce bool
	configInitOutput string
)

// configCmd shows the current configuration
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or manage configuration",
	Long: `Display the current configuration or manage .worktree.yaml

The config command allows you to view the effective configuration,
initialize a default config file, or check where the config is loaded from.

Subcommands:
  show    Display current configuration (default)
  init    Create a default .worktree.yaml file
  path    Show config file path`,
	RunE: runConfigShow, // Default to showing config
}

// configShowCmd displays the current config
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	Long: `Display the current effective configuration in YAML format.

The configuration is loaded from:
  1. Explicit path via --config flag
  2. .worktree.yaml in current directory
  3. .worktree.yaml in repository root
  4. .worktree.yaml in main worktree (for linked worktrees)
  5. Default values if no config file exists`,
	RunE: runConfigShow,
}

// configInitCmd creates a default config file
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default .worktree.yaml file",
	Long: `Create a default .worktree.yaml file in the repository root.

The generated file includes:
  • All available configuration options
  • Helpful comments explaining each setting
  • Sensible defaults ready for customization

Use --force to overwrite an existing config file.
Use --output to specify a different location.`,
	RunE: runConfigInit,
}

// configPathCmd shows where config is loaded from
var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config file path",
	Long: `Display the path to the configuration file that would be loaded.

Indicates whether the configuration is:
  • Loaded from a specific file
  • Inherited from the main worktree
  • Using default values (no config file)`,
	RunE: runConfigPath,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configPathCmd)

	// Flags for config init
	configInitCmd.Flags().BoolVarP(&configInitForce, "force", "f", false, "overwrite existing config file")
	configInitCmd.Flags().StringVarP(&configInitOutput, "output", "o", "", "output path (default: .worktree.yaml in repo root)")
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, sourcePath, err := loadConfig()
	if err != nil {
		return err
	}

	// Display source information
	if sourcePath != "" {
		// Check if inherited
		inherited, _ := config.IsInheritedConfig(".")
		if inherited {
			output.Info(fmt.Sprintf("Configuration source: %s (inherited from main worktree)\n", sourcePath))
		} else {
			output.Info(fmt.Sprintf("Configuration source: %s\n", sourcePath))
		}
	} else {
		output.Info("No configuration file found. Using defaults.\n")
	}

	// Marshal config to YAML
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Print YAML
	fmt.Println(string(yamlData))

	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	// Determine output path
	outputPath := configInitOutput
	if outputPath == "" {
		// Use repository root if available
		if git.IsRepository() {
			repoRoot, err := git.GetRepoRoot(".")
			if err != nil {
				return fmt.Errorf("failed to get repository root: %w", err)
			}
			outputPath = filepath.Join(repoRoot, ".worktree.yaml")
		} else {
			// Not in a repository, use current directory
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			outputPath = filepath.Join(cwd, ".worktree.yaml")
		}
	}

	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		if !configInitForce {
			output.Error(fmt.Sprintf("Configuration file already exists: %s\n", outputPath))
			output.Println("")
			output.Println("Use --force to overwrite.")
			return nil
		}
	}

	// Write the config file
	err := os.WriteFile(outputPath, []byte(config.DefaultConfigTemplate), 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Show success message
	output.Success(fmt.Sprintf("Created .worktree.yaml in %s\n", filepath.Dir(outputPath)))
	output.Println("")
	output.Println("Edit this file to customize gwt behavior for this repository.")

	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	// Check if explicit config path was provided
	if configFlag != "" {
		// Validate that the file exists
		if _, err := os.Stat(configFlag); err != nil {
			return fmt.Errorf("config file not found: %s", configFlag)
		}
		fmt.Println(configFlag)
		return nil
	}

	// Get effective config path
	configPath, err := config.GetEffectiveConfigPath(".")
	if err != nil {
		return err
	}

	if configPath == "" {
		output.Info("No configuration file found. Using defaults.")
		return nil
	}

	// Check if inherited
	inherited, _ := config.IsInheritedConfig(".")
	if inherited {
		fmt.Printf("%s (inherited from main worktree)\n", configPath)
	} else {
		fmt.Println(configPath)
	}

	return nil
}

// loadConfig loads the configuration based on the --config flag or default search
func loadConfig() (*config.Config, string, error) {
	if configFlag != "" {
		// Explicit config path provided
		cfg, err := config.Load(configFlag)
		if err != nil {
			return nil, "", err
		}
		return cfg, configFlag, nil
	}

	// Use inheritance-aware loading
	return config.LoadWithInheritance(".")
}
