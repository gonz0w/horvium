// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

func TestOwnershipPublicationOrderAndFailures(t *testing.T) {
	phases := []string{"write", "file_sync", "close", "verify", "rename", "directory_sync"}
	for _, fail := range append([]string{""}, phases...) {
		var calls []string
		step := func(name string) func() error {
			return func() error {
				calls = append(calls, name)
				if name == fail {
					return errors.New("fixture failure")
				}
				return nil
			}
		}
		tx := ownershipPublication{write: step("write"), syncFile: step("file_sync"), closeFile: step("close"), verify: step("verify"), rename: step("rename"), syncDirectory: step("directory_sync"), cleanup: step("cleanup")}
		err := commitOwnershipPublication(tx)
		if fail == "" {
			if err != nil || !reflect.DeepEqual(calls, phases) {
				t.Fatal("success acknowledged before ordered durability", calls, err)
			}
			continue
		}
		var want []string
		for _, name := range phases {
			want = append(want, name)
			if name == fail {
				break
			}
		}
		if fail == "directory_sync" {
			if !errors.Is(err, ErrStateUncertain) {
				t.Fatal("published uncertain result hidden", err)
			}
		} else {
			want = append(want, "cleanup")
			if !errors.Is(err, ErrStateDirectory) {
				t.Fatal("prepublication failure acknowledged", err)
			}
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatal("failure continued publication or rolled back published name", calls, want)
		}
	}
	// A failed prepublication cleanup also has an unresolved metadata effect.
	tx := ownershipPublication{write: func() error { return errors.New("write failure") }, cleanup: func() error { return errors.New("cleanup failure") }}
	if !errors.Is(commitOwnershipPublication(tx), ErrStateUncertain) {
		t.Fatal("cleanup failure lost")
	}
	// Final catalog failures keep their safety category after owned-temp cleanup.
	for _, cause := range []error{ErrDrift, ErrLimit, ErrStateRecord, ErrStateUncertain} {
		cleaned := false
		tx := ownershipPublication{write: func() error { return cause }, cleanup: func() error { cleaned = true; return nil }}
		if !errors.Is(commitOwnershipPublication(tx), cause) || !cleaned {
			t.Fatal("catalog denial hidden or owned cleanup skipped", cause)
		}
	}
}

func TestOwnershipPublicationRevision(t *testing.T) {
	for _, tc := range []struct {
		next, expected uint64
		ok             bool
	}{{1, 0, true}, {2, 1, true}, {math.MaxUint64, math.MaxUint64 - 1, true}, {0, 0, false}, {2, 0, false}, {1, 1, false}, {3, 1, false}, {0, math.MaxUint64, false}, {math.MaxUint64, math.MaxUint64, false}} {
		if validOwnershipPublicationRevision(tc.next, tc.expected) != tc.ok {
			t.Fatal("invalid publication revision contract", tc)
		}
	}
}
