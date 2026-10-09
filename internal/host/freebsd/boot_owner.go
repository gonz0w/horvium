// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import "sync"

// Observed identity only; no state, session, launch or restoration authority.
type bootIdentity [16]byte

type bootOwner struct {
	mu       sync.Mutex
	original *bootOwner
	identity bootIdentity
	read     func() (bootIdentity, error)
	retired  bool
}

func bindBootOwner(read func() (bootIdentity, error)) (*bootOwner, error) {
	if read == nil {
		return nil, ErrUnavailable
	}
	id, err := read()
	if err != nil {
		return nil, err
	}
	b := &bootOwner{identity: id, read: read}
	b.original = b
	return b, nil
}
func (b *bootOwner) check() error {
	if b == nil || b.original != b {
		return ErrUnavailable
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.retired || b.read == nil {
		return ErrUnavailable
	}
	id, err := b.read()
	if err != nil || id != b.identity {
		b.retired = true
		return ErrUnavailable
	}
	return nil
}
