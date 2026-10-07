// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"reflect"
	"testing"
)

func TestPropertySubmissionUnsupported(t *testing.T) {
	for _, s := range []*StateDirectory{nil, {}} {
		got, duplicate, err := s.PublishPropertySubmission(PropertyJob{})
		if !errors.Is(err, ErrUnavailable) || duplicate || !reflect.DeepEqual(got, PropertyJob{}) {
			t.Fatal("unsupported submission publication", err)
		}
	}
}
