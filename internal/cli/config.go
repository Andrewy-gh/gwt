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
	configInitForce  bool
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

// configEditCmd opens an interactive config editor
var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration interactively",
	Long: `Open an interactive TUI editor for configuration.

The editor allows you to:
  • View all configuration options
  • Edit string, boolean, and integer values
  • Add, remove, and modify array values
  • Save changes to the config file

Changes are validated before saving.`,
	RunE: runConfigEdit,
}

// configSetCmd sets a specific config value
var configSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "Set a config value",
	Long: `Set a specific configuration value.

Examples:
  gwt config set docker.port_offset 100
  gwt config set docker.default_mode shared
  gwt config set dependencies.auto_install true`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

// configGetCmd gets a specific config value
var configGetCmd = &cobra.Command{
	Use:   "get KEY",
	Short: "Get a config value",
	Long: `Get a specific configuration value.

Examples:
  gwt config get docker.port_offset
  gwt config get docker.default_mode
  gwt config get copy_defaults`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)

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

func runConfigEdit(cmd *cobra.Command, args []string) error {
	output.Info("Interactive config editing")
	output.Println("")
	output.Println("To edit configuration interactively, use:")
	output.Println("  gwt tui    # Navigate to 'Configuration' in the menu")
	output.Println("")
	output.Println("To set specific values from the command line:")
	output.Println("  gwt config set KEY VALUE")
	output.Println("")
	output.Println("Examples:")
	output.Println("  gwt config set docker.port_offset 100")
	output.Println("  gwt config set docker.default_mode shared")
	output.Println("  gwt config set dependencies.auto_install true")
	output.Println("")
	output.Println("Use 'gwt config show' to view current configuration.")
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Load current config
	cfg, _, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set the value based on the key
	switch key {
	case "docker.default_mode":
		if value != "shared" && value != "new" {
			return fmt.Errorf("docker.default_mode must be 'shared' or 'new'")
		}
		cfg.Docker.DefaultMode = value
	case "docker.port_offset":
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err != nil {
			return fmt.Errorf("docker.port_offset must be an integer: %w", err)
		}
		if intVal < 0 || intVal >= 65535 {
			return fmt.Errorf("docker.port_offset must be between 0 and 65534")
		}
		cfg.Docker.PortOffset = intVal
	case "dependencies.auto_install":
		cfg.Dependencies.AutoInstall = (value == "true" || value == "yes" || value == "1")
	case "migrations.auto_detect":
		cfg.Migrations.AutoDetect = (value == "true" || value == "yes" || value == "1")
	case "migrations.command":
		cfg.Migrations.Command = value
	default:
		return fmt.Errorf("unknown config key: %s\n\nSupported keys:\n  docker.default_mode, docker.port_offset\n  dependencies.auto_install\n  migrations.auto_detect, migrations.command", key)
	}

	// Validate the config
	if validationErrors := cfg.Validate(); len(validationErrors) > 0 {
		return fmt.Errorf("validation failed: %s", validationErrors[0].Error())
	}

	// Save the config
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := config.Save(cwd, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	output.Success(fmt.Sprintf("Set %s = %s", key, value))
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	// Load current config
	cfg, _, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get the value based on the key
	var value interface{}
	switch key {
	case "docker.default_mode":
		value = cfg.Docker.DefaultMode
	case "docker.port_offset":
		value = cfg.Docker.PortOffset
	case "docker.compose_files":
		value = cfg.Docker.ComposeFiles
	case "docker.data_directories":
		value = cfg.Docker.DataDirectories
	case "dependencies.auto_install":
		value = cfg.Dependencies.AutoInstall
	case "dependencies.paths":
		value = cfg.Dependencies.Paths
	case "migrations.auto_detect":
		value = cfg.Migrations.AutoDetect
	case "migrations.command":
		value = cfg.Migrations.Command
	case "copy_defaults":
		value = cfg.CopyDefaults
	case "copy_exclude":
		value = cfg.CopyExclude
	case "hooks.post_create":
		value = cfg.Hooks.PostCreate
	case "hooks.post_delete":
		value = cfg.Hooks.PostDelete
	default:
		return fmt.Errorf("unknown config key: %s\n\nSupported keys:\n  docker.default_mode, docker.port_offset, docker.compose_files, docker.data_directories\n  dependencies.auto_install, dependencies.paths\n  migrations.auto_detect, migrations.command\n  copy_defaults, copy_exclude\n  hooks.post_create, hooks.post_delete", key)
	}

	// Print the value
	switch v := value.(type) {
	case []string:
		if len(v) == 0 {
			output.Println("(empty)")
		} else {
			for _, item := range v {
				output.Println(item)
			}
		}
	case bool:
		if v {
			output.Println("true")
		} else {
			output.Println("false")
		}
	default:
		output.Println(fmt.Sprintf("%v", value))
	}

	return nil
}
