// SPDX-License-Identifier: BSD-3-Clause
package storage

import "testing"

func TestExecutionBackingObservation(t *testing.T) {
	for _, b := range []ExecutionBackingObservation{{Kind: "ufs", UFS: [2]uint32{1, 2}}, {Kind: "zfs", Pool: 3, Dataset: 4}} {
		got, e := b.identity()
		if e != nil || !got.valid() || got.kind != b.Kind || got.ufs != b.UFS || got.pool != b.Pool || got.dataset != b.Dataset {
			t.Fatalf("complete backing lost: %+v %v", got, e)
		}
	}
	for _, b := range []ExecutionBackingObservation{{}, {Kind: "other", UFS: [2]uint32{1, 2}}, {Kind: "ufs", UFS: [2]uint32{0, 2}}, {Kind: "ufs", UFS: [2]uint32{1, 0}}, {Kind: "ufs", UFS: [2]uint32{1, 2}, Pool: 3}, {Kind: "ufs", UFS: [2]uint32{1, 2}, Dataset: 4}, {Kind: "zfs", Dataset: 4}, {Kind: "zfs", Pool: 3}, {Kind: "zfs", Pool: 3, Dataset: 4, UFS: [2]uint32{1, 2}}} {
		got, e := b.identity()
		if e == nil || got != (executionBackingIdentity{}) {
			t.Fatalf("invalid backing admitted: %+v %v", got, e)
		}
	}
}
