// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

// ReadOwnership performs no file operation outside the FreeBSD runtime target.
func (*StateDirectory) ReadOwnership(string) (Policy, error) { return Policy{}, ErrUnavailable }
