// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"testing"
)

// Compiler provenance contains real multiline text; the control grammar rejects
// it deliberately. Source bootstrap must carry only the independently observed
// ABI while the full compiler receipt remains separate original evidence.
func TestDashboardWitnessControlBootstrapExcludesCompilerProse(t *testing.T) {
	type layout struct {
		CredSize int `json:"cred_size"`
		PIDSize  int `json:"pid_size"`
	}
	original := struct {
		ABI      layout `json:"abi"`
		Compiler string `json:"compiler_version"`
	}{layout{80, 4}, "FreeBSD clang version\nTarget: x86_64-unknown-freebsd"}
	var full bytes.Buffer
	if witnessControlWrite(&full, original) != nil {
		t.Fatal("original framed evidence")
	}
	var copied struct {
		ABI      layout `json:"abi"`
		Compiler string `json:"compiler_version"`
	}
	if witnessControlRead(&full, &copied) == nil {
		t.Fatal("multiline provenance became control input")
	}
	var bootstrap bytes.Buffer
	if witnessControlWrite(&bootstrap, original.ABI) != nil {
		t.Fatal("ABI-only bootstrap")
	}
	var actual layout
	if witnessControlRead(&bootstrap, &actual) != nil || witnessEOF(&bootstrap) != nil || actual != original.ABI {
		t.Fatal("original ABI lost in separate bootstrap")
	}
}
