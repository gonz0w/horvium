// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

func (*StateDirectory) propertyEnrollmentState(Policy, Policy, PoolSystemUse, stateCatalogLimits, func()) ([32]byte, error) {
	return [32]byte{}, ErrUnavailable
}
