// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

// No file operation occurs outside the FreeBSD runtime target.
func (*StateDirectory) ReadJob(string) (Job, error) { return Job{}, ErrUnavailable }
