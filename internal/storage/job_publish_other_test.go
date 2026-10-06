// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"testing"
)

func TestJobPublicationUnsupported(t *testing.T) {
	for _, s := range []*StateDirectory{nil, {}} {
		if e := s.PublishJob(Job{}, 0); !errors.Is(e, ErrUnavailable) {
			t.Fatal("unsupported publication", e)
		}
	}
}
