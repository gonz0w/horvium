// SPDX-License-Identifier: BSD-3-Clause
package service

import (
	"net"
	"testing"
)

func newListener(t *testing.T) net.Listener {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l
}
