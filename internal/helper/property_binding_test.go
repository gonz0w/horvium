// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

// Only a destructor fixture: these private fields prove no kernel/native liveness.
func bindingRetirementFixture(t *testing.T) (*propertyRequestBinding, *propertyConnection, *net.UnixConn) {
	t.Helper()
	p := connectionAdmission(t, context.Background(), 1, 3*time.Second)
	client, c := admitConnection(t, p)
	a := &admittedPropertyPeer{lease: c, exchange: &propertyPeer{peer: &actionPeer{}}}
	a.identity = a
	owner := &propertyRequestOwner{peer: a}
	a.bindings = owner
	r := &propertyRequestBinding{owner: owner, peer: a}
	r.identity = r
	owner.binding = r
	return r, c, client
}
func TestPropertyBindingUnissued(t *testing.T) {
	for _, name := range []string{"nil", "zero", "decoded", "copy", "reconstructed", "foreign-owner"} {
		t.Run(name, func(t *testing.T) {
			original, lease, client := bindingRetirementFixture(t)
			var r *propertyRequestBinding
			switch name {
			case "zero":
				r = &propertyRequestBinding{}
			case "decoded":
				r = &propertyRequestBinding{}
				if json.Unmarshal([]byte(`{}`), r) != nil {
					t.Fatal("fixture")
				}
			case "copy":
				copy := *original
				r = &copy
			case "reconstructed":
				r = &propertyRequestBinding{owner: original.owner, peer: original.peer}
				r.identity = r
			case "foreign-owner":
				r = &propertyRequestBinding{owner: &propertyRequestOwner{peer: original.peer}, peer: original.peer}
				r.identity = r
			}
			if r.check(context.Background()) == nil {
				t.Fatal("unissued binding passed")
			}
			if req, actor, err := r.snapshot(context.Background()); req != (propertyRequest{}) || actor != (actionIdentity{}) || err == nil {
				t.Fatal("unissued binding projected metadata")
			}
			if r.Close() != nil {
				t.Fatal("unissued close")
			}
			if _, _, err := lease.original(); err != nil {
				t.Fatal("unissued receipt retired original")
			}
			_ = lease.Close()
			connectionEOF(t, client)
		})
	}
	for _, a := range []*admittedPropertyPeer{nil, {}} {
		if r, err := receivePropertyBinding(a); r != nil || err == nil {
			t.Fatal("unissued peer created request binding")
		}
	}
}
func TestPropertyBindingOriginalRetirement(t *testing.T) {
	r, c, client := bindingRetirementFixture(t)
	if r.Close() != nil || r.Close() != nil {
		t.Fatal("original close")
	}
	connectionDone(t, c)
	connectionEOF(t, client)
	if r.check(context.Background()) == nil {
		t.Fatal("closed receipt lived")
	}
}
