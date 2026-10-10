// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

// Exclusive namespace provenance only. This does not initialize a journal,
// establish boot/restore provenance or register a runtime/global writer.
// The original anchors survive until close. Never reconstruct this from disk.
type freshExecutionDomains struct {
	mu              sync.Mutex
	original        *freshExecutionDomains
	pair            *ExecutionDomainLeases
	anchors         [2]freshExecutionAnchor
	closed, blocked bool
}
type freshExecutionAnchor struct {
	parent                          *os.File
	path, name                      string
	parentIdentity, directory, lock unix.Stat_t
	backing                         propertyStateBacking
}

// Private OS-root provisioning component. Both leaf names must be absent below
// already protected local parents. Failure releases descriptors but preserves
// every visible partial object; neither retry nor reopen can obtain freshness.
// step is a private synchronous native test seam, never client configuration.
func provisionFreshExecutionDomains(ctx context.Context, journal, ledger string, step func(string, string) error) (*freshExecutionDomains, error) {
	if ctx == nil || os.Geteuid() != 0 {
		return nil, ErrDenied
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	for _, path := range []string{journal, ledger} {
		if len(path) > 4096 || !filepath.IsAbs(path) || path == "/" || filepath.Clean(path) != path || !utf8.ValidString(path) || strings.IndexFunc(path, func(r rune) bool { return r < 32 || r == 127 }) >= 0 {
			return nil, ErrStateDirectory
		}
	}
	if journal == ledger || strings.HasPrefix(journal, ledger+"/") || strings.HasPrefix(ledger, journal+"/") {
		return nil, ErrDenied
	}
	f := &freshExecutionDomains{}
	f.original = f
	var states [2]*StateDirectory
	fail := func(e error) (*freshExecutionDomains, error) {
		for i := 1; i >= 0; i-- {
			if states[i] != nil {
				e = errors.Join(e, states[i].Close())
			}
			if f.anchors[i].parent != nil {
				e = errors.Join(e, f.anchors[i].parent.Close())
			}
		}
		return nil, e
	}
	// Validate both parents and both missing names before the first creation.
	for i, path := range []string{journal, ledger} {
		parent, e := openFreshExecutionParent(filepath.Dir(path))
		if e != nil {
			return fail(e)
		}
		a := &f.anchors[i]
		a.parent = parent
		a.path = filepath.Dir(path)
		a.name = filepath.Base(path)
		fd := int(parent.Fd())
		if unix.Fstat(fd, &a.parentIdentity) != nil {
			return fail(ErrStateDirectory)
		}
		a.backing, e = propertyStateFilesystem(fd)
		if e != nil {
			return fail(e)
		}
		var named unix.Stat_t
		if e = unix.Fstatat(fd, a.name, &named, unix.AT_SYMLINK_NOFOLLOW); e != unix.ENOENT {
			return fail(ErrDenied)
		}
	}
	for i, role := range []string{"journal", "ledger"} {
		s, e := createFreshExecutionState(ctx, role, &f.anchors[i], step)
		if e != nil {
			return fail(e)
		}
		states[i] = s
	}
	j := &executionJournalStore{lease: states[0]}
	j.original = j
	// The namespace receipt alone cannot supply an active journal initializer.
	p := &ExecutionDomainLeases{native: &executionDomainLeaseNative{j, states[1]}}
	p.origin = p
	f.pair = p
	if e := f.inspect(ctx); e != nil {
		return fail(e)
	}
	return f, nil
}

func openFreshExecutionParent(path string) (*os.File, error) {
	flags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC
	fd, e := unix.Open("/", flags, 0)
	if e != nil {
		return nil, ErrStateDirectory
	}
	fail := func() (*os.File, error) { _ = unix.Close(fd); return nil, ErrStateDirectory }
	parts := []string{}
	if path != "/" {
		parts = strings.Split(strings.TrimPrefix(path, "/"), "/")
	}
	for i := -1; i < len(parts); i++ {
		if i >= 0 {
			next, e := unix.Openat(fd, parts[i], flags, 0)
			_ = unix.Close(fd)
			if e != nil {
				return nil, ErrStateDirectory
			}
			fd = next
		}
		var st unix.Stat_t
		if unix.Fstat(fd, &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != 0 || st.Mode&07022 != 0 || !safeStateACL(fd, uint32(st.Mode), true) {
			return fail()
		}
	}
	b, e := propertyStateFilesystem(fd)
	if e != nil || b.readonly || b.softupdates || b.kind != "ufs" && b.kind != "zfs" {
		return fail()
	}
	return os.NewFile(uintptr(fd), "storage-fresh-parent"), nil
}

func createFreshExecutionState(ctx context.Context, role string, a *freshExecutionAnchor, step func(string, string) error) (*StateDirectory, error) {
	guard := func(stage string) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if os.Geteuid() != 0 {
			return ErrDenied
		}
		if step != nil {
			if e := step(role, stage); e != nil {
				return e
			}
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if os.Geteuid() != 0 {
			return ErrDenied
		}
		return nil
	}
	parent := int(a.parent.Fd())
	if e := guard("mkdir"); e != nil {
		return nil, e
	}
	if unix.Mkdirat(parent, a.name, 0700) != nil {
		return nil, ErrStateDirectory
	}
	var named unix.Stat_t
	if unix.Fstatat(parent, a.name, &named, unix.AT_SYMLINK_NOFOLLOW) != nil {
		return nil, ErrStateDirectory
	}
	if e := guard("open"); e != nil {
		return nil, e
	}
	fd, e := unix.Openat(parent, a.name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if e != nil {
		return nil, ErrStateDirectory
	}
	s := &StateDirectory{directory: os.NewFile(uintptr(fd), "storage-fresh-directory")}
	s.propertyOrigin = s
	fail := func(e error) (*StateDirectory, error) { return nil, errors.Join(e, s.Close()) }
	var opened unix.Stat_t
	if unix.Fstat(fd, &opened) != nil || !sameOwnershipMetadata(named, opened) || opened.Mode&07777 != 0700 || opened.Uid != 0 || !safeStateACL(fd, uint32(opened.Mode), true) {
		return fail(ErrStateDirectory)
	}
	if e = guard("lock"); e != nil {
		return fail(e)
	}
	lock, e := unix.Openat(fd, stateWriterName, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0600)
	if e != nil {
		return fail(ErrStateDirectory)
	}
	s.writer = os.NewFile(uintptr(lock), "storage-fresh-writer")
	if unix.Fstat(lock, &a.lock) != nil || !validStateLock(a.lock) || !safeStateACL(lock, uint32(a.lock.Mode), false) || unix.Flock(lock, unix.LOCK_EX|unix.LOCK_NB) != nil {
		return fail(ErrStateDirectory)
	}
	for _, sync := range []struct {
		stage string
		fd    int
	}{{"sync-lock", lock}, {"sync-directory", fd}, {"sync-parent", parent}} {
		if e = guard(sync.stage); e != nil {
			return fail(e)
		}
		if unix.Fsync(sync.fd) != nil {
			return fail(ErrStateDirectory)
		}
	}
	if unix.Fstat(fd, &a.directory) != nil {
		return fail(ErrStateDirectory)
	}
	if e = guard("after-parent-sync"); e != nil {
		return fail(e)
	}
	if e = guard("verify"); e != nil {
		return fail(e)
	}
	if e = a.inspectLocked(s); e != nil {
		return fail(e)
	}
	return s, nil
}

// Caller retains the original state mutex. Reopening the parent is inspection
// only: it must match the held original parent, and can never replace it.
func (a *freshExecutionAnchor) inspectLocked(s *StateDirectory) error {
	if a.parent == nil || s == nil || s.propertyOrigin != s {
		return ErrDenied
	}
	dir, e := s.ownershipDirectoryLocked()
	if e != nil {
		return e
	}
	var ds, ls, named, parent unix.Stat_t
	if unix.Fstat(dir, &ds) != nil || !sameOwnershipMetadata(ds, a.directory) || unix.Fstat(int(s.writer.Fd()), &ls) != nil || !sameOwnershipMetadata(ls, a.lock) || unix.Fstatat(int(a.parent.Fd()), a.name, &named, unix.AT_SYMLINK_NOFOLLOW) != nil || !sameOwnershipMetadata(ds, named) {
		return ErrDrift
	}
	current, e := openFreshExecutionParent(a.path)
	if e != nil {
		return e
	}
	defer current.Close()
	if unix.Fstat(int(current.Fd()), &parent) != nil || parent.Dev != a.parentIdentity.Dev || executionStatIdentity(parent) != executionStatIdentity(a.parentIdentity) {
		return ErrDrift
	}
	backing, e := propertyStateFilesystem(int(a.parent.Fd()))
	if e != nil || backing != a.backing {
		return ErrDrift
	}
	return nil
}

// Both original leases and the protected parent entries must still describe the
// untouched fresh namespaces. Any inspection uncertainty permanently retires it.
func (f *freshExecutionDomains) inspect(ctx context.Context) error {
	if f == nil || ctx == nil || os.Geteuid() != 0 {
		return ErrDenied
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.original != f || f.closed || f.pair == nil {
		return ErrDenied
	}
	if f.blocked {
		return ErrStateUncertain
	}
	p := f.pair
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.origin != p || p.closed || p.blocked || p.native == nil {
		return ErrDenied
	}
	j, l := p.native.journal, p.native.ledger
	if j == nil || j.original != j || j.lease == nil || l == nil {
		return ErrDenied
	}
	j.lease.mu.Lock()
	defer j.lease.mu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	success := false
	defer func() {
		if !success {
			f.blocked = true
			p.blocked = true
			j.retireLocked()
			l.publicationBlocked = true
		}
	}()
	if j.failed || j.acknowledged || j.initializable || j.lease.publicationBlocked || l.publicationBlocked {
		return ErrDenied
	}
	for i, s := range []*StateDirectory{j.lease, l} {
		if e := f.anchors[i].inspectLocked(s); e != nil {
			return e
		}
	}
	journal, e := j.inspectLocked("", nil)
	if e != nil {
		return e
	}
	if journal.present {
		return ErrDenied
	}
	catalog, e := l.inspectStateLocked(defaultStateCatalogLimits, nil, "")
	if e != nil {
		return e
	}
	if len(catalog.Ownership) != 0 || len(catalog.Jobs) != 0 {
		return ErrDenied
	}
	for i, s := range []*StateDirectory{j.lease, l} {
		if e := f.anchors[i].inspectLocked(s); e != nil {
			return e
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if os.Geteuid() != 0 {
		return ErrDenied
	}
	success = true
	return nil
}
func (f *freshExecutionDomains) close() error {
	if f == nil || f.original != f {
		return ErrDenied
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	f.blocked = true
	var e error
	if f.pair != nil {
		e = f.pair.Close()
	}
	for i := 1; i >= 0; i-- {
		if f.anchors[i].parent != nil {
			e = errors.Join(e, f.anchors[i].parent.Close())
			f.anchors[i].parent = nil
		}
	}
	return e
}
