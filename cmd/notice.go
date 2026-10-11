package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/nvizble/Lightyear42/internal/config"
	"github.com/nvizble/Lightyear42/internal/services"
	"github.com/nvizble/Lightyear42/internal/tui"
	"github.com/nvizble/Lightyear42/internal/update"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// newNotifier keeps the release lookup in the cache folder.
func newNotifier() (*update.Notifier, error) {
	paths, err := config.ResolvePaths()
	if err != nil {
		return nil, err
	}
	return update.NewNotifier(filepath.Join(paths.CacheDir, "update-check.json"),
		services.DefaultGitHubOwner, services.DefaultGitHubRepo), nil
}

// noticeEnabled is off for builds that aren't releases, in CI, when the
// user turns it off (LIGHTYEAR_NO_UPDATE_NOTIFIER) and when stderr isn't a
// terminal (scripts and pipes).
func noticeEnabled() bool {
	return update.IsReleaseVersion(Version) && os.Getenv("LIGHTYEAR_NO_UPDATE_NOTIFIER") == "" &&
		os.Getenv("CI") == "" && term.IsTerminal(int(os.Stderr.Fd()))
}

// noticeSkips are the commands that never show the notice: update does
// the job itself, version prints the version, and completion output is
// read by the shell.
var noticeSkips = map[string]bool{
	"update": true, "version": true, "completion": true, "help": true, "install": true, updateCheckCmd: true,
	cobra.ShellCompRequestCmd: true, cobra.ShellCompNoDescRequestCmd: true,
}

// startUpdateNotice returns what to run when the command ends: print the
// notice when the last lookup found a newer release. When the lookup is due
// (once a day), it starts `lightyear __update-check` in the background: a
// process of its own, so a quick command neither waits for GitHub nor kills
// the lookup by exiting; the notice shows from the next command on. It
// returns nil when there's nothing to do.
func startUpdateNotice(root *cobra.Command, args []string, out io.Writer) func() {
	if !noticeEnabled() {
		return nil
	}
	target, _, err := root.Find(args)
	if err != nil || target == nil || noticeSkips[target.Name()] {
		return nil
	}
	n, err := newNotifier()
	if err != nil {
		return nil
	}
	if n.Stale(Version) && n.Touch(Version) == nil {
		if exe, err := os.Executable(); err == nil {
			check := exec.Command(exe, updateCheckCmd)
			check.Stdin, check.Stdout, check.Stderr = nil, nil, nil
			detach(check)
			if check.Start() == nil {
				_ = check.Process.Release()
			}
		}
	}
	if target == root {
		return nil // the full-screen app shows it in its tab bar
	}
	latest := n.Newer(Version)
	if latest == "" {
		return nil
	}
	return func() { fmt.Fprintln(out, tui.RenderUpdateNotice(Version, latest)) }
}

// updateCheckCmd is the hidden command that does the day's lookup.
const updateCheckCmd = "__update-check"

func newUpdateCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:    updateCheckCmd,
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			n, err := newNotifier()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			return n.Refresh(ctx, Version)
		},
	}
}

// availableUpdate is the newer release the last lookup found, or "".
func availableUpdate() string {
	if !update.IsReleaseVersion(Version) || os.Getenv("LIGHTYEAR_NO_UPDATE_NOTIFIER") != "" {
		return ""
	}
	n, err := newNotifier()
	if err != nil {
		return ""
	}
	return n.Newer(Version)
}
