// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"errors"
	"testing"
)

func TestPropertyClaimApplyingAcknowledgementUnavailable(t *testing.T) {
	for _, c := range []*PropertyStateClaim{nil, {}} {
		if err := c.VerifyPropertyApplyingAcknowledgement(PropertyJob{}); !errors.Is(err, ErrUnavailable) {
			t.Fatal(err)
		}
	}
}
