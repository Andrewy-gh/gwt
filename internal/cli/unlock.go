package cli

import (
	"fmt"

	"github.com/Andrewy-gh/gwt/internal/create"
	"github.com/Andrewy-gh/gwt/internal/output"
	"github.com/spf13/cobra"
)

type unlockOptions struct {
	Force bool
}

var unlockOpts unlockOptions

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Remove the gwt operation lock",
	Long: `Remove the gwt operation lock for the current repository.

By default this only removes a missing, stale, or unreadable lock. Use --force
to remove a lock even when the owning PID still appears to be active.`,
	RunE: runUnlock,
}

func init() {
	unlockCmd.Flags().BoolVarP(&unlockOpts.Force, "force", "f", false, "remove the lock even if the owning PID still appears active")
	rootCmd.AddCommand(unlockCmd)
}

func runUnlock(cmd *cobra.Command, args []string) error {
	repoPath, err := getRepoPath(".")
	if err != nil {
		return err
	}

	exists, err := create.LockExists(repoPath)
	if err != nil {
		return err
	}
	if !exists {
		output.Info("No gwt operation lock found.")
		return nil
	}

	info, infoErr := create.GetLockInfo(repoPath)
	locked, lockedErr := create.IsLocked(repoPath)
	if lockedErr != nil {
		return lockedErr
	}

	switch {
	case infoErr != nil:
		output.Warning(fmt.Sprintf("Lock file exists but could not be read: %v", infoErr))
		output.Info("Removing unreadable gwt operation lock.")

	case locked && !unlockOpts.Force:
		return fmt.Errorf(
			"gwt operation lock is still active (PID %d, started %s); rerun with --force only if you are sure that process is stale",
			info.PID,
			info.StartTime.Format("2006-01-02T15:04:05Z07:00"),
		)

	case locked:
		output.Warning(fmt.Sprintf(
			"Forcing removal of active gwt operation lock held by PID %d (started %s).",
			info.PID,
			info.StartTime.Format("2006-01-02T15:04:05Z07:00"),
		))

	default:
		output.Info(fmt.Sprintf(
			"Removing stale gwt operation lock from PID %d (started %s).",
			info.PID,
			info.StartTime.Format("2006-01-02T15:04:05Z07:00"),
		))
	}

	if err := create.ForceUnlock(repoPath); err != nil {
		return err
	}

	output.Success("Removed gwt operation lock.")
	return nil
}
