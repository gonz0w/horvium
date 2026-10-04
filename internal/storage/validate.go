// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/sha256"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

const (
	maxRecords          = 100000
	maxFieldBytes       = 4096
	maxObservationBytes = 64 << 20
	maxPlanBytes        = 64 << 10
)

func validGUID(s string) bool {
	n, err := strconv.ParseUint(s, 10, 64)
	return err == nil && n > 0 && strconv.FormatUint(n, 10) == s
}
func validProperty(p Property) bool { return p == Quota || p == RefQuota || p == Compression }
func validLimit(s string) bool      { return s == "none" || validGUID(s) }
func validIntent(i Intent) bool {
	if !validGUID(i.FilesystemGUID) {
		return false
	}
	switch i.Property {
	case Quota, RefQuota:
		return validLimit(i.Value)
	case Compression:
		return i.Value == "off" || i.Value == "lz4"
	}
	return false
}
func token(s string) bool {
	if len(s) < 1 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.' || c == ':') {
			return false
		}
	}
	return true
}

// Deliberately narrow management-name subset. Unsupported native names stay
// observable in M1; this planner cannot authorize them. No native name parser is
// claimed. Pool reserved prefixes are conservative denial, not a force escape.
func validName(name string) bool {
	if len(name) > 1024 {
		return false
	}
	parts := strings.Split(name, "/")
	if len(parts) > 32 {
		return false
	}
	for _, part := range parts {
		if !token(part) {
			return false
		}
		first := part[0]
		if !(first >= 'a' && first <= 'z' || first >= 'A' && first <= 'Z') {
			return false
		}
		for _, c := range part {
			if c == '.' || c == ':' {
				return false
			}
		}
	}
	for _, prefix := range []string{"mirror", "raidz", "draid", "spare", "log"} {
		if strings.HasPrefix(parts[0], prefix) {
			return false
		}
	}
	return true
}
func valueFor(p Properties, property Property) Value {
	switch property {
	case Quota:
		return p.Quota
	case RefQuota:
		return p.RefQuota
	default:
		return p.Compression
	}
}
func validValue(v Value, property Property, guid string) bool {
	if property == Compression {
		if !token(v.Effective) {
			return false
		}
	} else if !validLimit(v.Effective) {
		return false
	}
	switch v.Source {
	case Default:
		return v.SourceGUID == "" && (property == Compression || v.Effective == "none")
	case Local, Received:
		return v.SourceGUID == guid
	case Inherited:
		return property == Compression && validGUID(v.SourceGUID) && v.SourceGUID != guid
	}
	return false
}

// Preflight every count/field and aggregate byte bound BEFORE validation maps,
// sorting or JSON encoding allocate. The estimate includes structural overhead.
func preflight(p Policy, o Observation) error {
	if len(o.Filesystems) > maxRecords || len(p.Scopes) > maxRecords {
		return ErrLimit
	}
	remaining := maxObservationBytes
	debit := func(fields ...string) error {
		if remaining < 256 {
			return ErrLimit
		}
		remaining -= 256
		for _, s := range fields {
			if len(s) > maxFieldBytes || len(s) > remaining {
				return ErrLimit
			}
			remaining -= len(s)
		}
		return nil
	}
	if err := debit(o.PoolGUID, o.PoolName, p.PoolGUID, p.PoolName); err != nil {
		return err
	}
	for _, f := range o.Filesystems {
		if err := debit(f.GUID, f.Name, f.ParentGUID); err != nil {
			return err
		}
		for _, v := range []Value{f.Properties.Quota, f.Properties.RefQuota, f.Properties.Compression} {
			if err := debit(v.Effective, string(v.Source), v.SourceGUID); err != nil {
				return err
			}
		}
	}
	for _, s := range p.Scopes {
		if len(s.Properties) > 3 {
			return ErrInvalid
		}
		if err := debit(s.GUID, s.Name, s.ParentGUID); err != nil {
			return err
		}
		for _, property := range s.Properties {
			if err := debit(string(property)); err != nil {
				return err
			}
		}
	}
	return nil
}

