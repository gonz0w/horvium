// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

var probeCommands commandSlot

type processControl struct {
	started func() // default nil; private tests publish an already-owned born Cmd

	stop  func() error
	proof func(error, time.Time) error
}

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
		// An over-budget chunk exhausts this collection's remaining allowance.
		c.used = c.limit
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
func execute(ctx context.Context, path string, args []string, limit int) (readOutput, error) {
	return executePrepared(ctx, path, args, limit, nil)
}

func executePrepared(ctx context.Context, path string, args []string, limit int, prepare func(*exec.Cmd, *processControl) func(error) error) (readOutput, error) {
	if err := ctx.Err(); err != nil {
		return readOutput{}, err
	}
	if !processOwnershipSupported {
		return readOutput{}, ErrUnsupported
	}
	if limit <= 0 {
		return readOutput{}, ErrLimit
	}
	child, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	output := &capture{limit: limit, cancel: cancel}
	// This owner alone controls cancellation. CommandContext/watchCtx/WaitDelay
	// could kill the live native reaper while its escaped-subtree drain is unknown.
	cmd := exec.Command(path, args...)
	cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	control := configureProcess(cmd)
	classify := commandError
	if prepare != nil {
		classify = prepare(cmd, &control)
	}
	err := ownedCommand(child, &probeCommands, commandCleanupReserve, func() (commandCalls, error) {
		calls := commandCalls{}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return calls, err
		}
		calls.close = append(calls.close, stdout.Close)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return calls, err
		}
		calls.close = append(calls.close, stderr.Close)
		if err := child.Err(); err != nil {
			return calls, err
		}
		if err := cmd.Start(); err != nil {
			return calls, err
		}
		// The slot already retains cmd before Start. From this point every path
		// has all positive operations and its sole wait bound to this fresh child.
		calls.stop, calls.wait, calls.proof = control.stop, cmd.Wait, control.proof
		calls.observe = func(ctx context.Context) error { return observeChildExit(ctx, cmd.Process.Pid) }
		for _, stream := range []struct {
			read io.ReadCloser
			keep bool
		}{{stdout, true}, {stderr, false}} {
			done := make(chan error, 1)
			calls.readers = append(calls.readers, done)
			go func(stream io.ReadCloser, keep bool) {
				_, err := io.Copy(sink{capture: output, keep: keep}, stream)
				done <- err
			}(stream.read, stream.keep)
		}
		if control.started != nil {
			control.started()
		}
		return calls, nil
	})
	output.Lock()
	defer output.Unlock()
	result := readOutput{consumed: output.used}
	if output.exceeded {
		return result, errors.Join(ErrLimit, cleanupDebt(err))
	}
	if ctx.Err() != nil {
		return result, errors.Join(ctx.Err(), cleanupDebt(err))
	}
	if child.Err() != nil {
		return result, errors.Join(child.Err(), cleanupDebt(err))
	}
	if err != nil {
		return result, errors.Join(classify(err), cleanupDebt(err))
	}
	result.data = output.out.Bytes()
	return result, nil
}

func cleanupDebt(err error) error {
	if errors.Is(err, ErrCleanupDebt) {
		return ErrCleanupDebt
	}
	return nil
}

func commandError(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return ErrUnavailable
	}
	if errors.Is(err, os.ErrPermission) {
		return ErrPermission
	}
	return ErrExit
}
