// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

func (*StateDirectory) ClaimPropertyState(string) (*PropertyStateClaim, error) {
	return nil, ErrUnavailable
}
func (*PropertyStateClaim) CheckPropertyState(Policy, PoolSystemUse, PropertyJob) error {
	return ErrUnavailable
}
