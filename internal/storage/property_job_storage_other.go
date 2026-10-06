// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

func (*StateDirectory) ReadPropertyJob(string) (PropertyJob, error) {
	return PropertyJob{}, ErrUnavailable
}
func (*StateDirectory) PublishPropertyJob(PropertyJob, uint64) error { return ErrUnavailable }
