# gwt

`gwt` is a Git worktree manager for teams that want fast branch switching without repeatedly hand-assembling local setup.

It wraps the usual Git worktree flow with repository checks, optional TUI flows, config-driven file copying, Docker setup, dependency installation, migrations, hooks, and branch cleanup.

## Highlights

- Create worktrees from new, existing, or remote branches.
- Use a TUI by default or run fully from flags for scripts.
- Copy selected ignored files into new worktrees.
- Scaffold Docker Compose overrides and helper scripts.
- Run dependency installation, migrations, and post-create hooks.
- List, inspect, delete, unlock, and clean up worktrees and branches.

## Installation

### From source

```bash
git clone https://github.com/Andrewy-gh/gwt
cd gwt
go build -o gwt ./cmd/gwt
```

### Prerequisites

- Git 2.20+
- Go 1.26+
- Docker and Docker Compose if you use the Docker or migration workflows

## Quick start

Check the environment:

```bash
gwt doctor
```

Create a default config:

```bash
gwt config init
```

Create a new worktree:

```bash
gwt create -b feature/auth
```

List worktrees:

```bash
gwt list
gwt status
```

Clean up merged branches:

```bash
gwt cleanup --merged --dry-run
```

## Common commands

| Command | What it does |
| --- | --- |
| `gwt doctor` | Validate Git, repo, and optional tooling prerequisites |
| `gwt create` | Create a new worktree from a branch source |
| `gwt list` | List worktrees |
| `gwt status` | Show detailed worktree status |
| `gwt delete` | Delete one or more worktrees safely |
| `gwt cleanup` | Remove merged or stale branches |
| `gwt config` | Show, edit, or initialize `.worktree.yaml` |
| `gwt unlock` | Remove a stale operation lock |

For the live CLI surface, run:

```bash
gwt --help
gwt create --help
gwt cleanup --help
```

## Example configuration

```yaml
copy_defaults:
  - ".env"
  - "**/.env.local"

docker:
  default_mode: "shared"
  port_offset: 1

dependencies:
  auto_install: true
  paths:
    - "."

migrations:
  auto_detect: true

hooks:
  post_create:
    - "echo setup"
```

## Documentation

Active docs live in [`docs/README.md`](docs/README.md).

- [`docs/GWT_SPEC.md`](docs/GWT_SPEC.md): product and workflow spec
- [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md): development notes and contributor guidance
- [`docs/WINDOWS_GUIDE.md`](docs/WINDOWS_GUIDE.md): Windows-specific setup and troubleshooting
- [`docs/CHANGELOG.md`](docs/CHANGELOG.md): release history

Historical phase plans, summaries, and milestone notes are maintained outside the repo.

## Development

Run the core checks before shipping changes:

```bash
go test ./...
go vet ./...
go fmt ./...
```

If you are changing docs, keep the root README focused on current usage and use `docs/` for deeper reference material or historical archive notes.
