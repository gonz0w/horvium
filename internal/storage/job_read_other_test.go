// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"reflect"
	"testing"
)

func TestJobReadUnsupported(t *testing.T) {
	for _, s := range []*StateDirectory{nil, {}} {
		got, err := s.ReadJob("../../fixture")
		if !errors.Is(err, ErrUnavailable) || !reflect.DeepEqual(got, Job{}) {
			t.Fatal("unsupported runtime returned job metadata", err)
		}
	}
}
