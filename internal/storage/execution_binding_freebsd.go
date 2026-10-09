// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"os"

	"golang.org/x/sys/unix"
)

// Actual protected object observation only. The live mount view is retained for
// in-call drift checks, never encoded as a persistent filesystem identity.
type executionBindingObjects struct {
	directory, lock executionObjectIdentity
	mount           propertyStateBacking
}

func (s *StateDirectory) inspectExecutionBindingObjects(afterRead func()) (executionBindingObjects, error) {
	return executionBindingObjects{}, ErrUnavailable
}

func executionStatIdentity(st unix.Stat_t) executionObjectIdentity {
	return executionObjectIdentity{st.Ino, st.Gen, st.Btim.Sec, uint32(st.Btim.Nsec)}
}

// Kept here so the native collector cannot accidentally replace the kernel UID
// with a claimed caller identity when its guarded implementation is assembled.
func executionBindingRoot() bool { return os.Geteuid() == 0 }
