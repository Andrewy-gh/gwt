package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Andrewy-gh/gwt/internal/config"
	"github.com/Andrewy-gh/gwt/internal/copy"
	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/docker"
	"github.com/Andrewy-gh/gwt/internal/git"
	"github.com/Andrewy-gh/gwt/internal/hooks"
)

// createWorktreeCmd performs the worktree creation operation
func createWorktreeCmd(state *CreateFlowState, repoPath string) tea.Cmd {
	return func() tea.Msg {
		// Stage 1: Create worktree
		if state.BranchSpec == nil {
			return CreateCompleteMsg{
				Error: fmt.Errorf("branch spec is not set"),
			}
		}

		// Determine worktree path
		worktreePath := state.TargetDir
		if worktreePath == "" {
			return CreateCompleteMsg{
				Error: fmt.Errorf("target directory is not set"),
			}
		}

		// Send progress for creating worktree stage
		// (Note: In a full implementation, you would use a channel to send progress updates)

		// Create the worktree using git commands
		var err error

		switch state.SourceType {
		case create.BranchSourceNewFromHEAD:
			// Create new branch from HEAD
			_, err = git.AddWorktreeForNewBranch(repoPath, worktreePath, state.BranchSpec.BranchName, "HEAD")

		case create.BranchSourceNewFromRef:
			// Create new branch from specific ref
			_, err = git.AddWorktreeForNewBranch(repoPath, worktreePath, state.BranchSpec.BranchName, state.StartPoint)

		case create.BranchSourceLocalExisting:
			// Checkout existing branch
			_, err = git.AddWorktreeForExistingBranch(repoPath, worktreePath, state.BranchSpec.BranchName)

		case create.BranchSourceRemote:
			// Checkout remote branch with tracking
			if state.SelectedRemote != nil {
				remoteBranch := state.SelectedRemote.Name
				_, err = git.AddWorktreeForRemoteBranch(repoPath, worktreePath, remoteBranch)
			} else {
				err = fmt.Errorf("remote branch not selected")
			}

		default:
			err = fmt.Errorf("unknown branch source type: %d", state.SourceType)
		}

		if err != nil {
			return CreateCompleteMsg{
				WorktreePath: worktreePath,
				Error:        fmt.Errorf("failed to create worktree: %w", err),
			}
		}

		// Stage 2: Copy files
		if state.FileSelection != nil && len(state.FileSelection.Files) > 0 {
			// Get selected files from Selection
			var selectedFiles []copy.SelectableFile
			for _, file := range state.FileSelection.Files {
				if file.Selected {
					selectedFiles = append(selectedFiles, file)
				}
			}

			if len(selectedFiles) > 0 {
				// Copy selected files
				copyOpts := copy.CopyOptions{
					SourceDir:    repoPath,
					TargetDir:    worktreePath,
					Files:        selectedFiles,
					PreserveMode: true,
				}

				_, err = copy.Copy(copyOpts)
				if err != nil {
					// Worktree was created but file copy failed
					return CreateCompleteMsg{
						WorktreePath: worktreePath,
						Error:        fmt.Errorf("worktree created but file copy failed: %w", err),
					}
				}
			}
		}

		// Stage 3: Docker setup
		if state.DockerMode != "none" && state.ComposeDetected {
			switch state.DockerMode {
			case "shared":
				// Symlink data directories
				_, err = docker.SetupSharedMode(docker.SharedModeOptions{
					MainWorktree:    repoPath,
					NewWorktree:     worktreePath,
					ComposeConfig:   state.ComposeConfig,
					DataDirectories: nil, // Will use defaults from config/detection
				})

			case "new":
				// Create isolated containers
				_, err = docker.SetupNewMode(docker.NewModeOptions{
					MainWorktree:    repoPath,
					NewWorktree:     worktreePath,
					BranchName:      state.BranchSpec.BranchName,
					ComposeConfig:   state.ComposeConfig,
					DataDirectories: nil, // Will use defaults from config/detection
					PortOffset:      0,   // Auto-detect
				})
			}

			if err != nil {
				// Worktree was created but Docker setup failed
				return CreateCompleteMsg{
					WorktreePath: worktreePath,
					Error:        fmt.Errorf("worktree created but Docker setup failed: %w", err),
				}
			}
		}

		// Stage 4: Run post-creation hooks
		// Load config and execute hooks (best-effort)
		cfg, _ := config.Load(repoPath)
		if cfg != nil {
			executor := hooks.NewExecutor(repoPath, cfg)
			mainWorktree, _ := git.GetMainWorktree(repoPath)
			mainPath := repoPath
			if mainWorktree != nil {
				mainPath = mainWorktree.Path
			}

			_, _ = executor.Execute(hooks.ExecuteOptions{
				HookType:         hooks.HookTypePostCreate,
				WorktreePath:     worktreePath,
				WorktreeBranch:   state.BranchSpec.BranchName,
				MainWorktreePath: mainPath,
			})
		}

		// Success!
		return CreateCompleteMsg{
			WorktreePath: worktreePath,
			Error:        nil,
		}
	}
}

