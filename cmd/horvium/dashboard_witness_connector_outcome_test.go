// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWitnessConnectorBoundaryWait(t *testing.T) {
	for _, tc := range []struct {
		name, action string
		exited       bool
		code, signal int
		want         bool
	}{
		{"released", "release", true, 0, 0, true},
		{"eof", "channel-eof", true, 1, 0, true},
		{"term", "sigterm", false, 0, 15, true},
		{"kill", "sigkill", false, 0, 9, true},
		{"success is not eof", "channel-eof", true, 0, 0, false},
		{"signal is not eof", "channel-eof", false, 0, 9, false},
		{"wrong signal", "sigterm", false, 0, 9, false},
		{"failed release", "release", true, 1, 0, false},
		{"unknown action", "other", true, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := witnessConnectorWaitMatches(tc.action, tc.exited, tc.code, tc.signal); got != tc.want {
				t.Fatalf("wait classification=%t, want %t", got, tc.want)
			}
		})
	}
}

func TestWitnessConnectorSignalReceiptCanonical(t *testing.T) {
	for _, signal := range []int{9, 15} {
		original := witnessConnectorWaitRepresentation(true, false, -1, signal, uint32(signal))
		raw, e := json.Marshal(original)
		if e != nil {
			t.Fatal(e)
		}
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if _, e := witnessJSONValue(d, 0); e != nil {
			t.Fatal("original signalled wait rejected by unchanged control grammar", e)
		}
		var reopened witnessConnectorWaitRecord
		if json.Unmarshal(raw, &reopened) != nil || reopened != original || reopened.ExitCode != 0 || reopened.Exited || !reopened.Present || reopened.Signal != signal || reopened.Status != uint32(signal) {
			t.Fatal("signal wait identity lost")
		}
	}
}

func TestWitnessConnectorInvalidWaitRemainsUnavailable(t *testing.T) {
	for _, input := range []struct {
		present, exited bool
		code, signal    int
	}{
		{false, true, 0, 0}, {true, true, -1, 0}, {true, false, -1, -1},
	} {
		r := witnessConnectorWaitRepresentation(input.present, input.exited, input.code, input.signal, 0)
		if r.Present {
			t.Fatal("missing/invalid wait became available")
		}
	}
}
