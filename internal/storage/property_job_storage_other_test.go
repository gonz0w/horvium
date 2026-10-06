// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"reflect"
	"testing"
)

func TestPropertyJobStorageUnavailable(t *testing.T) {
	for _, s := range []*StateDirectory{nil, {}} {
		got, e := s.ReadPropertyJob("../writer.lock")
		if !errors.Is(e, ErrUnavailable) || !reflect.DeepEqual(got, PropertyJob{}) {
			t.Fatal("non-FreeBSD read", e)
		}
		if e := s.PublishPropertyJob(PropertyJob{}, 0); !errors.Is(e, ErrUnavailable) {
			t.Fatal("non-FreeBSD publication", e)
		}
	}
}
