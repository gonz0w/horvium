// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

// executeNative accepts only projections supplied by readNative. Tests may use
// their own reexecuted binary; requests never supply an executable or argument.
func executeNative(ctx context.Context, path string, args []string, limit int) (readOutput, error) {
	return executePrepared(ctx, path, args, limit, func(cmd *exec.Cmd) func(error) error {
		// The base reaper stays outside the probe subtree. On helper death,
		// SIGALRM asks it to SIGKILL all descendants, including other groups.
		// Go installs the death signal before exec and checks a parent-exit race.
		duration := 10 * time.Second
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < duration {
			duration = time.Until(deadline)
		}
		if duration < time.Millisecond {
			duration = time.Millisecond
		}
		cmd.Path = "/usr/bin/timeout"
		cmd.Args = append([]string{cmd.Path, "-s", "KILL", strconv.FormatFloat(duration.Seconds(), 'f', 6, 64), path}, args...)
		cmd.SysProcAttr.Pdeathsig = syscall.SIGALRM
		cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGALRM) }
		return func(err error) error {
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				switch exit.ExitCode() {
				case 124:
					return context.DeadlineExceeded
				case 126:
					return ErrPermission
				case 127:
					return ErrUnavailable
				}
			}
			return commandError(err)
		}
	})
}
