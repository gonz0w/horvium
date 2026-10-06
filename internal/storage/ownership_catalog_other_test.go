// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"reflect"
	"testing"
)

func TestOwnershipCatalogUnavailable(t *testing.T) {
	for _, s := range []*StateDirectory{nil, {}} {
		got, err := s.InspectOwnership()
		if !errors.Is(err, ErrUnavailable) || !reflect.DeepEqual(got, OwnershipCatalog{}) {
			t.Fatal("off-target inspection performed work", got, err)
		}
	}
}
