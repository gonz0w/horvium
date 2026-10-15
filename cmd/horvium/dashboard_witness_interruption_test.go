// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestDashboardWitnessInterruptionKeepsActualPrefix(t *testing.T) {
	raw := bytes.Repeat([]byte{'x'}, 160)
	r := &witnessInterruptedReader{source: bytes.NewReader(raw), reach: func() error { return io.ErrClosedPipe }}
	got, e := io.ReadAll(r)
	if !errors.Is(e, io.ErrClosedPipe) || !bytes.Equal(got, raw[:80]) {
		t.Fatal("capture boundary lost original prefix", len(got), e)
	}
	var sink bytes.Buffer
	w := &witnessInterruptedWriter{target: &sink, reach: func() error { return io.ErrClosedPipe }}
	n, e := w.Write(raw)
	if n != 80 || !errors.Is(e, io.ErrClosedPipe) || !bytes.Equal(sink.Bytes(), raw[:80]) {
		t.Fatal("forward boundary fabricated delivery", n, e)
	}
}
func TestDashboardWitnessInterruptionReleasePreservesBytes(t *testing.T) {
	raw := bytes.Repeat([]byte{'x'}, 160)
	reads, writes := 0, 0
	r := &witnessInterruptedReader{source: bytes.NewReader(raw), reach: func() error { reads++; return nil }}
	got, e := io.ReadAll(r)
	if e != nil || !bytes.Equal(got, raw) || reads != 1 {
		t.Fatal("released read changed source")
	}
	var sink bytes.Buffer
	w := &witnessInterruptedWriter{target: &sink, reach: func() error { writes++; return nil }}
	n, e := w.Write(raw)
	if e != nil || n != len(raw) || !bytes.Equal(sink.Bytes(), raw) || writes != 1 {
		t.Fatal("released write changed source")
	}
}
