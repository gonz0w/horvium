// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"reflect"
	"testing"
)

func TestStateCatalogUnsupported(t *testing.T) {
	for _, s := range []*StateDirectory{nil, {}} {
		got, err := s.InspectState()
		if !errors.Is(err, ErrUnavailable) || !reflect.DeepEqual(got, StateCatalog{}) {
			t.Fatal("unsupported state census", err)
		}
	}
}
