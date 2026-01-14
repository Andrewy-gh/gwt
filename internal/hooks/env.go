package hooks

import "os"

type HookEnvironment struct {
	WorktreePath     string
	WorktreeBranch   string
	MainWorktreePath string
	RepoPath         string
	HookType         string
}

// BuildEnvironment creates GWT_* environment variables for hook execution.
// Returns a slice suitable for exec.Cmd.Env (merged with current env).
func BuildEnvironment(opts HookEnvironment) []string {
	env := os.Environ()

	gwtVars := map[string]string{
		"GWT_WORKTREE_PATH": opts.WorktreePath,
		"GWT_BRANCH":        opts.WorktreeBranch,
		"GWT_MAIN_WORKTREE": opts.MainWorktreePath,
		"GWT_REPO_PATH":     opts.RepoPath,
		"GWT_HOOK_TYPE":     opts.HookType,
	}

	for k, v := range gwtVars {
		if v != "" {
			env = append(env, k+"="+v)
		}
	}

	return env
}

// BuildEnvironmentMap returns GWT_* variables as a map (for testing/display).
func BuildEnvironmentMap(opts HookEnvironment) map[string]string {
	return map[string]string{
		"GWT_WORKTREE_PATH": opts.WorktreePath,
		"GWT_BRANCH":        opts.WorktreeBranch,
		"GWT_MAIN_WORKTREE": opts.MainWorktreePath,
		"GWT_REPO_PATH":     opts.RepoPath,
		"GWT_HOOK_TYPE":     opts.HookType,
	}
}
