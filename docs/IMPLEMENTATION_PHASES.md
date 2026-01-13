# GWT Implementation Phases

A phased breakdown of the Git Worktree Manager implementation.

---

## Phase 1: Project Foundation

- [X] Initialize Go module and project structure
- [X] Set up Cobra CLI framework with root command
- [X] Implement `gwt doctor` command (prerequisite checks)
- [X] Add global flags (`--verbose`, `--quiet`, `--help`, `--version`)
- [X] Create basic error handling and output utilities

---

## Phase 2: Git Operations Core

- [X] Implement git command execution wrapper
- [X] Build worktree operations (list, add, remove via git CLI)
- [X] Build branch operations (create, delete, list local/remote)
- [X] Implement repository state validation (is git repo, is bare, etc.)
- [X] Add worktree status detection (clean/dirty, last commit, age)

---

## Phase 3: Configuration System

- [X] Define config struct for `.worktree.yaml`
- [X] Implement config loading with Viper
- [X] Add config inheritance (read from main worktree)
- [X] Implement `gwt config` and `gwt config init` commands
- [X] Set up default values for all config options

---

## Phase 4: Create Worktree (CLI)

- [X] Implement `gwt create` command structure
- [X] Add branch name validation and directory name conversion
- [X] Support new branch creation from HEAD or specific ref
- [X] Support existing local branch checkout
- [X] Support remote branch checkout with local tracking branch
- [X] Implement directory collision detection and handling
- [X] Add rollback on failure (cleanup partial worktree)
- [X] Implement concurrent operation locking

---

## Phase 5: List & Delete Worktrees (CLI)

- [X] Implement `gwt list` command with table output
- [X] Add `--json` and `--simple` output formats
- [X] Implement `gwt status` command
- [X] Implement `gwt delete` command
- [X] Add pre-deletion checks (uncommitted changes, merged status, remote existence)
- [X] Support batch deletion with confirmation
- [X] Implement `--force` and `--delete-branch` flags
- [X] Prevent main worktree deletion

---

## Phase 6: File Copying

- [X] Implement gitignored file discovery via `git status --ignored`
- [X] Build file/directory copy with progress tracking
- [X] Apply copy_defaults and copy_exclude patterns from config
- [X] Auto-exclude dependency directories by default
- [X] Show file sizes in selection

---

## Phase 7: Docker Compose Scaffolding

- [X] Implement compose file auto-detection
- [X] Parse compose files for services and volumes
- [X] Implement Shared mode (symlink data directories)
- [X] Implement New mode (copy data, rename volumes, generate override)
- [X] Add Windows symlink fallback (junction, then copy)
- [X] Generate `dc` helper script
- [X] Handle port conflict warnings

---

## Phase 8: Dependency Installation

- [X] Implement package manager detection (npm, yarn, pnpm, bun, go, cargo, pip, poetry)
- [X] Support monorepo detection via config paths
- [X] Run installations with output streaming
- [X] Add `--skip-install` flag

---

## Phase 9: Database Migrations

- [ ] Implement migration tool detection (Makefile, Prisma, Drizzle, Alembic, raw SQL)
- [ ] Check database container status before running
- [ ] Execute migrations with output streaming
- [ ] Add `--skip-migrations` flag

---

## Phase 10: Post-Setup Hooks

- [ ] Implement hook execution from config
- [ ] Set up GWT_* environment variables
- [ ] Run hooks in new worktree directory
- [ ] Support post_create and post_delete hooks

---

## Phase 11: TUI Framework

- [ ] Set up Bubble Tea application structure
- [ ] Create Lip Gloss styles and theme
- [ ] Build reusable components (checkbox list, text input, table)
- [ ] Implement main menu view
- [ ] Add keyboard navigation and help footer

---

## Phase 12: TUI Views

- [ ] Build create worktree flow (branch input, source selection)
- [ ] Build remote branch selection with filter/refresh
- [ ] Build file selection view with checkboxes
- [ ] Build Docker mode selection view
- [ ] Build worktree list view with batch selection
- [ ] Implement delete confirmation with pre-flight checks display

---

## Phase 13: Integration & Polish

- [ ] Wire TUI views to core operations
- [ ] Add `--no-tui` flag for simple prompts fallback
- [ ] Implement progress indicators for long operations
- [ ] Add comprehensive error messages
- [ ] Test Windows symlink/junction fallback
- [ ] Write README with usage examples

---

## Dependency Order

```
Phase 1 (Foundation)
    ↓
Phase 2 (Git Core)
    ↓
Phase 3 (Config)
    ↓
┌───┴───┬───────┬───────┬───────┐
↓       ↓       ↓       ↓       ↓
Phase 4 Phase 5 Phase 6 Phase 7 Phase 8-10
(Create) (List)  (Files) (Docker) (Deps/Migrations/Hooks)
    ↓       ↓       ↓       ↓       ↓
    └───────┴───────┴───────┴───────┘
                    ↓
            Phase 11 (TUI Framework)
                    ↓
            Phase 12 (TUI Views)
                    ↓
            Phase 13 (Integration)
```

---

## Notes

- Phases 4-10 can be developed in parallel after Phase 3
- TUI development (11-12) can proceed alongside CLI features
- Each phase should include unit tests for core logic
- Windows compatibility should be verified throughout, not just at the end
