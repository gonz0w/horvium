// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"
)

type capture struct {
	sync.Mutex
	limit, used int
	exceeded    bool
	out         bytes.Buffer
	cancel      context.CancelFunc
}
type sink struct {
	capture *capture
	keep    bool
}

func (s sink) Write(p []byte) (int, error) {
	c := s.capture
	c.Lock()
	defer c.Unlock()
	if len(p) > c.limit-c.used {
		c.exceeded = true
		c.cancel()
		return 0, ErrLimit
	}
	c.used += len(p)
	if s.keep {
		return c.out.Write(p)
	}
	return len(p), nil
}

// execute is private: the native reader supplies only fixed commands/arguments.
// Combined stdout/stderr is bounded, and errors never return either stream.
func execute(ctx context.Context, path string, args []string, limit int) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		return nil, ErrLimit
	}
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	output := &capture{limit: limit, cancel: cancel}
	cmd := exec.CommandContext(child, path, args...)
	cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	cmd.Stdout = sink{capture: output, keep: true}
	cmd.Stderr = sink{capture: output}
	cmd.WaitDelay = 250 * time.Millisecond
	configureProcess(cmd)
	err := cmd.Run()
	if output.exceeded {
		return nil, ErrLimit
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrUnavailable
		}
		if errors.Is(err, os.ErrPermission) {
			return nil, ErrPermission
		}
		return nil, ErrExit
	}
	return output.out.Bytes(), nil
}
