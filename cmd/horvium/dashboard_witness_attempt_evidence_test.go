// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

// The full native gate must export every attempt's new forwarding observation.
// A later capacity failure may deny eligibility but must not rewrite those facts.
func TestDashboardWitnessAttemptCapPreservesForwardEvidence(t *testing.T) {
	store, e := newDashboardWitnessStore(witnessPrivateDirectory(t))
	if e != nil {
		t.Fatal(e)
	}
	defer store.close()
	var last *dashboardWitnessRef
	for n := 1; n <= witnessAttemptCap; n++ {
		id := fmt.Sprintf("%s-%d", witnessTestID[:32], n)
		raw := witnessCapacityGolden(id)
		ref, e := store.capture(bytes.NewReader(raw), id, "projection-10000", "original-management", "original-helper")
		if e != nil || !ref.Complete {
			t.Fatal("in-budget original", n, e)
		}
		var out bytes.Buffer
		if store.forward(ref, &out) != nil || !bytes.Equal(out.Bytes(), raw) || !ref.Delivered || ref.Forward == nil || ref.Forward.Bytes != int64(len(raw)) || ref.Forward.SHA256 != ref.RawSHA256 || ref.Forward.Error.Class != "" {
			t.Fatal("in-budget exact forwarding evidence", n)
		}
		last = ref
	}
	before, e := json.Marshal(store.refs)
	// This bounds only the actual portable reference census. Native qualification
	// separately checks the complete original 64 KiB control envelopes and sidecars.
	if e != nil || len(store.refs) != 32 || len(before) > witnessFrameCap {
		t.Fatal("complete reference census missing or oversized", e, len(before))
	}
	id := fmt.Sprintf("%s-33", witnessTestID[:32])
	if _, e := store.capture(bytes.NewReader(witnessCapacityGolden(id)), id, "projection-10000", "original-management", "original-helper"); e == nil || !store.failed {
		t.Fatal("attempt 33 admitted")
	}
	if store.forward(last, &bytes.Buffer{}) == nil {
		t.Fatal("failed candidate resumed forwarding")
	}
	after, e := json.Marshal(store.refs)
	if e != nil || !bytes.Equal(before, after) {
		t.Fatal("capacity denial rewrote historical delivery facts")
	}
	if _, e := store.original(last); e == nil {
		t.Fatal("latched capacity failure accepted cached witness")
	}
}
