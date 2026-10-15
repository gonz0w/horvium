// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
)

// Private fixture IPC is one bounded framed control at a time. The channel is
// inherited from an owned root parent; it is never an externally bound listener.
func witnessControlWrite(w io.Writer, v any) error {
	b, e := json.Marshal(v)
	if e != nil || len(b) == 0 || len(b) > witnessFrameCap {
		return errDashboardWitness
	}
	var h [4]byte
	binary.BigEndian.PutUint32(h[:], uint32(len(b)))
	if n, e := w.Write(h[:]); e != nil || n != 4 {
		return errDashboardWitness
	}
	if n, e := w.Write(b); e != nil || n != len(b) {
		return errDashboardWitness
	}
	return nil
}
func witnessControlRead(r io.Reader, v any) error {
	var budget witnessWireBudget
	b, e := budget.frame(r)
	if e != nil {
		return e
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if _, e := witnessJSONValue(d, 0); e != nil {
		return e
	}
	if _, e := d.Token(); e != io.EOF {
		return errDashboardWitness
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return errDashboardWitness
	}
	return nil
}
func witnessControlFrame(payload []byte) []byte {
	b := make([]byte, 4, len(payload)+4)
	binary.BigEndian.PutUint32(b, uint32(len(payload)))
	return append(b, payload...)
}
func witnessControlAdverse() map[string][]byte {
	return map[string][]byte{
		"empty-frame":      {0, 0, 0, 0},
		"oversized-header": {0, 1, 0, 1},
		"truncated-header": {0, 0},
		"truncated-body":   {0, 0, 0, 9, '{'},
		"duplicate-field":  witnessControlFrame([]byte(`{"operation":"readback","operation":"stop"}`)),
		"unknown-field":    witnessControlFrame([]byte(`{"operation":"readback","untrusted":true}`)),
		"trailing-json":    witnessControlFrame([]byte(`{"operation":"readback"} {}`)),
		"control-eof":      {},
	}
}
