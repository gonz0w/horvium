// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"encoding/json"
	"github.com/gonz0w/horvium/internal/storage"
	"strings"
	"testing"
	"time"
)

func TestPropertyPeerCannotBeReconstructed(t *testing.T) {
	for _, x := range []*propertyPeer{nil, {}, new(propertyPeer)} {
		req, a, err := x.receive()
		if err == nil || req != (propertyRequest{}) || a != (actionIdentity{}) {
			t.Fatal("unissued request binding")
		}
		if r, err := x.review("100", storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "1"}); err == nil || r.ID != "" {
			t.Fatal("unissued client")
		}
		if j, d, err := x.submit("100", strings.Repeat("a", 64), strings.Repeat("b", 64)); err == nil || d || j.ID != "" {
			t.Fatal("unissued submit")
		}
		if x.replyReview(storage.Review{}) == nil || x.replySubmission(storage.Job{}, false) == nil || x.refuse("denied") == nil {
			t.Fatal("unissued writer")
		}
		if x.Close() != nil || x.Close() != nil {
			t.Fatal("close")
		}
	}
	var x propertyPeer
	if json.Unmarshal([]byte(`{}`), &x) != nil {
		t.Fatal("decode")
	}
	if _, _, err := x.receive(); err == nil {
		t.Fatal("decoded peer")
	}
}
func TestPropertyPeerFactoriesCloseOwnedSocket(t *testing.T) {
	if nativeSupported {
		return
	}
	for _, root := range []bool{true, false} {
		client, server := connected(t)
		var x *propertyPeer
		var err error
		if root {
			x, err = acceptPropertyPeer(context.Background(), server, 42, time.Second)
		} else {
			x, err = connectPropertyPeer(context.Background(), server, 42, "property-1", strings.Repeat("a", 64), time.Second)
		}
		if x != nil || err == nil {
			t.Fatal("unsupported host authorized")
		}
		var b [1]byte
		if n, err := client.Read(b[:]); n != 0 || err == nil {
			t.Fatal("owned socket retained")
		}
	}
}
