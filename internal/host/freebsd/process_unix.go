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
	return pollGroupRetired(deadline, func() error {
		return syscall.Kill(-group, 0) // Read only; never recover signal authority.
	})
}

func pollGroupRetired(deadline time.Time, observe func() error) error {
	for {
		err := observe()
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		// EPERM is unknown, including Darwin's transient zombie-only groups.
		// Poll only within the same reserve; only actual ESRCH proves absence.
		if (err != nil && !errors.Is(err, syscall.EPERM)) || !time.Now().Before(deadline) {
			return ErrCleanupDebt
		}
		time.Sleep(min(10*time.Millisecond, max(0, time.Until(deadline))))
	}
}
