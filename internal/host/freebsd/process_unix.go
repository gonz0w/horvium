//go:build freebsd || darwin || linux || openbsd || netbsd

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func configureProcess(cmd *exec.Cmd) func() error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	killGroup := func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.Cancel = killGroup
	return func() error {
		// Run has joined its context watcher and I/O copiers, but WaitDelay
		// only closes inherited pipes. A parent can exit leaving children in
		// our isolated group; fixed inventory probes never own background work.
		err := killGroup()
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return err
	}
}
