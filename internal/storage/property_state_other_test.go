// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"testing"
)

func TestPropertyStoreUnsupported(t *testing.T) {
	for _, s := range []*StateDirectory{nil, {}} {
		if err := s.CheckPropertyState(Policy{}, PoolSystemUse{}); !errors.Is(err, ErrUnavailable) {
			t.Fatal(err)
		}
	}
}
