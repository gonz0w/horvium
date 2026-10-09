// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Private read-only facts. This collector does not own the original state lease,
// establish system/owned-pool exclusion or grant commissioning/restore authority.
type persistentBackingIdentity struct {
	kind          string
	ufs           [2]uint32
	pool, dataset uint64
}
type persistentBackingMount struct {
	kind, source, target string
	fsid                 [2]int32
	flags                uint64
}
type persistentBackingReader func(context.Context, string, string, int) (readOutput, error)
type persistentBackingCollector struct {
	read                         persistentBackingReader
	observe                      func() (persistentBackingMount, error)
	bytes                        int
	commandTimeout, totalTimeout time.Duration
}

func (s *persistentBackingCollector) collect(ctx context.Context) (persistentBackingIdentity, error) {
	if ctx == nil {
		return persistentBackingIdentity{}, ErrUnavailable
	}
	if s == nil || s.read == nil || s.observe == nil || s.commandTimeout <= 0 || s.totalTimeout <= 0 {
		return persistentBackingIdentity{}, ErrUnavailable
	}
	if s.bytes <= 0 || s.bytes > 64<<10 {
		return persistentBackingIdentity{}, ErrLimit
	}
	ctx, cancel := context.WithTimeout(ctx, s.totalTimeout)
	defer cancel()
	remaining := s.bytes
	capture := func(probe, name string) ([]byte, error) {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		if remaining <= 0 {
			return nil, ErrLimit
		}
		child, stop := context.WithTimeout(ctx, s.commandTimeout)
		defer stop()
		out, e := s.read(child, probe, name, remaining)
		if out.consumed < len(out.data) || out.consumed > remaining {
			return nil, ErrLimit
		}
		remaining -= out.consumed
		if e != nil {
			return nil, e
		}
		if e = child.Err(); e != nil {
			return nil, e
		}
		return out.data, nil
	}
	var first persistentBackingIdentity
	var firstMount persistentBackingMount
	version := ""
	for pass := 0; pass < 2; pass++ {
		m, e := s.observe()
		if e != nil {
			return persistentBackingIdentity{}, e
		}
		if !validPersistentMount(m) {
			return persistentBackingIdentity{}, ErrUnavailable
		}
		v, e := capture("versions", "")
		if e != nil {
			return persistentBackingIdentity{}, e
		}
		if !persistentVersions(v) {
			return persistentBackingIdentity{}, ErrUnavailable
		}
		id := persistentBackingIdentity{kind: m.kind}
		if m.kind == "ufs" {
			data, e := capture("ufs-id", m.source)
			if e != nil {
				return persistentBackingIdentity{}, e
			}
			id.ufs, e = parsePersistentUFS(data)
			if e != nil {
				return persistentBackingIdentity{}, e
			}
			if id.ufs != ([2]uint32{uint32(m.fsid[0]), uint32(m.fsid[1])}) {
				return persistentBackingIdentity{}, ErrUnavailable
			}
		} else {
			data, e := capture("zfs-id", m.source)
			if e != nil {
				return persistentBackingIdentity{}, e
			}
			id.dataset, e = parsePersistentZFS(data, m.source, m.target)
			if e != nil {
				return persistentBackingIdentity{}, e
			}
			pool := strings.Split(m.source, "/")[0]
			data, e = capture("pool-id", pool)
			if e != nil {
				return persistentBackingIdentity{}, e
			}
			id.pool, e = persistentNumber(strings.TrimSuffix(string(data), "\n"))
			if e != nil || strings.Count(string(data), "\n") != 1 || len(data) == 0 || data[len(data)-1] != '\n' {
				return persistentBackingIdentity{}, ErrUnavailable
			}
		}
		after, e := s.observe()
		if e != nil {
			return persistentBackingIdentity{}, e
		}
		if after != m {
			return persistentBackingIdentity{}, ErrUnavailable
		}
		if pass == 0 {
			first = id
			firstMount = m
			version = string(v)
		} else if first != id || firstMount != m || version != string(v) {
			return persistentBackingIdentity{}, ErrUnavailable
		}
	}
	if e := ctx.Err(); e != nil {
		return persistentBackingIdentity{}, e
	}
	return first, nil
}

func persistentVersions(data []byte) bool {
	if len(data) > 512 || strings.Count(string(data), "\n") != 3 || len(data) == 0 || data[len(data)-1] != '\n' {
		return false
	}
	format := ""
	for _, v := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		m := propertyReleasePattern.FindStringSubmatch(v)
		if len(m) != 2 || format != "" && format != m[1] {
			return false
		}
		format = m[1]
	}
	return true
}
func validPersistentMount(m persistentBackingMount) bool {
	if len(m.target) > 1023 || !filepath.IsAbs(m.target) || filepath.Clean(m.target) != m.target || strings.IndexFunc(m.target, func(r rune) bool { return r < 32 || r == 127 }) >= 0 {
		return false
	}
	switch m.kind {
	case "ufs":
		return validPersistentProvider(m.source)
	case "zfs":
		return validPersistentDataset(m.source)
	}
	return false
}
func validPersistentProvider(name string) bool {
	return strings.HasPrefix(name, "/dev/") && len(name) <= 1023 && filepath.Clean(name) == name && persistentComponents(strings.TrimPrefix(name, "/dev/"))
}
func validPersistentDataset(name string) bool {
	if len(name) == 0 || len(name) > 1023 || name[0] < 'A' || name[0] > 'Z' && name[0] < 'a' || name[0] > 'z' {
		return false
	}
	return persistentComponents(name)
}
func persistentComponents(name string) bool {
	parts := strings.Split(name, "/")
	if len(parts) > 32 {
		return false
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." || p[0] == '-' {
			return false
		}
		for _, r := range p {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' || r == ':') {
				return false
			}
		}
	}
	return true
}
func parsePersistentUFS(data []byte) ([2]uint32, error) {
	var id [2]uint32
	if len(data) != 28 || string(data[:11]) != "/dev/ufsid/" || data[27] != '\n' {
		return id, ErrUnavailable
	}
	for i := range id {
		s := string(data[11+i*8 : 19+i*8])
		n, e := strconv.ParseUint(s, 16, 32)
		if e != nil || n == 0 || strings.ToLower(s) != s || strings.Trim(s, "0123456789abcdef") != "" {
			return [2]uint32{}, ErrUnavailable
		}
		id[i] = uint32(n)
	}
	return id, nil
}
func persistentNumber(s string) (uint64, error) {
	if len(s) > 20 {
		return 0, ErrUnavailable
	}
	n, e := strconv.ParseUint(s, 10, 64)
	if e != nil || n == 0 || strconv.FormatUint(n, 10) != s {
		return 0, ErrUnavailable
	}
	return n, nil
}
func parsePersistentZFS(data []byte, name, target string) (uint64, error) {
	if len(data) > 8192 || strings.Count(string(data), "\n") != 4 || len(data) == 0 || data[len(data)-1] != '\n' {
		return 0, ErrUnavailable
	}
	values := map[string]string{}
	for _, row := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		f := strings.Split(row, "\t")
		if len(f) != 3 || f[0] != name || f[2] == "" || values[f[1]] != "" {
			return 0, ErrUnavailable
		}
		values[f[1]] = f[2]
	}
	if len(values) != 4 || values["type"] != "filesystem" || values["mounted"] != "yes" || values["mountpoint"] != target && values["mountpoint"] != "legacy" {
		return 0, ErrUnavailable
	}
	return persistentNumber(values["guid"])
}
