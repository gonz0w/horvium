// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"net"
	"os"
	"syscall"
	"time"
)

// Diagnostics are bounded observations, never publication or retry authority.
// Preserve wrapped OS errno without retaining paths, inventories or raw errors.
type witnessForwardError struct {
	Class string `json:"class"`
	Errno uint64 `json:"errno,omitempty"`
}

func witnessClassifyForwardError(e error) witnessForwardError {
	r := witnessForwardError{Class: "io"}
	var errno syscall.Errno
	if errors.As(e, &errno) {
		r.Errno = uint64(errno)
	}
	var timeout net.Error
	switch {
	case e == nil:
		r.Class = ""
	case errors.Is(e, context.DeadlineExceeded), errors.Is(e, os.ErrDeadlineExceeded), errors.As(e, &timeout) && timeout.Timeout():
		r.Class = "deadline"
	case errors.Is(e, context.Canceled):
		r.Class = "cancelled"
	case errors.Is(e, io.ErrShortWrite):
		r.Class = "short-write"
	case errors.Is(e, io.ErrClosedPipe), errors.Is(e, net.ErrClosed), errors.Is(e, os.ErrClosed):
		r.Class = "closed"
	case r.Errno != 0:
		r.Class = "syscall"
	}
	return r
}

type witnessForwardReceipt struct {
	Stage           string              `json:"stage"`
	Bytes           int64               `json:"bytes"`
	SHA256          string              `json:"sha256"`
	Error           witnessForwardError `json:"error"`
	WriteHalfClosed bool                `json:"write_half_closed"`
	ElapsedNS       int64               `json:"elapsed_ns"`
}

// The observer cannot inspect the manager's internal context. These are its own
// original collection deadline and context observations, separate from peer wait.
type witnessForwardBoundary struct {
	ID, Management, Helper                                        string
	StartedNS, DeadlineNS                                         int64
	CapturedElapsedNS, ForwardStartedElapsedNS, FinishedElapsedNS int64
	DeadlineReached, ContextDeadlineExceeded, ContextCancelled    bool
	Forward                                                       *witnessForwardReceipt
}

// MultiWriter hashes none of a partial Write when the first sink returns an
// error. Hash exactly the prefix acknowledged by that sink, including n>0,error.
// Accepted socket bytes do not establish peer consumption or publication.
type witnessForwardWriter struct {
	target io.Writer
	hash   hash.Hash
	bytes  int64
}

func (w *witnessForwardWriter) Write(b []byte) (int, error) {
	n, e := w.target.Write(b)
	if n < 0 || n > len(b) {
		return 0, io.ErrShortWrite
	}
	w.hash.Write(b[:n])
	w.bytes += int64(n)
	if e == nil && n != len(b) {
		e = io.ErrShortWrite
	}
	return n, e
}

func newWitnessForwardWriter(w io.Writer) *witnessForwardWriter {
	return &witnessForwardWriter{target: w, hash: sha256.New()}
}

func (w *witnessForwardWriter) receipt(stage string, e error, closed bool, start time.Time) *witnessForwardReceipt {
	return &witnessForwardReceipt{Stage: stage, Bytes: w.bytes, SHA256: fmt.Sprintf("%x", w.hash.Sum(nil)), Error: witnessClassifyForwardError(e), WriteHalfClosed: closed, ElapsedNS: time.Since(start).Nanoseconds()}
}
