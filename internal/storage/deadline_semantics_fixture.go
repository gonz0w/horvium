//go:build horvium_native_fixture

// SPDX-License-Identifier: BSD-3-Clause
package storage

// DeadlineProvenanceErrorPreserved checks only a disposable diagnostic error.
func DeadlineProvenanceErrorPreserved(error) bool { return false }
