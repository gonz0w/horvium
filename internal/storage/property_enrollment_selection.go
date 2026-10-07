// SPDX-License-Identifier: BSD-3-Clause
package storage

// ValidatePropertyEnrollmentSelection checks an explicit root proposal and its
// exact predecessor before native collection. It authenticates no actor and
// establishes neither native correspondence nor stored-state authority.
func ValidatePropertyEnrollmentSelection(rootContext string, next, previous Policy) error {
	if !token(rootContext) {
		return ErrInvalid
	}
	canonical, err := normalizeOwnership(next)
	if err != nil {
		return err
	}
	if len(canonical.Scopes) == 0 || len(canonical.Members) == 0 {
		return ErrDenied
	}
	_, err = enrollmentPredecessor(canonical, previous)
	return err
}
