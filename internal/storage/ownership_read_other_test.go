// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"reflect"
	"testing"
)

func TestOwnershipReadUnavailableOffFreeBSD(t *testing.T) {
	for _, s := range []*StateDirectory{nil, {}} {
		if p, err := s.ReadOwnership("100"); !errors.Is(err, ErrUnavailable) || !reflect.DeepEqual(p, Policy{}) {
			t.Fatal("unsupported reader returned policy", err)
		}
	}
}
