// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

func (*StateDirectory) InspectPropertyJob(string) (PropertyJob, [32]byte, error) {
	return PropertyJob{}, [32]byte{}, ErrUnavailable
}
