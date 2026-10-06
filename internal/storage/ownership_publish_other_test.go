// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"testing"
)

func TestOwnershipPublicationUnavailable(t *testing.T) {
	for _, s := range []*StateDirectory{nil, {}, {}} {
		if err := s.PublishOwnership(Policy{}, 0); !errors.Is(err, ErrUnavailable) {
			t.Fatal("non-FreeBSD publication available", err)
		}
	}
}
