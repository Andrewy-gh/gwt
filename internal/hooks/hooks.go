package hooks

import (
	"fmt"

	"github.com/Andrewy-gh/gwt/internal/config"
	"github.com/Andrewy-gh/gwt/internal/output"
)

const (
	HookTypePostCreate = "post_create"
	HookTypePostDelete = "post_delete"
)

type Executor struct {
	repoPath string
	config   *config.Config
}

type ExecuteOptions struct {
	HookType         string
	WorktreePath     string
	WorktreeBranch   string
	MainWorktreePath string
}

type ExecuteResult struct {
	Executed   int
	Successful int
	Failed     int
	Errors     []*HookError
}

func NewExecutor(repoPath string, cfg *config.Config) *Executor {
	return &Executor{
		repoPath: repoPath,
		config:   cfg,
	}
}

// Execute runs all hooks of the specified type.
// Hooks are executed sequentially; failures are collected but don't stop execution.
func (e *Executor) Execute(opts ExecuteOptions) (*ExecuteResult, error) {
	commands := e.getCommands(opts.HookType)
	if len(commands) == 0 {
		return &ExecuteResult{}, nil
	}

	result := &ExecuteResult{
		Errors: make([]*HookError, 0),
	}

	// Build environment variables
	env := BuildEnvironment(HookEnvironment{
		WorktreePath:     opts.WorktreePath,
		WorktreeBranch:   opts.WorktreeBranch,
		MainWorktreePath: opts.MainWorktreePath,
		RepoPath:         e.repoPath,
		HookType:         opts.HookType,
	})

	// Determine working directory
	workDir := opts.WorktreePath
	if workDir == "" {
		workDir = e.repoPath // For post_delete, worktree may not exist
	}

	// Execute each hook
	for _, cmd := range commands {
		result.Executed++
		output.Verbose(fmt.Sprintf("Running hook: %s", cmd))

		execResult, err := ExecuteCommand(ExecOptions{
			Command: cmd,
			Dir:     workDir,
			Env:     env,
		})

		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, &HookError{
				Command: cmd,
				Err:     err,
			})
			continue
		}

		if execResult.ExitCode != 0 {
			result.Failed++
			result.Errors = append(result.Errors, &HookError{
				Command:  cmd,
				ExitCode: execResult.ExitCode,
				Stderr:   execResult.Stderr,
			})
			continue
		}

		result.Successful++

		// Show stdout in verbose mode if present
		if execResult.Stdout != "" {
			output.Verbose(execResult.Stdout)
		}
	}

	return result, nil
}

func (e *Executor) getCommands(hookType string) []string {
	if e.config == nil {
		return nil
	}

	switch hookType {
	case HookTypePostCreate:
		return e.config.Hooks.PostCreate
	case HookTypePostDelete:
		return e.config.Hooks.PostDelete
	default:
		return nil
	}
}

// HasHooks returns true if any hooks are configured for the given type.
func (e *Executor) HasHooks(hookType string) bool {
	return len(e.getCommands(hookType)) > 0
}
