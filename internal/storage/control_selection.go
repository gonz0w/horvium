// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"strings"
	"sync"
)

// ErrControlRelease records an unconfirmed temporary descriptor release. It
// suppresses output even when the retained selection's own Close succeeds.
var ErrControlRelease = errors.New("storage control release unconfirmed")

// ControlSelection owns the original no-follow selection descriptor and native
// path custody. It is not a writer, admission permit or serializable authority.
// Do not copy; Check retires drift and Close always attempts original release.
type ControlSelection struct {
	mu              sync.Mutex
	original        *ControlSelection
	targets         ControlTargets
	native          *controlSelectionNative
	blocked, closed bool
	closeErr        error
}

func (s *ControlSelection) Targets() (ControlTargets, error) {
	if s == nil || s.original != s {
		return ControlTargets{}, ErrDenied
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.blocked {
		return ControlTargets{}, ErrDenied
	}
	return s.targets, nil
}

// Live native mount facts only. This is not persistent backing identity or
// qualification of restoration; SC-N04 remains required before registration.
func checkControlRestoreScope(config, domain propertyStateBacking) error {
	if !validControlFilesystem(config) || !validControlFilesystem(domain) {
		return ErrUnavailable
	}
	if config.kind == "zfs" && domain.kind == "zfs" && (config.fsid == domain.fsid || config.source == domain.source || strings.HasPrefix(config.source, domain.source+"/")) {
		return ErrDenied
	}
	return nil
}

func validControlFilesystem(b propertyStateBacking) bool {
	if b.readonly || b.softupdates || b.fsid == ([2]int32{}) || len(b.source) > 1023 || len(b.target) > 1023 || (b.target != "/" && !ValidControlPath(b.target)) {
		return false
	}
	if b.kind == "ufs" {
		return strings.HasPrefix(b.source, "/dev/") && ValidControlPath(b.source)
	}
	if b.kind != "zfs" || len(b.source) == 0 {
		return false
	}
	for _, part := range strings.Split(b.source, "/") {
		if part == "" || part == "." || part == ".." || part[0] == '-' {
			return false
		}
		for _, r := range part {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' || r == ':') {
				return false
			}
		}
	}
	return true
}
