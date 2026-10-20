// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

type controlAncestor struct {
	file    *os.File
	name    string
	stat    unix.Stat_t
	backing propertyStateBacking
}
type controlSelectionNative struct {
	file    *os.File
	name    string
	stat    unix.Stat_t
	digest  [32]byte
	backing propertyStateBacking
	// Original configuration, journal-parent, ledger-parent chains, respectively.
	chains        [3][]controlAncestor
	domains       [2]unix.Stat_t
	present       [2]bool
	domainBacking [2]propertyStateBacking
	bound         bool
}

// OpenControlSelection performs only read-only descriptor custody. Target paths
// supply metadata, never authority; original execution producers still own every
// lease, complete census, native backing fact and publication acknowledgement.
// A failed admission may return a retired nonnil original. Its caller must Close
// it once and confirm release before invoking any output sink.
func OpenControlSelection(ctx context.Context, path string) (*ControlSelection, error) {
	if !ValidControlPath(path) {
		return nil, ErrInvalid
	}
	if e := controlRootContext(ctx); e != nil {
		return nil, e
	}
	s := &ControlSelection{native: &controlSelectionNative{}}
	s.original = s
	n := s.native
	// Return the retired original on failed admission so its one caller can always
	// attempt release and distinguish confirmed cleanup before choosing output.
	fail := func(e error) (*ControlSelection, error) { s.blocked = true; return s, e }
	var e error
	n.chains[0], e = openControlAncestors(ctx, filepath.Dir(path))
	if e != nil {
		return fail(e)
	}
	parent := int(n.chains[0][len(n.chains[0])-1].file.Fd())
	n.name = filepath.Base(path)
	var named unix.Stat_t
	if unix.Fstatat(parent, n.name, &named, unix.AT_SYMLINK_NOFOLLOW) != nil || !privateOwnershipFile(named) || named.Size < 1 || named.Size > maxControlConfigBytes {
		return fail(ErrStateDirectory)
	}
	fd, e := unix.Openat(parent, n.name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if e != nil {
		return fail(ErrStateDirectory)
	}
	n.file = os.NewFile(uintptr(fd), "storage-control-selection")
	if unix.Fstat(fd, &n.stat) != nil || !sameOwnershipMetadata(named, n.stat) {
		return fail(ErrDrift)
	}
	n.backing, e = propertyStateFilesystem(fd)
	if e != nil {
		return fail(e)
	}
	if e = controlSafeFilesystem(n.backing); e != nil {
		return fail(e)
	}
	data, e := n.read(ctx)
	if e != nil {
		return fail(e)
	}
	n.digest = sha256.Sum256(data)
	s.targets, e = ParseControlConfig(bytes.NewReader(data))
	if e != nil {
		return fail(e)
	}
	for _, domain := range []string{s.targets.JournalDirectory, s.targets.LedgerDirectory} {
		if path == domain || strings.HasPrefix(path, domain+"/") {
			return fail(ErrDenied)
		}
	}
	for i, domain := range []string{s.targets.JournalDirectory, s.targets.LedgerDirectory} {
		n.chains[i+1], e = openControlAncestors(ctx, filepath.Dir(domain))
		if e != nil {
			return fail(e)
		}
		var st unix.Stat_t
		parent := int(n.chains[i+1][len(n.chains[i+1])-1].file.Fd())
		e = unix.Fstatat(parent, filepath.Base(domain), &st, unix.AT_SYMLINK_NOFOLLOW)
		if e == nil {
			n.present[i] = true
			n.domains[i] = st
		} else if e != unix.ENOENT {
			return fail(ErrStateDirectory)
		}
	}
	if e = s.checkLocked(ctx, false); e != nil {
		return fail(e)
	}
	return s, nil
}

func controlRootContext(ctx context.Context) error {
	if ctx == nil {
		return ErrDenied
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	if os.Getuid() != 0 || os.Geteuid() != 0 {
		return ErrDenied
	}
	return nil
}
func controlSafeFilesystem(b propertyStateBacking) error {
	if !validControlFilesystem(b) {
		return ErrUnavailable
	}
	return nil
}

// Directory content timestamps/link counts legitimately change when the fixed
// fresh producer creates siblings. Custody pins identity, ownership, mode,
// flags, birth/generation, ACL and mount; it never treats timestamps as a fence.
func sameControlDirectory(a, b unix.Stat_t) bool {
	return a.Dev == b.Dev && a.Ino == b.Ino && a.Mode == b.Mode && a.Uid == b.Uid && a.Gid == b.Gid && a.Flags == b.Flags && a.Gen == b.Gen && a.Btim == b.Btim
}
func openControlAncestors(ctx context.Context, path string) ([]controlAncestor, error) {
	var chain []controlAncestor
	fail := func(e error) ([]controlAncestor, error) { return chain, e }
	flags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC
	parts := []string{"/"}
	if path != "/" {
		parts = append(parts, strings.Split(strings.TrimPrefix(path, "/"), "/")...)
	}
	for i, name := range parts {
		if e := controlRootContext(ctx); e != nil {
			return fail(e)
		}
		var fd int
		var e error
		if i == 0 {
			fd, e = unix.Open("/", flags, 0)
		} else {
			fd, e = unix.Openat(int(chain[i-1].file.Fd()), name, flags, 0)
		}
		if e != nil {
			return fail(ErrStateDirectory)
		}
		a := controlAncestor{file: os.NewFile(uintptr(fd), "storage-control-ancestor"), name: name}
		chain = append(chain, a)
		current := &chain[len(chain)-1]
		if unix.Fstat(fd, &current.stat) != nil || current.stat.Mode&unix.S_IFMT != unix.S_IFDIR || current.stat.Uid != 0 || current.stat.Mode&07022 != 0 || !safeStateACL(fd, uint32(current.stat.Mode), true) {
			return fail(ErrStateDirectory)
		}
		current.backing, e = propertyStateFilesystem(fd)
		if e != nil {
			return fail(e)
		}
	}
	if e := controlSafeFilesystem(chain[len(chain)-1].backing); e != nil {
		return fail(e)
	}
	return chain, nil
}
func closeControlChain(chain []controlAncestor) error {
	var e error
	for i := len(chain) - 1; i >= 0; i-- {
		e = errors.Join(e, chain[i].file.Close())
	}
	return e
}
func checkControlChain(ctx context.Context, chain []controlAncestor) error {
	if len(chain) == 0 {
		return ErrDenied
	}
	for i, a := range chain {
		if e := controlRootContext(ctx); e != nil {
			return e
		}
		var got, named unix.Stat_t
		fd := int(a.file.Fd())
		if unix.Fstat(fd, &got) != nil || !sameControlDirectory(got, a.stat) || !safeStateACL(fd, uint32(got.Mode), true) {
			return ErrDrift
		}
		if i == 0 {
			if unix.Lstat("/", &named) != nil {
				return ErrDrift
			}
		} else if unix.Fstatat(int(chain[i-1].file.Fd()), a.name, &named, unix.AT_SYMLINK_NOFOLLOW) != nil {
			return ErrDrift
		}
		if !sameControlDirectory(got, named) {
			return ErrDrift
		}
		b, e := propertyStateFilesystem(fd)
		if e != nil {
			return e
		}
		if b != a.backing {
			return ErrDrift
		}
	}
	return nil
}
func (n *controlSelectionNative) read(ctx context.Context) ([]byte, error) {
	if e := controlRootContext(ctx); e != nil {
		return nil, e
	}
	fd := int(n.file.Fd())
	parent := int(n.chains[0][len(n.chains[0])-1].file.Fd())
	var before, after, named unix.Stat_t
	if unix.Fstat(fd, &before) != nil || !privateOwnershipFile(before) || !sameOwnershipMetadata(before, n.stat) || !safeStateACL(fd, uint32(before.Mode), false) {
		return nil, ErrDrift
	}
	data, e := io.ReadAll(io.NewSectionReader(n.file, 0, maxControlConfigBytes+1))
	if e != nil || int64(len(data)) != before.Size || len(data) > maxControlConfigBytes {
		return nil, ErrDrift
	}
	if unix.Fstat(fd, &after) != nil || !sameOwnershipMetadata(before, after) || unix.Fstatat(parent, n.name, &named, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(after, named) {
		return nil, ErrDrift
	}
	b, e := propertyStateFilesystem(fd)
	if e != nil {
		return nil, e
	}
	if b != n.backing {
		return nil, ErrDrift
	}
	if e = controlRootContext(ctx); e != nil {
		return nil, e
	}
	return data, nil
}
func (s *ControlSelection) checkLocked(ctx context.Context, fresh bool) error {
	n := s.native
	if n == nil || n.file == nil || s.closed || s.blocked {
		return ErrDenied
	}
	for _, chain := range n.chains {
		if e := checkControlChain(ctx, chain); e != nil {
			return e
		}
	}
	data, e := n.read(ctx)
	if e != nil {
		return e
	}
	if sha256.Sum256(data) != n.digest {
		return ErrDrift
	}
	for i, path := range []string{s.targets.JournalDirectory, s.targets.LedgerDirectory} {
		chain := n.chains[i+1]
		parent := int(chain[len(chain)-1].file.Fd())
		b := chain[len(chain)-1].backing
		var st unix.Stat_t
		e = unix.Fstatat(parent, filepath.Base(path), &st, unix.AT_SYMLINK_NOFOLLOW)
		if e == unix.ENOENT {
			if n.present[i] {
				return ErrDrift
			}
		} else if e != nil {
			return ErrStateDirectory
		} else {
			if fresh {
				return ErrDenied
			}
			if n.present[i] && !sameControlDirectory(st, n.domains[i]) {
				return ErrDrift
			}
			fd, e := unix.Openat(parent, filepath.Base(path), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
			if e != nil {
				return ErrStateDirectory
			}
			var opened unix.Stat_t
			bad := unix.Fstat(fd, &opened) != nil || !sameControlDirectory(st, opened) || opened.Uid != 0 || opened.Mode&07777 != 0700 || !safeStateACL(fd, uint32(opened.Mode), true)
			b, e = propertyStateFilesystem(fd)
			var current unix.Stat_t
			bad = bad || unix.Fstatat(parent, filepath.Base(path), &current, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameControlDirectory(opened, current)
			closeErr := unix.Close(fd)
			if closeErr != nil {
				return errors.Join(ErrControlRelease, closeErr)
			}
			if bad || e != nil {
				return errors.Join(ErrDrift, e)
			}
			for _, a := range n.chains[0] {
				if a.stat.Dev == opened.Dev && a.stat.Ino == opened.Ino {
					return ErrDenied
				}
			}
			// An initially absent leaf may be created only on its original parent FS.
			// Original publication owners still check every new object and mount binding.
			if !n.present[i] && b != chain[len(chain)-1].backing {
				return ErrDrift
			}
		}
		if e = controlSafeFilesystem(b); e != nil {
			return e
		}
		if n.bound && b != n.domainBacking[i] {
			return ErrDrift
		}
		if !n.bound {
			n.domainBacking[i] = b
		}
		if i == 1 {
			if e = checkControlRestoreScope(n.backing, b); e != nil {
				return e
			}
		}
	}
	for _, chain := range n.chains {
		if e = checkControlChain(ctx, chain); e != nil {
			return e
		}
	}
	data, e = n.read(ctx)
	if e != nil {
		return e
	}
	if sha256.Sum256(data) != n.digest {
		return ErrDrift
	}
	if e = controlRootContext(ctx); e != nil {
		return e
	}
	n.bound = true
	return nil
}
func (s *ControlSelection) check(ctx context.Context, fresh bool) error {
	if s == nil || s.original != s {
		return ErrDenied
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.checkLocked(ctx, fresh)
	if e != nil {
		s.blocked = true
	}
	return e
}
func (s *ControlSelection) Check(ctx context.Context) error { return s.check(ctx, false) }

// CheckFresh is an additional read-only absence guard. The exclusive original
// fresh producer repeats both-name checks before its first namespace effect.
func (s *ControlSelection) CheckFresh(ctx context.Context) error { return s.check(ctx, true) }
func (s *ControlSelection) Close() error {
	if s == nil || s.original != s {
		return ErrDenied
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return s.closeErr
	}
	s.closed, s.blocked = true, true
	n := s.native
	if n == nil {
		return ErrDenied
	}
	if n.file != nil {
		s.closeErr = n.file.Close()
	}
	for i := len(n.chains) - 1; i >= 0; i-- {
		s.closeErr = errors.Join(s.closeErr, closeControlChain(n.chains[i]))
	}
	return s.closeErr
}
