//go:build freebsd || darwin || openbsd || netbsd

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"time"
)

const processOwnershipSupported = true

// Kqueue observes this fresh, unreaped child; it does not consume the only wait.
// The observer owns/closes its descriptor itself. No concurrent numeric-fd close
// can race Kevent and accidentally close a subsequently reused descriptor.
func observeChildExit(ctx context.Context, pid int) (result error) {
	if pid <= 1 {
		return ErrCleanupDebt
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	queue, err := unix.Kqueue()
	if err != nil {
		return err
	}
	defer func() {
		if err := unix.Close(queue); err != nil {
			result = errors.Join(result, ErrCleanupDebt, err)
		}
	}()
	change := unix.Kevent_t{}
	unix.SetKevent(&change, pid, unix.EVFILT_PROC, unix.EV_ADD|unix.EV_ENABLE|unix.EV_ONESHOT)
	change.Fflags = unix.NOTE_EXIT
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err = unix.Kevent(queue, []unix.Kevent_t{change}, nil, nil)
	// A child that exited before registration is still unreaped/reserved. The
	// actual Wait and independent retirement proof remain mandatory.
	if errors.Is(err, unix.ESRCH) {
		return nil
	}
	if err != nil {
		return err
	}
	events := make([]unix.Kevent_t, 1)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		timeout := unix.NsecToTimespec(int64(25 * time.Millisecond))
		n, err := unix.Kevent(queue, nil, events, &timeout)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		if events[0].Flags&unix.EV_ERROR != 0 {
			return unix.Errno(events[0].Data)
		}
		if events[0].Fflags&unix.NOTE_EXIT != 0 {
			return nil
		}
	}
}
