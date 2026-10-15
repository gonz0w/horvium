// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"testing"
)

type witnessPrefixWriter struct {
	bytes.Buffer
	terminal error
	short    bool
}

func (w *witnessPrefixWriter) Write(b []byte) (int, error) {
	n := min(80, len(b))
	w.Buffer.Write(b[:n])
	if w.short {
		return n, nil
	}
	return n, w.terminal
}

func TestDashboardWitnessForwardFailureRetainsAcceptedPrefix(t *testing.T) {
	for _, tc := range []struct {
		name, class string
		err         error
		short       bool
		errno       int
	}{
		{"partial-error", "io", errors.New("private path and producer data must stay out of metadata"), false, 0},
		{"short-write", "short-write", nil, true, 0},
		{"cancelled", "cancelled", context.Canceled, false, 0},
		{"deadline", "deadline", os.ErrDeadlineExceeded, false, 0},
		{"broken-pipe", "syscall", fmt.Errorf("wrapped: %w", syscall.EPIPE), false, int(syscall.EPIPE)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, e := newDashboardWitnessStore(witnessPrivateDirectory(t))
			if e != nil {
				t.Fatal(e)
			}
			defer store.close()
			raw := witnessFrames(witnessGolden())
			ref, e := store.capture(bytes.NewReader(raw), witnessTestID, "native-small", "management-1", "helper-1")
			if e != nil {
				t.Fatal(e)
			}
			sink := &witnessPrefixWriter{terminal: tc.err, short: tc.short}
			if store.forward(ref, sink) == nil || !store.failed || ref.Delivered || ref.ForwardSHA256 != "" {
				t.Fatal("failed delivery accepted")
			}
			encoded, e := json.Marshal(ref)
			if e != nil {
				t.Fatal(e)
			}
			var receipt struct {
				Forward *struct {
					Stage  string
					Bytes  int64
					SHA256 string
					Error  struct {
						Class string
						Errno int
					}
					WriteHalfClosed bool `json:"write_half_closed"`
				}
			}
			if json.Unmarshal(encoded, &receipt) != nil || receipt.Forward == nil {
				t.Fatal("partial forwarding diagnostic missing")
			}
			f := receipt.Forward
			if f.Stage != "copy" || f.Bytes != int64(sink.Len()) || f.SHA256 != fmt.Sprintf("%x", sha256.Sum256(sink.Bytes())) || f.Error.Class != tc.class || f.Error.Errno != tc.errno || f.WriteHalfClosed {
				t.Fatalf("wrong accepted-prefix receipt: %s", encoded)
			}
			if bytes.Contains(encoded, []byte("private path")) {
				t.Fatal("raw error leaked")
			}
			before := bytes.Clone(encoded)
			if store.forward(ref, &bytes.Buffer{}) == nil {
				t.Fatal("failed candidate resumed")
			}
			after, _ := json.Marshal(ref)
			if !bytes.Equal(before, after) {
				t.Fatal("second denial rewrote first forwarding failure")
			}
			if _, e := store.original(ref); e == nil {
				t.Fatal("partial original became eligible")
			}
		})
	}
}

func TestDashboardWitnessHalfCloseRetainsFullWriteWithoutDelivery(t *testing.T) {
	store, e := newDashboardWitnessStore(witnessPrivateDirectory(t))
	if e != nil {
		t.Fatal(e)
	}
	defer store.close()
	raw := witnessFrames(witnessGolden())
	ref, e := store.capture(bytes.NewReader(raw), witnessTestID, "native-small", "management-1", "helper-1")
	if e != nil {
		t.Fatal(e)
	}
	if store.forward(ref, &witnessHalfCloseFailure{}) == nil {
		t.Fatal("half-close failure accepted")
	}
	encoded, _ := json.Marshal(ref)
	var receipt struct {
		Forward *struct {
			Stage           string
			Bytes           int64
			SHA256          string
			Error           struct{ Class string }
			WriteHalfClosed bool `json:"write_half_closed"`
		}
	}
	if json.Unmarshal(encoded, &receipt) != nil || receipt.Forward == nil {
		t.Fatal("half-close receipt missing")
	}
	f := receipt.Forward
	if f.Stage != "close-write" || f.Bytes != int64(len(raw)) || f.SHA256 != ref.RawSHA256 || f.Error.Class != "closed" || f.WriteHalfClosed || ref.Delivered {
		t.Fatalf("full write incorrectly implies EOF/delivery: %s", encoded)
	}
}

var _ io.Writer = (*witnessPrefixWriter)(nil)
