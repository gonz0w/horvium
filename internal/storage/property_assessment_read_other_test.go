// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestAssessmentReaderUnsupported(t *testing.T) {
	for _, s := range []*StateDirectory{nil, {}} {
		p, digest, err := s.InspectPropertyJob(strings.Repeat("a", 64))
		if !errors.Is(err, ErrUnavailable) || !reflect.DeepEqual(p, PropertyJob{}) || digest != ([32]byte{}) {
			t.Fatal("unsupported platform exposed native state", err)
		}
	}
}
