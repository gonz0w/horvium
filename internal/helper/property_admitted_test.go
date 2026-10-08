// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

func TestAdmittedPropertyPeerUnissued(t *testing.T) {
	p := connectionAdmission(t, context.Background(), 1, time.Second)
	client, c := admitConnection(t, p)
	var decoded admittedPropertyPeer
	if json.Unmarshal([]byte(`{}`), &decoded) != nil {
		t.Fatal("fixture")
	}
	for _, a := range []*admittedPropertyPeer{nil, {}, &decoded, {lease: c, exchange: &propertyPeer{}}} {
		if req, actor, err := a.receive(); req != (propertyRequest{}) || actor != (actionIdentity{}) || err == nil {
			t.Fatal("unissued admission projected request")
		}
		if a.replyReview(storage.Review{}) == nil || a.replySubmission(storage.Job{}, false) == nil || a.refuse("denied") == nil {
			t.Fatal("unissued writer")
		}
		if a.Close() != nil {
			t.Fatal("unissued close")
		}
	}
	if _, _, err := c.original(); err != nil {
		t.Fatal("unissued wrapper retired unrelated original", err)
	}
	_ = c.Close()
	connectionEOF(t, client)
}
func TestAdmittedPropertyFactoryInvalidUIDReleasesSlot(t *testing.T) {
	for _, uid := range []uint32{0, ^uint32(0)} {
		p := connectionAdmission(t, context.Background(), 1, time.Second)
		client, c := admitConnection(t, p)
		if a, err := c.accept(uid); a != nil || !errors.Is(err, ErrConfig) {
			t.Fatal("invalid UID bypassed original factory", err)
		}
		connectionDone(t, c)
		connectionEOF(t, client)
		next, lease := admitConnection(t, p)
		_ = lease.Close()
		connectionEOF(t, next)
	}
}
func TestAdmittedPropertyFactoryRetiredOrCopiedLease(t *testing.T) {
	p := connectionAdmission(t, context.Background(), 1, time.Second)
	client, c := admitConnection(t, p)
	copy := *c
	if a, err := copy.accept(42); a != nil || err == nil {
		t.Fatal("copied lease accepted")
	}
	if _, _, err := c.original(); err != nil {
		t.Fatal("copied attempt retired original")
	}
	_ = c.Close()
	connectionEOF(t, client)
	if a, err := c.accept(42); a != nil || err == nil {
		t.Fatal("retired lease accepted")
	}
	var decoded propertyConnection
	if json.Unmarshal([]byte(`{}`), &decoded) != nil {
		t.Fatal("fixture")
	}
	for _, c := range []*propertyConnection{nil, {}, &decoded} {
		if a, err := c.accept(42); a != nil || err == nil {
			t.Fatal("unissued lease accepted")
		}
	}
}
func TestAdmittedPropertyUnsupportedHostReleasesSlot(t *testing.T) {
	if nativeSupported {
		return
	}
	p := connectionAdmission(t, context.Background(), 1, time.Second)
	client, c := admitConnection(t, p)
	if a, err := c.accept(42); a != nil || !errors.Is(err, ErrUnsupported) {
		t.Fatal("unsupported admission authorized", err)
	}
	connectionDone(t, c)
	connectionEOF(t, client)
	next, lease := admitConnection(t, p)
	_ = lease.Close()
	connectionEOF(t, next)
}
