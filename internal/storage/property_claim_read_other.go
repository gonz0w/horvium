// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

func (*PropertyStateClaim) InspectPropertyJob(string) (PropertyJob, [32]byte, error) {
	return PropertyJob{}, [32]byte{}, ErrUnavailable
}
func (*PropertyStateClaim) ReadOwnership(string) (Policy, error) { return Policy{}, ErrUnavailable }
