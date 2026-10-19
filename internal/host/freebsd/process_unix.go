//go:build freebsd || darwin || linux || openbsd || netbsd

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func configureProcess(cmd *exec.Cmd) processControl {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return processControl{
		stop: func() error {
			if cmd.Process == nil {
				return os.ErrProcessDone
			}
			if cmd.Process.Pid <= 1 {
				return ErrCleanupDebt
			}
			err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			// Darwin can reject a zombie-only group. Only the independent post-Wait
			// read-only absence proof may make that retirement clean.
			if errors.Is(err, syscall.ESRCH) || errors.Is(err, syscall.EPERM) {
				return nil
			}
			return err
		},
		proof: func(waitErr error, deadline time.Time) error {
			if cmd.ProcessState == nil {
				return ErrCleanupDebt
			}
			return observeGroupRetired(cmd.Process.Pid, deadline)
		},
	}
}

func observeGroupRetired(group int, deadline time.Time) error {
	if group <= 1 {
		return ErrCleanupDebt
	}
	for {
		err := syscall.Kill(-group, 0) // Observation only; never recover signal authority.
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		if err != nil || !time.Now().Before(deadline) {
			return ErrCleanupDebt
		}
		time.Sleep(min(10*time.Millisecond, max(0, time.Until(deadline))))
	}
}
