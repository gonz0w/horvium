// SPDX-License-Identifier: BSD-3-Clause
package helper

func credentialUID(version, uid uint32) (uint32, error) {
	// XUCRED_VERSION in the pinned FreeBSD ABI is 0. Never interpret UID
	// from an unknown credential layout, even if its value would be allowed.
	if version != 0 {
		return 0, ErrPermission
	}
	return uid, nil
}
