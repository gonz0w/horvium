// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ErrStateRecord = errors.New("invalid or unsupported storage state record")

const maxStateRecordBytes = 64 << 10

// These explicit private types are the v1 ownership record, independent of the
// planning structs' Go field names. Every JSON field is required; none is nullable.
type ownershipScope struct {
	GUID       string     `json:"guid"`
	Name       string     `json:"name"`
	ParentGUID string     `json:"parent_guid"`
	Properties []Property `json:"properties"`
}
type ownershipPolicy struct {
	Revision uint64           `json:"revision"`
	PoolGUID string           `json:"pool_guid"`
	PoolName string           `json:"pool_name"`
	Scopes   []ownershipScope `json:"scopes"`
}
type ownershipContent struct {
	Version uint64          `json:"version"`
	Kind    string          `json:"kind"`
	Policy  ownershipPolicy `json:"policy"`
}
type ownershipRecord struct {
	Version  uint64          `json:"version"`
	Kind     string          `json:"kind"`
	Policy   ownershipPolicy `json:"policy"`
	Checksum string          `json:"checksum"`
}

// EncodeOwnershipRecord returns bounded canonical metadata bytes only. The caller
// must establish root enrollment, actual ownership and fresh native eligibility;
// a policy or checksum is not that authority. No file/store/admission operation
// occurs here. Inputs are owned by the caller throughout the call.
func EncodeOwnershipRecord(p Policy) ([]byte, error) {
	canonical, err := normalizeOwnership(p)
	if err != nil {
		return nil, err
	}
	if len(canonical.Members) > 0 {
		return encodeOwnershipV2(canonical)
	}
	wire := ownershipWire(canonical)
	r := ownershipRecord{Version: 1, Kind: "ownership", Policy: wire, Checksum: ownershipChecksum(wire)}
	b, err := json.Marshal(r)
	if err != nil {
		return nil, ErrStateRecord
	}
	if len(b) > maxStateRecordBytes {
		return nil, ErrLimit
	}
	return b, nil
}

// DecodeOwnershipRecord rejects unknown/duplicate/case-variant/missing fields,
// nulls, unsupported versions, damage and invalid scope semantics. On any error
// it returns no policy and never modifies the input. A valid result still needs
// trusted file provenance and native reconciliation before it can grant writes.
func DecodeOwnershipRecord(data []byte) (Policy, error) {
	if len(data) > maxStateRecordBytes {
		return Policy{}, ErrLimit
	}
	if !utf8.Valid(data) {
		return Policy{}, ErrStateRecord
	}
	if !ownershipShape(data, ownershipRecord{}) {
		return decodeOwnershipV2(data)
	}
	var r ownershipRecord
	if json.Unmarshal(data, &r) != nil || r.Version != 1 || r.Kind != "ownership" {
		return Policy{}, ErrStateRecord
	}
	p := Policy{Revision: r.Policy.Revision, PoolGUID: r.Policy.PoolGUID, PoolName: r.Policy.PoolName, Scopes: make([]Scope, len(r.Policy.Scopes))}
	for i, s := range r.Policy.Scopes {
		p.Scopes[i] = Scope{GUID: s.GUID, Name: s.Name, ParentGUID: s.ParentGUID, Properties: s.Properties}
	}
	canonical, err := normalizeOwnership(p)
	if err != nil {
		return Policy{}, err
	}
	if len(r.Checksum) != 64 || r.Checksum != ownershipChecksum(ownershipWire(canonical)) {
		return Policy{}, ErrStateRecord
	}
	return canonical, nil
}

// Both supported schemas are fixed-depth strict walks. Dispatch never probes
// arbitrary JSON or guesses missing members; each walk rejects unknown shapes.
func ownershipShape(data []byte, record any) bool {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if ownershipJSONValue(d, reflect.TypeOf(record)) != nil {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}

// Shape-directed token traversal never descends into arbitrary JSON: unknown
// keys or wrong container types fail before recursion. The schema has fixed
// depth; the input byte ceiling bounds arrays, strings and per-object key maps.
func ownershipJSONValue(d *json.Decoder, shape reflect.Type) error {
	v, err := d.Token()
	if err != nil || v == nil {
		return ErrStateRecord
	}
	switch shape.Kind() {
	case reflect.Struct:
		if v != json.Delim('{') {
			return ErrStateRecord
		}
		fields := make(map[string]reflect.Type, shape.NumField())
		for i := range shape.NumField() {
			f := shape.Field(i)
			fields[f.Tag.Get("json")] = f.Type
		}
		seen := make(map[string]bool, len(fields))
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return ErrStateRecord
			}
			name, ok := key.(string)
			child, known := fields[name]
			if !ok || !known || seen[name] {
				return ErrStateRecord
			}
			seen[name] = true
			if ownershipJSONValue(d, child) != nil {
				return ErrStateRecord
			}
		}
		end, err := d.Token()
		if err != nil || end != json.Delim('}') || len(seen) != len(fields) {
			return ErrStateRecord
		}
	case reflect.Slice:
		if v != json.Delim('[') {
			return ErrStateRecord
		}
		for d.More() {
			if ownershipJSONValue(d, shape.Elem()) != nil {
				return ErrStateRecord
			}
		}
		end, err := d.Token()
		if err != nil || end != json.Delim(']') {
			return ErrStateRecord
		}
	case reflect.String:
		if _, ok := v.(string); !ok {
			return ErrStateRecord
		}
	// Only the reviewed-property topology shape declares a boolean. Existing
	// ownership/job schemas still accept solely their exact declared types.
	case reflect.Bool:
		if _, ok := v.(bool); !ok {
			return ErrStateRecord
		}
	case reflect.Uint64:
		n, ok := v.(json.Number)
		if !ok {
			return ErrStateRecord
		}
		parsed, err := strconv.ParseUint(string(n), 10, 64)
		if err != nil || strconv.FormatUint(parsed, 10) != string(n) {
			return ErrStateRecord
		}
	default:
		return ErrStateRecord
	}
	return nil
}

