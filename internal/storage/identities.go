// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ParseIdentities accepts a successful, bounded full pool projection from:
// zfs list -H -p -r -t filesystem,volume,snapshot,bookmark -o name,guid,type.
// Syntactic validation cannot establish native completeness or eligibility.
func ParseIdentities(data []byte, poolName string) ([]DatasetIdentity, error) {
	if len(data) > maxObservationBytes || bytes.Count(data, []byte{'\n'}) > maxRecords {
		return nil, ErrLimit
	}
	if len(data) == 0 || !utf8.Valid(data) {
		return nil, ErrInvalid
	}
	var ids []DatasetIdentity
	for len(data) > 0 {
		end := bytes.IndexByte(data, '\n')
		if end > 3*maxFieldBytes+2 {
			return nil, ErrLimit
		}
		if end < 0 || bytes.Count(data[:end], []byte{'\t'}) != 2 {
			return nil, ErrInvalid
		}
		fields := strings.Split(string(data[:end]), "\t")
		for _, field := range fields {
			if len(field) > maxFieldBytes {
				return nil, ErrLimit
			}
			if field == "" {
				return nil, ErrInvalid
			}
			for _, c := range field {
				if unicode.IsControl(c) {
					return nil, ErrInvalid
				}
			}
		}
		ids = append(ids, DatasetIdentity{GUID: fields[1], Name: fields[0], Type: DatasetType(fields[2])})
		data = data[end+1:]
	}
	// The native projection has names, not owner GUIDs. Resolve owners through the
	// same table; received snapshots and their bookmarks can share a source GUID.
	owners := make(map[string]string, len(ids))
	for _, id := range ids {
		if id.Type == FilesystemType || id.Type == VolumeType {
			owners[id.Name] = id.GUID
		}
	}
	for n, id := range ids {
		if id.Type == SnapshotType || id.Type == BookmarkType {
			if at := strings.IndexAny(id.Name, "@#"); at >= 0 {
				ids[n].OwnerGUID = owners[id.Name[:at]]
			}
		}
	}
	if err := validateIdentities(poolName, ids); err != nil {
		return nil, err
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].Name < ids[j].Name })
	return ids, nil
}

func identityBudget(ids []DatasetIdentity) error {
	if len(ids) > maxRecords {
		return ErrLimit
	}
	remaining := maxObservationBytes
	for _, id := range ids {
		remaining -= 256
		for _, field := range []string{id.Name, id.GUID, string(id.Type), id.OwnerGUID} {
			if len(field) > maxFieldBytes || len(field) > remaining {
				return ErrLimit
			}
			remaining -= len(field)
		}
	}
	return nil
}

func validateIdentities(poolName string, ids []DatasetIdentity) error {
	if err := identityBudget(ids); err != nil {
		return err
	}
	if !validName(poolName) || strings.Contains(poolName, "/") || len(ids) == 0 {
		return ErrInvalid
	}
	names := make(map[string]DatasetIdentity, len(ids))
	guids := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !validGUID(id.GUID) {
			return ErrInvalid
		}
		if _, exists := names[id.Name]; exists {
			return ErrInvalid
		}
		owner := id.Name
		switch id.Type {
		case FilesystemType, VolumeType:
			if id.OwnerGUID != "" || guids[id.GUID] {
				return ErrInvalid
			}
			guids[id.GUID] = true
		case SnapshotType, BookmarkType:
			delimiter := byte('@')
			if id.Type == BookmarkType {
				delimiter = '#'
			}
			at := strings.IndexByte(id.Name, delimiter)
			if at < 1 || strings.Count(id.Name, string(delimiter)) != 1 {
				return ErrInvalid
			}
			owner = id.Name[:at]
			tail := id.Name[at+1:]
			if !token(tail) || strings.ContainsAny(tail, ".:") || tail[0] == '-' || tail[0] == '_' || !validGUID(id.OwnerGUID) {
				return ErrInvalid
			}
		default:
			return ErrInvalid
		}
		if !validName(owner) || (owner != poolName && !strings.HasPrefix(owner, poolName+"/")) {
			return ErrInvalid
		}
		names[id.Name] = id
	}
	if names[poolName].Type != FilesystemType {
		return ErrInvalid
	}
	for _, id := range ids {
		switch id.Type {
		case FilesystemType, VolumeType:
			if id.Name == poolName {
				continue
			}
			slash := strings.LastIndexByte(id.Name, '/')
			if slash < 0 || names[id.Name[:slash]].Type != FilesystemType {
				return ErrInvalid
			}
		case SnapshotType, BookmarkType:
			at := strings.IndexAny(id.Name, "@#")
			owner, ok := names[id.Name[:at]]
			if !ok || (owner.Type != FilesystemType && owner.Type != VolumeType) || id.OwnerGUID != owner.GUID {
				return ErrInvalid
			}
		}
	}
	return nil
}

// VerifyFilesystemIdentities binds independent property and full type tables.
// Volume impacts are not represented by this slice, so any volume refuses pool
// qualification. Matching tables do not prove eligibility or native freshness.
func VerifyFilesystemIdentities(poolName string, files []Filesystem, ids []DatasetIdentity) error {
	if err := preflight(Policy{}, Observation{Filesystems: files, Identities: ids}); err != nil {
		return err
	}
	if err := validateIdentities(poolName, ids); err != nil {
		return err
	}
	observed, err := indexFilesystems(poolName, files)
	if err != nil {
		return err
	}
	count := 0
	for _, id := range ids {
		if id.Type == VolumeType {
			return ErrUnavailable
		}
		if id.Type != FilesystemType {
			continue
		}
		count++
		if f, ok := observed[id.GUID]; !ok || f.Name != id.Name {
			return ErrDrift
		}
	}
	if count != len(files) {
		return ErrDrift
	}
	return nil
}

// VerifyIdentityStability checks two independent enumerations around property
// collection. This detects observed changes; it is not an atomic native fence
// and cannot rule out changes that disappear between reads.
func VerifyIdentityStability(poolName string, before, after []DatasetIdentity) error {
	for _, ids := range [][]DatasetIdentity{before, after} {
		if err := validateIdentities(poolName, ids); err != nil {
			return err
		}
	}
	left := append([]DatasetIdentity(nil), before...)
	right := append([]DatasetIdentity(nil), after...)
	sort.Slice(left, func(i, j int) bool { return left[i].Name < left[j].Name })
	sort.Slice(right, func(i, j int) bool { return right[i].Name < right[j].Name })
	if !reflect.DeepEqual(left, right) {
		return ErrDrift
	}
	return nil
}
