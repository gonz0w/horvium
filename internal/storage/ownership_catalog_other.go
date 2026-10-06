// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

func (*StateDirectory) InspectOwnership() (OwnershipCatalog, error) {
	return OwnershipCatalog{}, ErrUnavailable
}