func normalizeOwnership(p Policy) (Policy, error) {
	// Account for structural overhead/fields before allocating copies/maps or
	// marshalling. This conservative in-memory budget can refuse a record whose
	// JSON alone fits 64 KiB. No ownership scope is truncated to make it fit.
	if len(p.Scopes) > maxRecords {
		return Policy{}, ErrLimit
	}
	remaining := maxStateRecordBytes - 256
	debit := func(fields ...string) bool {
		remaining -= 128
		for _, s := range fields {
			if len(s) > maxFieldBytes || len(s) > remaining {
				return false
			}
			remaining -= len(s)
		}
		return remaining >= 0
	}
	if !debit(p.PoolGUID, p.PoolName) {
		return Policy{}, ErrLimit
	}
	for _, s := range p.Scopes {
		if len(s.Properties) > 3 {
			return Policy{}, ErrStateRecord
		}
		if !debit(s.GUID, s.Name, s.ParentGUID) {
			return Policy{}, ErrLimit
		}
	}
	if len(p.Members) > maxVdevs {
		return Policy{}, ErrLimit
	}
	for _, m := range p.Members {
		if !debit(m.VdevGUID, m.Serial, m.SerialSource) || remaining < 128 {
			return Policy{}, ErrLimit
		}
		remaining -= 128
	}
	members, err := normalizeMembers(p.Members)
	if err != nil {
		return Policy{}, err
	}
	if p.Revision == 0 || !validGUID(p.PoolGUID) || !validName(p.PoolName) || strings.Contains(p.PoolName, "/") {
		return Policy{}, ErrStateRecord
	}
	canonical := Policy{Revision: p.Revision, PoolGUID: p.PoolGUID, PoolName: p.PoolName, Scopes: make([]Scope, len(p.Scopes)), Members: members}
	guids, names := make(map[string]Scope, len(p.Scopes)), make(map[string]bool, len(p.Scopes))
	for i, s := range p.Scopes {
		if !validGUID(s.GUID) || !validName(s.Name) || names[s.Name] || len(s.Properties) == 0 {
			return Policy{}, ErrStateRecord
		}
		if _, exists := guids[s.GUID]; exists {
			return Policy{}, ErrStateRecord
		}
		if s.Name == p.PoolName {
			if s.ParentGUID != "" {
				return Policy{}, ErrStateRecord
			}
		} else if !strings.HasPrefix(s.Name, p.PoolName+"/") || !validGUID(s.ParentGUID) || s.ParentGUID == s.GUID {
			return Policy{}, ErrStateRecord
		}
		seen := map[Property]bool{}
		for _, property := range s.Properties {
			if !validProperty(property) || seen[property] {
				return Policy{}, ErrStateRecord
			}
			seen[property] = true
		}
		s.Properties = append([]Property(nil), s.Properties...)
		sort.Slice(s.Properties, func(i, j int) bool { return s.Properties[i] < s.Properties[j] })
		canonical.Scopes[i], guids[s.GUID], names[s.Name] = s, s, true
	}
	for _, s := range canonical.Scopes {
		if parent, selected := guids[s.ParentGUID]; selected {
			slash := strings.LastIndex(s.Name, "/")
			if slash < 0 || parent.Name != s.Name[:slash] {
				return Policy{}, ErrStateRecord
			}
		}
	}
	sort.Slice(canonical.Scopes, func(i, j int) bool { return canonical.Scopes[i].GUID < canonical.Scopes[j].GUID })
	return canonical, nil
}

func ownershipWire(p Policy) ownershipPolicy {
	w := ownershipPolicy{Revision: p.Revision, PoolGUID: p.PoolGUID, PoolName: p.PoolName, Scopes: make([]ownershipScope, len(p.Scopes))}
	for i, s := range p.Scopes {
		w.Scopes[i] = ownershipScope{s.GUID, s.Name, s.ParentGUID, s.Properties}
	}
	return w
}
func ownershipChecksum(p ownershipPolicy) string {
	// The typed content uses only validated strings/uints/slices, so Marshal
	// cannot fail. Version and kind are part of the domain-bound damage checksum.
	b, _ := json.Marshal(ownershipContent{Version: 1, Kind: "ownership", Policy: p})
	h := sha256.Sum256(append([]byte("horvium:ownership:v1\n"), b...))
	return hex.EncodeToString(h[:])
}