// fetchRemotesCmd fetches remote branches
func fetchRemotesCmd(repoPath string, remote string, prune bool) tea.Cmd {
	return func() tea.Msg {
		// Fetch from remote
		err := git.Fetch(repoPath, remote, prune)
		if err != nil {
			return FetchRemotesCompleteMsg{
				Branches: nil,
				Error:    fmt.Errorf("failed to fetch remotes: %w", err),
			}
		}

		// List remote branches
		branches, err := git.ListRemoteBranches(repoPath)
		if err != nil {
			return FetchRemotesCompleteMsg{
				Branches: nil,
				Error:    fmt.Errorf("failed to list remote branches: %w", err),
			}
		}

		return FetchRemotesCompleteMsg{
			Branches: branches,
			Error:    nil,
		}
	}
}

// deleteWorktreesCmd performs batch deletion of worktrees
func deleteWorktreesCmd(repoPath string, targets []string, force bool) tea.Cmd {
	return func() tea.Msg {
		var deleted []string
		var failed []DeleteFailure

		for _, path := range targets {
			// Remove the worktree
			err := git.RemoveWorktree(repoPath, git.RemoveWorktreeOptions{
				Path:  path,
				Force: force,
			})

			if err != nil {
				failed = append(failed, DeleteFailure{
					Path:  path,
					Error: err,
				})
			} else {
				deleted = append(deleted, path)

				// Run post-deletion hooks (best-effort)
				cfg, _ := config.Load(repoPath)
				if cfg != nil {
					executor := hooks.NewExecutor(repoPath, cfg)
					mainWorktree, _ := git.GetMainWorktree(repoPath)
					mainPath := repoPath
					if mainWorktree != nil {
						mainPath = mainWorktree.Path
					}

					_, _ = executor.Execute(hooks.ExecuteOptions{
						HookType:         hooks.HookTypePostDelete,
						WorktreePath:     path,
						MainWorktreePath: mainPath,
					})
				}
			}
		}

		// If all failed, return error
		if len(deleted) == 0 && len(failed) > 0 {
			return DeleteCompleteMsg{
				Deleted: deleted,
				Failed:  failed,
				Error:   fmt.Errorf("all deletions failed"),
			}
		}

		return DeleteCompleteMsg{
			Deleted: deleted,
			Failed:  failed,
			Error:   nil,
		}
	}
}

// discoverFilesCmd discovers ignored files in the background
func discoverFilesCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		files, err := copy.DiscoverIgnored(repoPath)
		if err != nil {
			return DiscoverFilesCompleteMsg{
				Files: nil,
				Error: fmt.Errorf("failed to discover files: %w", err),
			}
		}

		return DiscoverFilesCompleteMsg{
			Files: files,
			Error: nil,
		}
	}
}

// validateBranchCmd validates a branch name and creates a spec
func validateBranchCmd(repoPath string, branchName string, sourceType create.BranchSource) tea.Cmd {
	return func() tea.Msg {
		// Validate branch name format
		if err := create.ValidateBranchName(branchName); err != nil {
			return BranchValidationResultMsg{
				Valid: false,
				Error: err.Error(),
				Spec:  nil,
			}
		}

		// Check branch existence based on source type
		switch sourceType {
		case create.BranchSourceNewFromHEAD, create.BranchSourceNewFromRef:
			// Branch should not exist
			exists, err := git.LocalBranchExists(repoPath, branchName)
			if err != nil {
				return BranchValidationResultMsg{
					Valid: false,
					Error: fmt.Sprintf("Failed to check branch: %v", err),
					Spec:  nil,
				}
			}
			if exists {
				return BranchValidationResultMsg{
					Valid: false,
					Error: "Branch already exists",
					Spec:  nil,
				}
			}

		case create.BranchSourceLocalExisting:
			// Branch must exist
			exists, err := git.LocalBranchExists(repoPath, branchName)
			if err != nil {
				return BranchValidationResultMsg{
					Valid: false,
					Error: fmt.Sprintf("Failed to check branch: %v", err),
					Spec:  nil,
				}
			}
			if !exists {
				return BranchValidationResultMsg{
					Valid: false,
					Error: "Branch does not exist",
					Spec:  nil,
				}
			}
		}

		// Create branch spec
		spec := &create.BranchSpec{
			BranchName: branchName,
			Source:     sourceType,
		}

		return BranchValidationResultMsg{
			Valid: true,
			Error: "",
			Spec:  spec,
		}
	}
}

// detectDockerCmd detects docker compose files
func detectDockerCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		// Detect compose files
		files, err := docker.DetectComposeFiles(repoPath)
		if err != nil {
			return DetectDockerCompleteMsg{
				Detected: false,
				Config:   nil,
				Files:    nil,
				Error:    err,
			}
		}

		if len(files) == 0 {
			return DetectDockerCompleteMsg{
				Detected: false,
				Config:   nil,
				Files:    nil,
				Error:    nil,
			}
		}

		// Parse compose files
		var config *docker.ComposeConfig

		for _, file := range files {
			config, err = docker.ParseComposeFile(file.Path)
			if err == nil {
				// Successfully parsed
				break
			}
		}

		if err != nil {
			return DetectDockerCompleteMsg{
				Detected: true,
				Config:   nil,
				Files:    files,
				Error:    fmt.Errorf("failed to parse compose file: %w", err),
			}
		}

		return DetectDockerCompleteMsg{
			Detected: true,
			Config:   config,
			Files:    files,
			Error:    nil,
		}
	}
}
