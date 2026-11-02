// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"time"
)

const processOwnershipSupported = true

func observeChildExit(ctx context.Context, pid int) error {
	if pid <= 1 {
		return ErrCleanupDebt
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var info unix.Siginfo
		err := unix.Waitid(unix.P_PID, pid, &info, unix.WEXITED|unix.WNOHANG|unix.WNOWAIT, nil)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if errors.Is(err, unix.ECHILD) {
			return errors.Join(errReservationLost, ErrCleanupDebt, err)
		}
		if err != nil {
			return err
		}
		if info.Signo != 0 {
			return nil
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