func indexObservation(o Observation) (map[string]Filesystem, error) {
	if !o.Complete || !o.Eligible {
		return nil, ErrUnavailable
	}
	if !validGUID(o.PoolGUID) || !validName(o.PoolName) || strings.Contains(o.PoolName, "/") || len(o.Filesystems) == 0 {
		return nil, ErrInvalid
	}
	files := make(map[string]Filesystem, len(o.Filesystems))
	names := make(map[string]bool, len(o.Filesystems))
	for _, f := range o.Filesystems {
		if !validGUID(f.GUID) || !validName(f.Name) || names[f.Name] {
			return nil, ErrInvalid
		}
		if _, ok := files[f.GUID]; ok {
			return nil, ErrInvalid
		}
		if f.Name != o.PoolName && !strings.HasPrefix(f.Name, o.PoolName+"/") {
			return nil, ErrInvalid
		}
		for _, property := range []Property{Quota, RefQuota, Compression} {
			if !validValue(valueFor(f.Properties, property), property, f.GUID) {
				return nil, ErrInvalid
			}
		}
		files[f.GUID] = f
		names[f.Name] = true
	}
	if !names[o.PoolName] {
		return nil, ErrInvalid
	}
	for _, f := range o.Filesystems {
		if f.Name == o.PoolName {
			if f.ParentGUID != "" || f.Properties.Compression.Source == Inherited {
				return nil, ErrInvalid
			}
			continue
		}
		parent, ok := files[f.ParentGUID]
		slash := strings.LastIndexByte(f.Name, '/')
		if !ok || slash < 0 || parent.Name != f.Name[:slash] {
			return nil, ErrInvalid
		}
		childValue, parentValue := f.Properties.Compression, parent.Properties.Compression
		switch childValue.Source {
		case Inherited:
			if parentValue.Source == Default || childValue.SourceGUID != parentValue.SourceGUID || childValue.Effective != parentValue.Effective {
				return nil, ErrInvalid
			}
		case Default:
			if parentValue.Source != Default || childValue.Effective != parentValue.Effective {
				return nil, ErrInvalid
			}
		}
	}
	return files, nil
}

func indexPolicy(p Policy, o Observation, files map[string]Filesystem) (map[string]Scope, [32]byte, error) {
	var zero [32]byte
	if p.Revision == 0 || !validGUID(p.PoolGUID) || !validName(p.PoolName) {
		return nil, zero, ErrDenied
	}
	if p.PoolGUID != o.PoolGUID || p.PoolName != o.PoolName {
		return nil, zero, ErrDenied
	}
	scopes := make(map[string]Scope, len(p.Scopes))
	canonical := append([]Scope(nil), p.Scopes...)
	for n, s := range canonical {
		f, ok := files[s.GUID]
		if !ok || s.Name != f.Name || s.ParentGUID != f.ParentGUID || len(s.Properties) == 0 {
			return nil, zero, ErrDenied
		}
		if _, ok := scopes[s.GUID]; ok {
			return nil, zero, ErrInvalid
		}
		seen := map[Property]bool{}
		for _, property := range s.Properties {
			if !validProperty(property) || seen[property] {
				return nil, zero, ErrInvalid
			}
			seen[property] = true
		}
		s.Properties = append([]Property(nil), s.Properties...)
		sort.Slice(s.Properties, func(i, j int) bool { return s.Properties[i] < s.Properties[j] })
		canonical[n] = s
		scopes[s.GUID] = s
	}
	sort.Slice(canonical, func(i, j int) bool { return canonical[i].GUID < canonical[j].GUID })
	h := sha256.New()
	encoder := json.NewEncoder(h)
	// Fixed small header and preflight-bounded individual scopes. No whole-policy
	// encoding buffer, and no caller-owned slice is retained by a plan.
	if err := encoder.Encode(struct {
		Revision   uint64
		GUID, Name string
	}{p.Revision, p.PoolGUID, p.PoolName}); err != nil {
		return nil, zero, ErrInvalid
	}
	for _, s := range canonical {
		if err := encoder.Encode(s); err != nil {
			return nil, zero, ErrInvalid
		}
	}
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return scopes, digest, nil
}
func permits(s Scope, property Property) bool {
	for _, p := range s.Properties {
		if p == property {
			return true
		}
	}
	return false
}
