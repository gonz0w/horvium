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

// executeNative accepts fixed projections or the private validated property-set
// control. Tests may use their own reexecuted binary; requests never supply an
// executable or argument. Control errors do not establish executor quiescence.
func executeNative(ctx context.Context, path string, args []string, limit int) (readOutput, error) {
	return executePrepared(ctx, path, args, limit, prepareNativeCommand(ctx, path, args))
}

// Private native fault fixtures can hold proof acknowledgement while retaining
// this exact production guard preparation and original command owner.
func prepareNativeCommand(ctx context.Context, path string, args []string) func(*exec.Cmd, *processControl) func(error) error {
	return func(cmd *exec.Cmd, control *processControl) func(error) error {
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
		control.stop = func() error { return cmd.Process.Signal(syscall.SIGALRM) }
		groupProof := control.proof
		control.proof = func(waitErr error, deadline time.Time) error {
			// A signalled/lost guard is not a completed escaped-subtree drain.
			// Keep the original pending slot; no replacement can infer success.
			if cmd.ProcessState == nil || !nativeGuardDrainStatus(cmd.ProcessState.Exited(), cmd.ProcessState.ExitCode()) {
				return ErrCleanupDebt
			}
			return groupProof(waitErr, deadline)
		}
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
	}
}
