//go:build !freebsd && !darwin && !linux && !openbsd && !netbsd

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"os/exec"
	"time"
)

const processOwnershipSupported = false

func configureProcess(cmd *exec.Cmd) processControl {
	return processControl{stop: func() error { return ErrUnsupported }, proof: func(error, time.Time) error { return ErrUnsupported }}
}
func observeChildExit(context.Context, int) error { return ErrUnsupported }
