// SPDX-License-Identifier: BSD-3-Clause
package storage

// ValidatePropertyJobSelection checks only the canonical opaque job ID before
// metadata/native I/O. A valid spelling supplies no authorization or provenance.
func ValidatePropertyJobSelection(id string) error {
	if !validJobDigest(id) {
		return ErrInvalid
	}
	return nil
}
