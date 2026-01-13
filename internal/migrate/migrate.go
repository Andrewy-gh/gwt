package migrate

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/Andrewy-gh/gwt/internal/config"
	"github.com/Andrewy-gh/gwt/internal/output"
)

const defaultTimeout = 5 * time.Minute

// Run executes migrations for the given worktree
func Run(opts RunOptions, cfg *config.MigrationsConfig) (*Result, error) {
	result := &Result{}

	// Step 1: Detect migration tool
	tool, err := Detect(opts.WorktreePath, cfg)
	if err != nil {
		return nil, &DetectionError{Path: opts.WorktreePath, Err: err}
	}

	if tool == nil {
		result.Skipped = true
		result.Reason = "no migration tool detected"
		return result, nil
	}

	result.Tool = tool

	// Step 2: Handle raw SQL (no auto-execution)
	if tool.Name == "sql" {
		result.Skipped = true
		result.Reason = fmt.Sprintf("raw SQL files found in %s - manual execution required", tool.Path)
		return result, nil
	}

	// Step 3: Check container readiness (unless skipped)
	if !opts.SkipContainerCheck {
		status, err := CheckDatabaseContainer(opts.WorktreePath)
		if err != nil {
			output.Verbose(fmt.Sprintf("Container check failed: %v", err))
		}
		if status != nil && !status.Running {
			result.Skipped = true
			result.Reason = fmt.Sprintf("database container %q is not running", status.Name)
			return result, nil
		}
	}

	// Step 4: Dry run mode
	if opts.DryRun {
		result.Skipped = true
		result.Reason = fmt.Sprintf("would run: %s", strings.Join(tool.Command, " "))
		return result, nil
	}

	// Step 5: Execute migrations
	output.Info(fmt.Sprintf("Running %s migrations...", tool.Name))
	output.Verbose(fmt.Sprintf("Command: %s", strings.Join(tool.Command, " ")))

	stdout, stderr, exitCode, err := runCommand(opts, tool)
	result.Output = stdout + stderr

	if err != nil || exitCode != 0 {
		result.Success = false
		result.Error = &MigrationError{
			Tool:     tool.Name,
			Command:  tool.Command,
			Stderr:   stderr,
			ExitCode: exitCode,
		}
		return result, nil // Non-fatal, return result with error info
	}

	result.Success = true
	return result, nil
}

func runCommand(opts RunOptions, tool *MigrationTool) (stdout, stderr string, exitCode int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, tool.Command[0], tool.Command[1:]...)
	cmd.Dir = opts.WorktreePath

	// Set up pipes for streaming
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return "", "", -1, err
	}

	// Stream output if verbose
	var stdoutBuf, stderrBuf strings.Builder

	done := make(chan struct{})
	go func() {
		streamOutput(stdoutPipe, &stdoutBuf, opts.Verbose, "")
		done <- struct{}{}
	}()
	go func() {
		streamOutput(stderrPipe, &stderrBuf, opts.Verbose, "stderr: ")
		done <- struct{}{}
	}()

	// Wait for output goroutines
	<-done
	<-done

	err = cmd.Wait()
	exitCode = 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
		err = nil // Don't treat non-zero exit as error
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode, err
}

func streamOutput(r io.Reader, buf *strings.Builder, verbose bool, prefix string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteString("\n")
		if verbose {
			output.Verbose(prefix + line)
		}
	}
}
