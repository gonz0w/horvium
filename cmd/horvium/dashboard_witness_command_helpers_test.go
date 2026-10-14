// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"time"
)

type witnessCommandDiagnostic struct {
	Reason                                                      string
	StatePresent                                                bool
	ExitCode                                                    int
	DeadlineExceeded, Cancelled, StdoutOverflow, StderrOverflow bool
	ElapsedNS                                                   int64
	Stderr                                                      string
}

type witnessCommandOutput struct {
	data     []byte
	limit    int
	overflow bool
}

func (w *witnessCommandOutput) Write(p []byte) (int, error) {
	n := min(len(p), w.limit-len(w.data))
	w.data = append(w.data, p[:n]...)
	if n != len(p) {
		w.overflow = true
		return n, io.ErrShortBuffer
	}
	return n, nil
}

// Retain only bounded original command output and fixed error classifications.
// No error string/path or process inventory is emitted in the public diagnostic.
func witnessRunSampleCommand(ctx context.Context, cmd *exec.Cmd) ([]byte, witnessCommandDiagnostic, error) {
	stdout := witnessCommandOutput{limit: witnessFrameCap}
	stderr := witnessCommandOutput{limit: 4096}
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	started := time.Now()
	e := cmd.Run()
	d := witnessCommandDiagnostic{ExitCode: -1, ElapsedNS: int64(time.Since(started)), Stderr: string(stderr.data), StdoutOverflow: stdout.overflow, StderrOverflow: stderr.overflow, DeadlineExceeded: errors.Is(ctx.Err(), context.DeadlineExceeded), Cancelled: errors.Is(ctx.Err(), context.Canceled)}
	if cmd.ProcessState != nil {
		d.StatePresent = true
		d.ExitCode = cmd.ProcessState.ExitCode()
	}
	switch {
	case stdout.overflow || stderr.overflow:
		d.Reason = "overflow"
	case d.DeadlineExceeded:
		d.Reason = "deadline"
	case d.Cancelled:
		d.Reason = "cancelled"
	case e != nil:
		var exit *exec.ExitError
		if cmd.Process == nil {
			d.Reason = "start"
		} else if errors.As(e, &exit) {
			d.Reason = "exit"
		} else {
			d.Reason = "io"
		}
	}
	if d.Reason != "" {
		return stdout.data, d, errDashboardWitness
	}
	return stdout.data, d, nil
}
