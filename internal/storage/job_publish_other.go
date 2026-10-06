// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

func (*StateDirectory) PublishJob(Job, uint64) error { return ErrUnavailable }
