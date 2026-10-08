// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"reflect"
	"testing"
)

func TestPropertyClaimApplyingUnsupported(t *testing.T) {
	for _, c := range []*PropertyStateClaim{nil, {}} {
		got, err := c.PublishApplyingPropertyJob(propertyJobFixture(t))
		if !errors.Is(err, ErrUnavailable) || !reflect.DeepEqual(got, PropertyJob{}) {
			t.Fatal("unsupported publication", err)
		}
	}
}
