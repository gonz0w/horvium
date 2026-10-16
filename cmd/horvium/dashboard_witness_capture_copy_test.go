// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"syscall"
	"testing"
)

type witnessCapturePrefixSink struct {
	bytes.Buffer
	remaining int
	terminal  error
}

func (w *witnessCapturePrefixSink) Write(p []byte) (int, error) {
	n := min(len(p), w.remaining)
	w.remaining -= n
	w.Buffer.Write(p[:n])
	if n < len(p) {
		return n, w.terminal
	}
	return n, nil
}

func TestDashboardWitnessCaptureHashesAcknowledgedStoragePrefix(t *testing.T) {
	raw := bytes.Repeat([]byte("original-storage-prefix/"), 7000)
	for _, tc := range []struct {
		name           string
		accepted       int
		terminal, want error
	}{
		{"normal", len(raw), nil, nil},
		{"empty-enospc", 0, syscall.ENOSPC, syscall.ENOSPC},
		{"first-write-enospc", 8193, syscall.ENOSPC, syscall.ENOSPC},
		{"later-write-enospc", witnessFrameCap + 8193, syscall.ENOSPC, syscall.ENOSPC},
		{"short-without-error", 8193, nil, io.ErrShortWrite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sink := &witnessCapturePrefixSink{remaining: tc.accepted, terminal: tc.terminal}
			n, digest, e := witnessCopyCapture(sink, bytes.NewReader(raw), int64(len(raw)))
			if !errors.Is(e, tc.want) || n != int64(tc.accepted) || !bytes.Equal(sink.Bytes(), raw[:tc.accepted]) || digest != fmt.Sprintf("%x", sha256.Sum256(raw[:tc.accepted])) {
				t.Fatalf("acknowledged prefix lost: bytes=%d want=%d digest=%s error=%v", n, tc.accepted, digest, e)
			}
		})
	}
}

func TestDashboardWitnessCaptureCopyHonorsRemainingLimit(t *testing.T) {
	raw := bytes.Repeat([]byte("captured/"), witnessFrameCap)
	var sink bytes.Buffer
	source := bytes.NewReader(raw)
	n, digest, e := witnessCopyCapture(&sink, source, 8193)
	if e != nil || n != 8193 || source.Len() != len(raw)-8193 || !bytes.Equal(sink.Bytes(), raw[:8193]) || digest != fmt.Sprintf("%x", sha256.Sum256(raw[:8193])) {
		t.Fatal("remaining archive/raw bound changed", n, e)
	}
}
