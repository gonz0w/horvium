// SPDX-License-Identifier: BSD-3-Clause
package main

import "io"

type witnessInterruptedReader struct {
	source         io.Reader
	reach          func() error
	first, reached bool
	failed         error
}

func (r *witnessInterruptedReader) Read(p []byte) (int, error) {
	if r.failed != nil {
		return 0, r.failed
	}
	if len(p) == 0 {
		return 0, nil
	}
	if !r.first {
		n, e := r.source.Read(p[:min(len(p), 80)])
		if n > 0 {
			r.first = true
		}
		return n, e
	}
	if !r.reached {
		r.reached = true
		r.failed = r.reach()
		if r.failed != nil {
			return 0, r.failed
		}
	}
	return r.source.Read(p)
}

type witnessInterruptedWriter struct {
	target  io.Writer
	reach   func() error
	reached bool
	failed  error
}

func (w *witnessInterruptedWriter) Write(p []byte) (int, error) {
	if w.failed != nil {
		return 0, w.failed
	}
	if w.reached || len(p) == 0 {
		return w.target.Write(p)
	}
	prefix := min(len(p), 80)
	n, e := w.target.Write(p[:prefix])
	if e != nil || n != prefix {
		if e == nil {
			e = io.ErrShortWrite
		}
		w.failed = e
		return n, e
	}
	w.reached = true
	w.failed = w.reach()
	if w.failed != nil {
		return n, w.failed
	}
	m, e := w.target.Write(p[prefix:])
	return n + m, e
}
func (w *witnessInterruptedWriter) CloseWrite() error {
	if w.failed != nil {
		return w.failed
	}
	if closer, ok := w.target.(interface{ CloseWrite() error }); ok {
		return closer.CloseWrite()
	}
	return nil
}
