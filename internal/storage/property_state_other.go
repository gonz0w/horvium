// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

func (*StateDirectory) CheckPropertyState(Policy, PoolSystemUse) error { return ErrUnavailable }
