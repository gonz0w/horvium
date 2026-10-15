// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestDashboardWitnessControlBounds(t *testing.T) {
	type command struct {
		Operation string `json:"operation"`
	}
	for name, raw := range witnessControlAdverse() {
		t.Run(name, func(t *testing.T) {
			var c command
			if witnessControlRead(bytes.NewReader(raw), &c) == nil {
				t.Fatal("malformed control accepted")
			}
		})
	}
	payload := []byte(`{"operation":"readback"}`)
	payload = append(payload, bytes.Repeat([]byte(" "), witnessFrameCap-len(payload))...)
	reader := bytes.NewReader(append(witnessControlFrame(payload), witnessControlFrame([]byte(`{"operation":"stop"}`))...))
	for _, want := range []string{"readback", "stop"} {
		var c command
		if witnessControlRead(reader, &c) != nil || c.Operation != want {
			t.Fatal("exact boundary or next frame lost")
		}
	}
	if reader.Len() != 0 {
		t.Fatal("frame cursor")
	}
	source := &witnessControlCount{source: bytes.NewReader([]byte("unread"))}
	var c command
	if witnessControlRead(io.MultiReader(bytes.NewReader([]byte{0, 1, 0, 1}), source), &c) == nil || source.reads != 0 {
		t.Fatal("over-limit header read body")
	}
	var out bytes.Buffer
	if witnessControlWrite(&out, strings.Repeat("x", witnessFrameCap)) == nil || out.Len() != 0 {
		t.Fatal("oversized write was partially emitted")
	}
	deep := strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34)
	if witnessControlRead(bytes.NewReader(witnessControlFrame([]byte(deep))), new(any)) == nil {
		t.Fatal("depth bound")
	}
}

type witnessControlCount struct {
	source io.Reader
	reads  int
}

func (r *witnessControlCount) Read(b []byte) (int, error) { r.reads++; return r.source.Read(b) }
