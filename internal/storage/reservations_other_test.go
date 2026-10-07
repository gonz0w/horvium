// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"reflect"
	"testing"
)

func TestReservationInspectionUnsupported(t *testing.T) {
	for _, s := range []*StateDirectory{nil, {}} {
		got, err := s.InspectJobReservations()
		if !errors.Is(err, ErrUnavailable) || !reflect.DeepEqual(got, JobReservations{}) {
			t.Fatal("unsupported reservation inspection", err)
		}
	}
}
