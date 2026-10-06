// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

// PublishOwnership performs no filesystem operation outside FreeBSD.
func (*StateDirectory) PublishOwnership(Policy, uint64) error { return ErrUnavailable }
