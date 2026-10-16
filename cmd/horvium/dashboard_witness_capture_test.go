// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

func TestDashboardWitnessCaptureRetainsOriginalIOFailure(t *testing.T) {
	for _, label := range []string{"open", "copy", "sealed"} {
		t.Run(label, func(t *testing.T) {
			directory := witnessPrivateDirectory(t)
			s, e := newDashboardWitnessStore(directory)
			if e != nil {
				t.Fatal(e)
			}
			defer s.close()
			raw := witnessFrames(witnessGolden())
			var source interface{ Read([]byte) (int, error) } = bytes.NewReader(raw)
			if label == "open" {
				if os.WriteFile(filepath.Join(directory, "original-001.wire"), []byte("unadopted"), 0400) != nil {
					t.Fatal("open fault preparation")
				}
			}
			if label == "copy" {
				source = &witnessInterruptedReader{source: bytes.NewReader(raw), reach: func() error { return context.Canceled }}
			}
			ref, e := s.capture(source, witnessTestID, "native-small", "original-management", "original-helper")
			b := s.captureBoundary
			if b == nil || b.ID != witnessTestID || b.Management != "original-management" || b.Helper != "original-helper" || b.Stage != label {
				t.Fatal("capture I/O boundary lost", b)
			}
			switch label {
			case "open":
				if e == nil || ref != nil || !s.failed || b.AcknowledgedBytes != 0 || b.Error.Class != "syscall" || b.Error.Errno != uint64(syscall.EEXIST) {
					t.Fatal("open errno/ref fabricated", b, e)
				}
			case "copy":
				if e == nil || ref == nil || !s.failed || ref.CaptureFailure != "capture-io" || b.AcknowledgedBytes != ref.RawBytes || b.Error.Class != "cancelled" || b.Error.Errno != 0 {
					t.Fatal("original read cancellation lost", b, e)
				}
			case "sealed":
				if e != nil || ref == nil || !ref.Complete || b.AcknowledgedBytes != int64(len(raw)) || b.Error.Class != "" {
					t.Fatal("normal capture changed", b, e)
				}
			}
			encoded, _ := json.Marshal(b)
			if bytes.Contains(encoded, []byte(directory)) || bytes.Contains(encoded, []byte("file exists")) {
				t.Fatal("raw OS error/path leaked")
			}
			if label != "sealed" {
				before := *b
				if _, e = s.capture(bytes.NewReader(raw), witnessTestID, "native-small", "original-management", "original-helper"); e == nil || !reflect.DeepEqual(before, *s.captureBoundary) {
					t.Fatal("later denial replaced original I/O facts")
				}
			}
		})
	}
}
