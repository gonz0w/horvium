// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

// OpenStateDirectory is unavailable outside the qualified runtime target.
func OpenStateDirectory(string) (*StateDirectory, error) { return nil, ErrUnavailable }
