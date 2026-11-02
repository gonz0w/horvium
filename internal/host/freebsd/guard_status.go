// SPDX-License-Identifier: BSD-3-Clause
package freebsd

// The accepted base timeout(1) has post-fork fatal paths returning EXIT_FAILURE
// (1) before its zero-child drain loop. A normal status 1 is therefore ambiguous,
// even when the fixed command itself failed normally. Keep admission fenced.
// Other normal codes follow the accepted base reaper's loop; native qualification
// of the pinned utility/kernel remains separate from this source predicate.
func nativeGuardDrainStatus(exited bool, code int) bool {
	return exited && code >= 0 && code <= 255 && code != 1
}
