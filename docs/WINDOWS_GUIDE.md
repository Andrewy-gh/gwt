# GWT Windows Guide

Complete guide for using Git Worktree Manager (GWT) on Windows.

## Prerequisites

### Required
- **Git for Windows** (2.25 or later)
  - Download from [git-scm.com](https://git-scm.com/download/win)
  - Includes Git Bash and core Git tools
- **Windows 10/11** or **Windows Server 2016+**
  - Earlier versions may work but are not officially supported

### Recommended
- **Windows Terminal** (for better TUI experience)
  - Install from Microsoft Store or [GitHub](https://github.com/microsoft/terminal)
  - Provides better Unicode and color support for the TUI
- **Developer Mode enabled** (for symlink support without admin rights)
  - See [Symlink Support](#symlink-support) below

## Installation

Download the latest Windows release from the [releases page](https://github.com/Andrewy-gh/gwt/releases) or build from source:

```bash
git clone https://github.com/Andrewy-gh/gwt.git
cd gwt
go build -o gwt.exe ./cmd/gwt
```

Add `gwt.exe` to your PATH or move it to a directory already in PATH.

## Symlink Support

GWT uses symlinks for Docker Compose shared mode. On Windows, symlinks require special privileges.

### Option 1: Enable Developer Mode (Recommended)

This grants symlink creation privileges to all users without requiring administrator rights.

**Steps:**
1. Open **Settings** > **Update & Security** > **For Developers**
2. Enable **"Developer Mode"**
3. Restart your terminal
4. Verify by running: `gwt doctor`

### Option 2: Run as Administrator

If you cannot enable Developer Mode:
- Right-click your terminal (PowerShell, cmd, or Windows Terminal)
- Select **"Run as Administrator"**
- All `gwt` commands will have symlink permissions

### Fallback Behavior

If symlinks cannot be created, GWT automatically falls back through these options:

1. **Symlinks** (requires Developer Mode or Administrator)
2. **Junctions** (directory junctions via `mklink /J` - no special privileges needed)
3. **Copy** (full directory copy as last resort)

**Limitations:**
- Junctions only work for directories (not files)
- Junctions cannot span different drives (C: to D:)
- File copies consume more disk space and don't reflect changes

Check symlink support with:
```bash
gwt doctor
```

## Path Handling

### Supported Path Formats

GWT handles all common Windows path formats:

```bash
# Absolute paths with backslashes
C:\Users\username\projects\myapp

# Absolute paths with forward slashes (recommended)
C:/Users/username/projects/myapp

# UNC paths (network shares - limited support)
\\server\share\projects\myapp

# Relative paths
..\myapp
.\worktrees\feature-x
```

**Recommendation:** Use forward slashes (`/`) for better cross-platform compatibility.

### Long Path Support

Windows traditionally limits paths to 260 characters. Modern Windows supports longer paths if enabled.

**Enable Long Paths:**

**Option 1: Group Policy (Windows Pro/Enterprise)**
1. Open Group Policy Editor: `Win+R`, type `gpedit.msc`
2. Navigate to: **Local Computer Policy** > **Computer Configuration** > **Administrative Templates** > **System** > **Filesystem**
3. Enable **"Enable Win32 long paths"**
4. Restart

**Option 2: Registry (All editions)**
1. Open Registry Editor: `Win+R`, type `regedit`
2. Navigate to: `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\FileSystem`
3. Create or modify DWORD value: `LongPathsEnabled` = `1`
4. Restart

**Option 3: Git Configuration**
```bash
git config --system core.longpaths true
```

**Verify:**
```bash
gwt doctor
```

## Reserved Names

Windows reserves certain filenames that cannot be used as directory or file names. GWT prevents creating worktrees with these names:

### Reserved Device Names
- `CON` (console)
- `PRN` (printer)
- `AUX` (auxiliary)
- `NUL` (null device)
- `COM1` through `COM9` (serial ports)
- `LPT1` through `LPT9` (parallel ports)

**Important:** These restrictions apply **even with extensions**:
- ❌ `CON.txt` - Still reserved
- ❌ `com1.log` - Still reserved (case-insensitive)
- ❌ `feature/AUX` - Still reserved (in any directory)

**Error you'll see:**
```
Error: Invalid branch name 'con': reserved Windows device name
```

### Invalid Characters

Windows also prohibits these characters in filenames:
- `<` `>` `:` `"` `/` `\` `|` `?` `*`

Git branch names already prevent most of these, but be aware when naming branches that will become worktree directories.

## Hook Execution

Hooks are executed differently on Windows compared to Unix systems.

### Default Behavior

GWT executes hooks using `cmd.exe` on Windows:

```yaml
# .worktree.yaml
hooks:
  post_create:
    - "echo Setting up environment"
    - "npm install"
    - "copy .env.example .env"
```

### Using PowerShell

To run PowerShell scripts or commands:

```yaml
hooks:
  post_create:
    - "powershell -ExecutionPolicy Bypass -File setup.ps1"
    - "powershell -Command Get-ChildItem"
```

### Using Batch Files

Batch files (`.bat`, `.cmd`) run directly:

```yaml
hooks:
  post_create:
    - "setup.bat"
    - "scripts\\init.cmd"
```

**Note:** Use double backslashes (`\\`) in YAML strings or forward slashes (`/`).

### Using Bash (Git Bash)

To run bash scripts:

```yaml
hooks:
  post_create:
    - "bash -c 'source .env && npm install'"
    - "bash scripts/setup.sh"
```

### Environment Variables

Hooks receive these environment variables:

```bash
GWT_WORKTREE_PATH=C:\projects\app\worktrees\feature-x
GWT_BRANCH=feature-x
GWT_REPO_ROOT=C:\projects\app
```

Access in batch files:
```batch
@echo off
echo Creating worktree at %GWT_WORKTREE_PATH%
```

Access in PowerShell:
```powershell
Write-Host "Creating worktree at $env:GWT_WORKTREE_PATH"
```

## Docker Compose

### Docker Desktop for Windows

GWT works with Docker Desktop for Windows. Ensure Docker Desktop is installed and running.

**Download:** [Docker Desktop for Windows](https://www.docker.com/products/docker-desktop/)

### Volume Path Conversion

GWT automatically converts Windows paths for Docker:

**Host Path:**
```
C:\projects\app\data
```

**Docker Path (WSL2 backend):**
```
/c/projects/app/data
```

**Docker Path (Hyper-V backend):**
```
/host_mnt/c/projects/app/data
```

This conversion happens automatically in shared mode.

### Port Offsets

Use `port_offset` to avoid port conflicts between worktrees:

```yaml
# .worktree.yaml
docker:
  port_offset: 100
```

**Example:**
- Main worktree: `localhost:3000`
- Feature worktree: `localhost:3100` (offset +100)

### Data Directories

GWT handles data directory sharing on Windows:

```yaml
docker:
  default_mode: shared
  data_directories:
    - data/postgres
    - data/redis
```

**Shared mode:** Creates symlinks (or junctions) to main worktree data
**New mode:** Each worktree gets independent data directories

## Troubleshooting

### "Symlink privilege not held"

**Error:**
```
Error: failed to create symlink: A required privilege is not held by the client.
```

**Solution:**
1. Enable Developer Mode (see [Symlink Support](#symlink-support))
2. OR run terminal as Administrator
3. OR use junction/copy fallback (automatic)

Verify with: `gwt doctor`

### "The filename, directory name, or volume label syntax is incorrect"

**Causes:**
1. Branch name contains reserved Windows device name (CON, PRN, etc.)
2. Branch name contains invalid characters (`< > : " / \ | ? *`)
3. Path exceeds 260 characters (enable long paths)

**Solutions:**
- Rename branch to avoid reserved names
- Enable long path support (see [Long Path Support](#long-path-support))
- Use shorter worktree paths

### "The process cannot access the file because it is being used"

**Causes:**
1. Another program has files open in the worktree
2. Git operations in progress
3. Antivirus software scanning files

**Solutions:**
- Close applications using files in the worktree (check editors, terminals)
- Wait for git operations to complete
- Temporarily disable antivirus scanning on the worktree directory
- Use Process Explorer to find which process has the file locked

### TUI Display Issues

**Problem:** Box drawing characters display incorrectly, or colors are wrong.

**Solutions:**
1. **Use Windows Terminal** (recommended)
   - Best Unicode and color support
   - Download from Microsoft Store

2. **Set Console Font**
   - Right-click terminal title bar > Properties > Font
   - Select a monospace font with Unicode support (e.g., "Consolas", "Cascadia Code")

3. **Enable VT100 Support** (Windows 10+)
   - Already enabled by default on Windows 10 1511+
   - GWT automatically enables it

4. **Use Legacy Console** (fallback)
   - Right-click terminal title bar > Properties
   - Check "Use legacy console"
   - May lose color support

### Docker Volumes Not Mounting

**Problem:** Docker containers can't access volumes.

**Causes:**
1. Docker Desktop not running
2. Drive not shared in Docker Desktop settings
3. Path conversion issues

**Solutions:**
1. Ensure Docker Desktop is running
2. Open Docker Desktop Settings > Resources > File Sharing
3. Add your project drive (C:, D:, etc.)
4. Restart Docker Desktop

### Git Worktree Errors

**Problem:** `git worktree` commands fail.

**Solutions:**
1. Update Git for Windows to latest version (2.25+)
   ```bash
   git --version
   # Should be 2.25.0 or higher
   ```
2. Check repository is not corrupted:
   ```bash
   git fsck
   ```
3. Ensure you're in a git repository:
   ```bash
   git status
   ```

## Performance Tips

### Antivirus Exclusions

Add your repository directories to antivirus exclusions for better performance:

**Windows Defender:**
1. Open Windows Security > Virus & threat protection
2. Scroll to "Virus & threat protection settings" > Manage settings
3. Scroll to "Exclusions" > Add or remove exclusions
4. Add your project directory (e.g., `C:\projects`)

**Other antivirus:** Consult your antivirus documentation.

### Git Configuration

Optimize Git for Windows performance:

```bash
# Enable filesystem cache
git config --global core.fscache true

# Enable parallel operations
git config --global core.preloadindex true

# Use sparse checkout for large repos
git config --global core.sparseCheckout true

# Enable long path support
git config --system core.longpaths true
```

### SSD Storage

For best performance, store repositories on SSD drives, not HDDs.

## Known Limitations

### Windows-Specific Limitations

1. **Case Sensitivity**
   - Windows filesystems are case-insensitive by default
   - Branch names `feature-X` and `feature-x` are treated as the same
   - Enable case sensitivity per-directory (Windows 10 1803+):
     ```powershell
     fsutil.exe file setCaseSensitiveInfo C:\projects\app enable
     ```

2. **Junctions Cannot Span Drives**
   - Junctions only work on the same drive
   - Symlink from C: to D: requires symlink privileges (not junction)
   - Copy fallback will be used if junction fails

3. **File Locking**
   - Windows locks files more aggressively than Unix
   - May prevent deletion of worktrees with running processes
   - Close all applications using worktree files before deleting

4. **Path Length**
   - Default 260-character limit (enable long paths to fix)
   - Affects deeply nested directory structures
   - Worktree paths should be kept reasonably short

### Docker Desktop Limitations

1. **WSL2 vs Hyper-V**
   - Different path mounting strategies
   - Performance varies between backends
   - GWT handles both automatically

2. **Volume Performance**
   - Bind mounts from Windows to Docker can be slow
   - Consider using named volumes for better performance
   - Or enable WSL2 backend and work from WSL2 filesystem

## Advanced Configuration

### Custom Shell for Hooks

Override the default shell (`cmd.exe`) for hook execution:

```yaml
# .worktree.yaml
hooks:
  shell: "powershell -ExecutionPolicy Bypass -Command"
  post_create:
    - "Write-Host 'Setting up...'"
```

Or use Git Bash:

```yaml
hooks:
  shell: "bash -c"
  post_create:
    - "echo 'Setting up...'"
```

### Network Drives

GWT supports UNC paths but with limitations:

```bash
# This works
gwt create feature-x \\server\share\projects\app\worktrees\feature-x

# But symlinks on network drives may fail
# Junction/copy fallback will be used
```

**Recommendation:** Work with local drives for best compatibility.

## Getting Help

1. **Run diagnostics:**
   ```bash
   gwt doctor
   ```
   Reports on symlink support, git version, long path support, etc.

2. **Enable verbose logging:**
   ```bash
   gwt --verbose create feature-x
   ```

3. **Check the docs index:** [README.md](README.md)

4. **Report issues:** [GitHub Issues](https://github.com/Andrewy-gh/gwt/issues)
   - Include output from `gwt doctor`
   - Include Windows version (`winver`)
   - Include Git version (`git --version`)

## Additional Resources

- [Git for Windows Documentation](https://git-scm.com/docs)
- [Windows Terminal Documentation](https://docs.microsoft.com/en-us/windows/terminal/)
- [Docker Desktop for Windows](https://docs.docker.com/desktop/windows/)
- [Git Worktree Documentation](https://git-scm.com/docs/git-worktree)

## Quick Reference

### Commands
```bash
# Check Windows compatibility
gwt doctor

# Create worktree (auto-fallback to junction if needed)
gwt create feature-x

# List worktrees
gwt list

# Delete worktree (handles Windows file locks)
gwt delete feature-x

# Cleanup merged branches
gwt cleanup --merged
```

### Common Issues
| Issue | Quick Fix |
|-------|-----------|
| Symlink error | Enable Developer Mode or run as admin |
| Path too long | Enable long path support |
| Reserved name | Rename branch (avoid CON, PRN, etc.) |
| File locked | Close applications using the files |
| TUI broken | Use Windows Terminal |
| Docker volumes fail | Share drive in Docker Desktop settings |

---

**Last Updated:** 2026-01-16
**GWT Version:** Phase 13
**Windows Version Tested:** Windows 10 21H2, Windows 11 23H2
