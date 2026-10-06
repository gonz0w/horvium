// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

func (*StateDirectory) InspectState() (StateCatalog, error) {
	return StateCatalog{}, ErrUnavailable
}
