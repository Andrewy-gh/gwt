package install

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/Andrewy-gh/gwt/internal/config"
	"github.com/Andrewy-gh/gwt/internal/output"
)

// Install runs dependency installation for all detected package managers
func Install(worktreePath string, cfg *config.DependenciesConfig, opts InstallOptions) (*Result, error) {
	if !cfg.AutoInstall {
		return &Result{Skipped: true, Reason: "auto_install disabled"}, nil
	}

	// Detect package managers
	managers, err := DetectPackageManagers(worktreePath, cfg.Paths)
	if err != nil {
		return nil, fmt.Errorf("detection failed: %w", err)
	}

	if len(managers) == 0 {
		return &Result{Skipped: true, Reason: "no package managers detected"}, nil
	}

	result := &Result{
		Managers: make([]ManagerResult, 0, len(managers)),
	}

	for _, pm := range managers {
		output.Info(fmt.Sprintf("Installing %s dependencies in %s...", pm.Name, pm.Path))

		mrResult := runInstall(pm, opts)
		result.Managers = append(result.Managers, mrResult)

		if mrResult.Success {
			output.Success(fmt.Sprintf("%s install completed", pm.Name))
		} else {
			output.Warning(fmt.Sprintf("%s install failed: %v", pm.Name, mrResult.Error))
		}
	}

	return result, nil
}

// runInstall executes the install command for a single package manager
func runInstall(pm PackageManager, opts InstallOptions) ManagerResult {
	cmd := exec.Command(pm.InstallCmd, pm.InstallArgs...)
	cmd.Dir = pm.Path

	// Create pipes for streaming output
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return ManagerResult{
			Manager: pm.Name,
			Path:    pm.Path,
			Success: false,
			Error:   err,
		}
	}

	// Stream output if callback provided
	var wg sync.WaitGroup
	if opts.OnProgress != nil {
		wg.Add(2)
		go streamLines(stdout, opts.OnProgress, &wg)
		go streamLines(stderr, opts.OnProgress, &wg)
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		wg.Wait()
		done <- cmd.Wait()
	}()

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute // Default 5 minute timeout
	}

	select {
	case err := <-done:
		return ManagerResult{
			Manager: pm.Name,
			Path:    pm.Path,
			Success: err == nil,
			Error:   err,
		}
	case <-time.After(timeout):
		cmd.Process.Kill()
		return ManagerResult{
			Manager: pm.Name,
			Path:    pm.Path,
			Success: false,
			Error:   fmt.Errorf("installation timed out after %v", timeout),
		}
	}
}

// streamLines reads lines from a reader and calls the callback for each line
func streamLines(r io.Reader, callback func(string), wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		callback(scanner.Text())
	}
}
