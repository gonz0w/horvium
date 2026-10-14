// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestDashboardWitnessKernelRecordUsesObservedLayout(t *testing.T) {
	layout := witnessKernelLayout{Size: 64, PID: 0, Parent: 4, UID: 8, Sec: 16, Usec: 24, RSS: 32, Runtime: 40, PageBytes: 4096}
	b := make([]byte, layout.Size)
	put32 := func(offset int, v uint32) { binary.LittleEndian.PutUint32(b[offset:], v) }
	put64 := func(offset int, v uint64) { binary.LittleEndian.PutUint64(b[offset:], v) }
	put32(layout.PID, 47)
	put32(layout.Parent, 42)
	put32(layout.UID, 1001)
	put64(layout.Sec, 12345)
	put64(layout.Usec, 123)
	put64(layout.RSS, 17)
	put64(layout.Runtime, 2345678)
	got, e := witnessDecodeKernelProcess(b, layout)
	if e != nil || got.PID != 47 || got.Parent != 42 || got.UID != 1001 || got.Sec != 12345 || got.Usec != 123 || got.RSS != 17*4096 || got.CPU != 2345678000 {
		t.Fatal("actual kernel units/identity lost", got, e)
	}
	for _, tc := range []struct {
		name   string
		change func([]byte, *witnessKernelLayout)
	}{
		{"overlap", func(_ []byte, l *witnessKernelLayout) { l.RSS = l.Runtime }},
		{"offset outside", func(_ []byte, l *witnessKernelLayout) { l.RSS = l.Size - 7 }},
		{"unknown page size", func(_ []byte, l *witnessKernelLayout) { l.PageBytes = 0 }},
		{"non page size", func(_ []byte, l *witnessKernelLayout) { l.PageBytes = 3000 }},
		{"negative PID", func(b []byte, l *witnessKernelLayout) { binary.LittleEndian.PutUint32(b[l.PID:], math.MaxUint32) }},
		{"negative start", func(b []byte, l *witnessKernelLayout) { binary.LittleEndian.PutUint64(b[l.Sec:], math.MaxUint64) }},
		{"invalid start fraction", func(b []byte, l *witnessKernelLayout) { binary.LittleEndian.PutUint64(b[l.Usec:], 1000000) }},
		{"negative pages", func(b []byte, l *witnessKernelLayout) { binary.LittleEndian.PutUint64(b[l.RSS:], math.MaxUint64) }},
		{"RSS overflow", func(b []byte, l *witnessKernelLayout) { binary.LittleEndian.PutUint64(b[l.RSS:], math.MaxInt64) }},
		{"CPU overflow", func(b []byte, l *witnessKernelLayout) { binary.LittleEndian.PutUint64(b[l.Runtime:], math.MaxUint64) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copyBytes := append([]byte(nil), b...)
			copyLayout := layout
			tc.change(copyBytes, &copyLayout)
			if _, e := witnessDecodeKernelProcess(copyBytes, copyLayout); e == nil {
				t.Fatal("malformed kernel record admitted")
			}
		})
	}
	for _, data := range [][]byte{nil, b[:len(b)-1], append(append([]byte(nil), b...), 0)} {
		if _, e := witnessDecodeKernelProcess(data, layout); e == nil {
			t.Fatal("wrong object size admitted")
		}
	}
}
