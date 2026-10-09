// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"errors"
	"testing"
)

func TestBootOwnerOriginalLifetime(t *testing.T) {
	id := bootIdentity{1, 2, 3}
	calls := 0
	read := func() (bootIdentity, error) { calls++; return id, nil }
	b, err := bindBootOwner(read)
	if err != nil {
		t.Fatal(err)
	}
	if b.check() != nil || calls != 2 {
		t.Fatal("must independently repeat exact raw observation")
	}
	copied := &bootOwner{original: b.original, identity: b.identity, read: b.read}
	if copied.check() == nil || b.check() != nil {
		t.Fatal("copied owner acquired or retired original authority")
	}
	id[15] = 9
	if b.check() == nil {
		t.Fatal("changed boot retained original owner")
	}
	id[15] = 0
	if b.check() == nil {
		t.Fatal("retired identity was restored by matching bytes")
	}
}
func TestBootOwnerUnavailableAndZeroBytes(t *testing.T) {
	if b, err := bindBootOwner(nil); b != nil || err == nil {
		t.Fatal("nil source accepted")
	}
	if b, err := bindBootOwner(func() (bootIdentity, error) { return bootIdentity{}, ErrUnsupported }); b != nil || !errors.Is(err, ErrUnsupported) {
		t.Fatal("unsupported read accepted")
	}
	var b *bootOwner
	if b.check() == nil {
		t.Fatal("nil owner accepted")
	}
	// Raw random bytes, including a trailing zero, have no UUID/string semantics.
	b, err := bindBootOwner(func() (bootIdentity, error) { return bootIdentity{}, nil })
	if err != nil || b.check() != nil {
		t.Fatal("raw zero byte was trimmed or invented")
	}
	b.read = func() (bootIdentity, error) { return bootIdentity{}, ErrUnavailable }
	if b.check() == nil {
		t.Fatal("failed read retained owner")
	}
	b.read = func() (bootIdentity, error) { return bootIdentity{}, nil }
	if b.check() == nil {
		t.Fatal("failed source restored authority")
	}
}
