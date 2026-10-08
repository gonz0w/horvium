// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"reflect"
	"testing"
)

func TestPropertyClaimSuccessorUnsupported(t *testing.T) {
	for _, c := range []*PropertyStateClaim{nil, {}} {
		got, err := c.PublishPropertyJobSuccessor(propertyJobFixture(t))
		if !errors.Is(err, ErrUnavailable) || !reflect.DeepEqual(got, PropertyJob{}) {
			t.Fatal("unsupported successor publication", err)
		}
	}
}
