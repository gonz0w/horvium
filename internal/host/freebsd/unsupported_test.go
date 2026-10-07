//go:build !freebsd

// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"errors"
	"github.com/gonz0w/horvium/internal/inventory"
	"testing"
)

func TestUnsupportedHostFailsExplicitly(t *testing.T) {
	if _, err := New(inventory.DefaultLimits()); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("host discovery pretended to support this OS: %v", err)
	}
	if _, err := NewPoolPropertySource(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("pool property collection pretended to support this OS: %v", err)
	}
}
