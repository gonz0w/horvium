//go:build horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "github.com/gonz0w/horvium/internal/storage"

var policyProvenanceChildControls = []string{"cpu_affinity", "child_budget", "caller_deadline", "original_call", "sampler_joined", "samples_bounded", "owner_release", "deadline_unchanged", "trace_enabled", "error_preserved", "snapshots_before_cancel", "trace_inventory", "original_contexts"}

func policyProvenanceChild([]byte) ([]byte, []byte, bool) { return nil, nil, false }
func policyProvenanceTrace([]byte) (storage.DeadlineProvenance, bool) {
	return storage.DeadlineProvenance{}, false
}
func policyProvenanceOriginalContexts(storage.DeadlineProvenance, int64) bool { return false }
