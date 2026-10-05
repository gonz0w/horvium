// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var stateProperties = [...]string{"guid", "mountpoint", "canmount", "readonly", "mounted", "encryption", "encryptionroot", "keystatus"}

func stateValues(s FilesystemState) [7]Value {
	return [7]Value{s.Mountpoint, s.CanMount, s.Readonly, s.Mounted, s.Encryption, s.EncryptionRoot, s.KeyStatus}
}

// ParseFilesystemStates consumes only the successful complete fixed projection:
// zfs get -H -p -r -t filesystem -o name,property,value,received,source
// guid,mountpoint,canmount,readonly,mounted,encryption,encryptionroot,keystatus.
// It preserves observations; command success, completeness, pool/system/topology
// eligibility and independent identity checks remain the trusted collector's job.
func ParseFilesystemStates(data []byte, poolName string) ([]FilesystemState, error) {
	if len(data) > maxObservationBytes || bytes.Count(data, []byte{'\n'}) > 8*maxRecords {
		return nil, ErrLimit
	}
	if len(data) == 0 || !utf8.Valid(data) || !validName(poolName) || strings.Contains(poolName, "/") {
		return nil, ErrInvalid
	}
	type group struct {
		rows [8]propertyRow
		seen [8]bool
	}
	groups := map[string]*group{}
	for len(data) > 0 {
		end := bytes.IndexByte(data, '\n')
		if end > 5*maxFieldBytes+4 {
			return nil, ErrLimit
		}
		if end < 0 || bytes.Count(data[:end], []byte{'\t'}) != 4 {
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
		name := fields[0]
		if !validName(name) || (name != poolName && !strings.HasPrefix(name, poolName+"/")) {
			return nil, ErrInvalid
		}
		property := -1
		for n, p := range stateProperties {
			if p == fields[1] {
				property = n
				break
			}
		}
		if property < 0 {
			return nil, ErrInvalid
		}
		g := groups[name]
		if g == nil {
			if len(groups) >= maxRecords {
				return nil, ErrLimit
			}
			g = &group{}
			groups[name] = g
		}
		if g.seen[property] {
			return nil, ErrInvalid
		}
		g.seen[property] = true
		g.rows[property] = propertyRow{fields[2], fields[3], fields[4]}
		data = data[end+1:]
	}
	guids := make(map[string]string, len(groups))
	for name, g := range groups {
		for _, seen := range g.seen {
			if !seen {
				return nil, ErrInvalid
			}
		}
		if !validGUID(g.rows[0].value) || g.rows[0].received != "-" || g.rows[0].source != "-" {
			return nil, ErrInvalid
		}
		guids[name] = g.rows[0].value
	}
	result := make([]FilesystemState, 0, len(groups))
	for name, g := range groups {
		s := FilesystemState{GUID: guids[name], Name: name}
		var values [7]Value
		for n, row := range g.rows[1:] {
			v := Value{Effective: row.value, Source: Source(row.source)}
			if row.received != "-" {
				v.ReceivedValue = row.received
			}
			switch v.Source {
			case Local, Received, Temporary:
				v.SourceGUID = s.GUID
			case Default, NoSource:
			default:
				const prefix = "inherited from "
				if !strings.HasPrefix(row.source, prefix) {
					return nil, ErrInvalid
				}
				v.Source = Inherited
				v.SourceGUID = guids[strings.TrimPrefix(row.source, prefix)]
			}
			values[n] = v
		}
		s.Mountpoint, s.CanMount, s.Readonly, s.Mounted, s.Encryption, s.EncryptionRoot, s.KeyStatus = values[0], values[1], values[2], values[3], values[4], values[5], values[6]
		if s.EncryptionRoot.Effective != "-" {
			s.EncryptionRootGUID = guids[s.EncryptionRoot.Effective]
		}
		result = append(result, s)
	}
	if _, err := indexFilesystemStates(poolName, result); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func stateLiteral(value string, n int) bool {
	switch n {
	case 0:
		return value == "none" || value == "legacy" || strings.HasPrefix(value, "/")
	case 1:
		return value == "on" || value == "off" || value == "noauto"
	case 2:
		return value == "on" || value == "off"
	case 3:
		return value == "yes" || value == "no"
	case 4:
		return value == "off" || value == "on" || value == "aes-128-ccm" || value == "aes-192-ccm" || value == "aes-256-ccm" || value == "aes-128-gcm" || value == "aes-192-gcm" || value == "aes-256-gcm"
	case 5:
		return value == "-" || validName(value)
	case 6:
		return value == "none" || value == "-" || value == "available" || value == "unavailable"
	}
	return false
}

func stateValueValid(v Value, n int, guid string) bool {
	for _, field := range []string{v.Effective, string(v.Source), v.SourceGUID, v.ReceivedValue} {
		if !utf8.ValidString(field) {
			return false
		}
		for _, c := range field {
			if unicode.IsControl(c) {
				return false
			}
		}
	}
	if !stateLiteral(v.Effective, n) || (v.ReceivedValue != "" && !stateLiteral(v.ReceivedValue, n)) {
		return false
	}
	if n == 3 || n == 5 || n == 6 {
		return (v.Source == NoSource || n != 3 && v.Source == Default) && v.SourceGUID == "" && v.ReceivedValue == ""
	}
	if v.Source == Received && v.ReceivedValue != v.Effective {
		return false
	}
	switch v.Source {
	case Default:
		return v.SourceGUID == ""
	case Local, Received:
		return v.SourceGUID == guid
	case Inherited:
		return n != 1 && validGUID(v.SourceGUID) && v.SourceGUID != guid
	case NoSource:
		return n == 4 && v.SourceGUID == "" && v.ReceivedValue == ""
	case Temporary:
		return n == 2 && v.SourceGUID == guid
	}
	return false
}

func indexFilesystemStates(poolName string, states []FilesystemState) (map[string]FilesystemState, error) {
	if err := preflight(Policy{}, Observation{States: states}); err != nil {
		return nil, err
	}
	if !validName(poolName) || strings.Contains(poolName, "/") || len(states) == 0 {
		return nil, ErrInvalid
	}
	files := make(map[string]FilesystemState, len(states))
	names := make(map[string]string, len(states))
	for _, s := range states {
		if !validGUID(s.GUID) || !validName(s.Name) || (s.Name != poolName && !strings.HasPrefix(s.Name, poolName+"/")) {
			return nil, ErrInvalid
		}
		if _, ok := files[s.GUID]; ok {
			return nil, ErrInvalid
		}
		if names[s.Name] != "" {
			return nil, ErrInvalid
		}
		for n, v := range stateValues(s) {
			if !stateValueValid(v, n, s.GUID) {
				return nil, ErrInvalid
			}
		}
		files[s.GUID] = s
		names[s.Name] = s.GUID
	}
	if names[poolName] == "" {
		return nil, ErrInvalid
	}
	for _, s := range states {
		if s.Name != poolName {
			slash := strings.LastIndexByte(s.Name, '/')
			if slash < 0 || names[s.Name[:slash]] == "" {
				return nil, ErrInvalid
			}
		}
		for n, v := range stateValues(s) {
			if v.Source != Inherited {
				continue
			}
			ancestor, ok := files[v.SourceGUID]
			if !ok || !strings.HasPrefix(s.Name, ancestor.Name+"/") {
				return nil, ErrInvalid
			}
			supplied := stateValues(ancestor)[n]
			if supplied.Source != Local && supplied.Source != Received {
				return nil, ErrInvalid
			}
			if (n == 2 || n == 4) && v.Effective != supplied.Effective {
				return nil, ErrInvalid
			}
		}
		if s.Encryption.Effective == "off" {
			if s.EncryptionRoot.Effective != "-" || s.EncryptionRootGUID != "" || (s.KeyStatus.Effective != "none" && s.KeyStatus.Effective != "-") {
				return nil, ErrInvalid
			}
		} else {
			root, ok := files[s.EncryptionRootGUID]
			if !ok || root.Name != s.EncryptionRoot.Effective || (root.Name != s.Name && !strings.HasPrefix(s.Name, root.Name+"/")) || root.Encryption.Effective != s.Encryption.Effective || root.EncryptionRootGUID != root.GUID || (s.KeyStatus.Effective != "available" && s.KeyStatus.Effective != "unavailable") {
				return nil, ErrInvalid
			}
		}
	}
	return files, nil
}

// VerifyFilesystemStates binds complete mount/crypto observations to full native
// identities and refuses encrypted, read-only or temporary-readonly states.
// It supplies only these checks, never the remaining native eligibility proof.
func VerifyFilesystemStates(poolName string, ids []DatasetIdentity, states []FilesystemState) error {
	if err := preflight(Policy{}, Observation{Identities: ids, States: states}); err != nil {
		return err
	}
	if err := validateIdentities(poolName, ids); err != nil {
		return err
	}
	files, err := indexFilesystemStates(poolName, states)
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
		s, ok := files[id.GUID]
		if !ok || s.Name != id.Name {
			return ErrDrift
		}
		if s.Encryption.Effective != "off" || s.Readonly.Effective != "off" || s.Readonly.Source == Temporary {
			return ErrUnavailable
		}
	}
	if count != len(states) {
		return ErrDrift
	}
	return nil
}
