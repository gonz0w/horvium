//go:build !freebsd && !darwin && !linux && !openbsd && !netbsd

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "os/exec"

func configureProcess(cmd *exec.Cmd) {}
