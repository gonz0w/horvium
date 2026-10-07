// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type ownershipMember struct {
	VdevGUID     string `json:"vdev_guid"`
	Serial       string `json:"serial"`
	SerialSource string `json:"serial_source"`
	MediaSize    uint64 `json:"media_size"`
	SectorSize   uint64 `json:"sector_size"`
	StripeSize   uint64 `json:"stripe_size"`
	StripeOffset uint64 `json:"stripe_offset"`
}
type ownershipPolicyV2 struct {
	Revision uint64            `json:"revision"`
	PoolGUID string            `json:"pool_guid"`
	PoolName string            `json:"pool_name"`
	Scopes   []ownershipScope  `json:"scopes"`
	Members  []ownershipMember `json:"members"`
}
type ownershipContentV2 struct {
	Version uint64            `json:"version"`
	Kind    string            `json:"kind"`
	Policy  ownershipPolicyV2 `json:"policy"`
}
type ownershipRecordV2 struct {
	Version  uint64            `json:"version"`
	Kind     string            `json:"kind"`
	Policy   ownershipPolicyV2 `json:"policy"`
	Checksum string            `json:"checksum"`
}

func normalizeMembers(members []PoolMember) ([]PoolMember, error) {
	if len(members) > maxVdevs {
		return nil, ErrLimit
	}
	remaining := maxStateRecordBytes - 256
	for _, m := range members {
		remaining -= 256
		for _, s := range []string{m.VdevGUID, m.Serial, m.SerialSource} {
			if len(s) > maxFieldBytes || len(s) > remaining {
				return nil, ErrLimit
			}
			remaining -= len(s)
		}
		if remaining < 0 {
			return nil, ErrLimit
		}
	}
	if len(members) == 0 {
		return nil, nil
	}
	result := append([]PoolMember(nil), members...)
	guids, serials := map[string]bool{}, map[string]bool{}
	for _, m := range result {
		if !validGUID(m.VdevGUID) || guids[m.VdevGUID] || m.Serial == "" || serials[m.Serial] || !utf8.ValidString(m.Serial) || strings.TrimSpace(m.Serial) != m.Serial || strings.IndexFunc(m.Serial, unicode.IsControl) >= 0 {
			return nil, ErrStateRecord
		}
		if m.SerialSource != "geom.config.ident" && m.SerialSource != "provider.config.ident" {
			return nil, ErrStateRecord
		}
		if m.SectorSize == 0 || m.SectorSize > math.MaxUint32 || m.MediaSize == 0 || m.MediaSize > math.MaxInt64 || m.MediaSize%m.SectorSize != 0 || m.StripeSize > math.MaxInt64 || m.StripeOffset > math.MaxInt64 {
			return nil, ErrStateRecord
		}
		guids[m.VdevGUID], serials[m.Serial] = true, true
	}
	sort.Slice(result, func(i, j int) bool { return result[i].VdevGUID < result[j].VdevGUID })
	return result, nil
}
func ownershipWireV2(p Policy) ownershipPolicyV2 {
	v1 := ownershipWire(p)
	w := ownershipPolicyV2{Revision: v1.Revision, PoolGUID: v1.PoolGUID, PoolName: v1.PoolName, Scopes: v1.Scopes, Members: make([]ownershipMember, len(p.Members))}
	for i, m := range p.Members {
		w.Members[i] = ownershipMember{m.VdevGUID, m.Serial, m.SerialSource, m.MediaSize, m.SectorSize, m.StripeSize, m.StripeOffset}
	}
	return w
}
func ownershipChecksumV2(p ownershipPolicyV2) string {
	b, _ := json.Marshal(ownershipContentV2{2, "ownership", p})
	h := sha256.Sum256(append([]byte("horvium:ownership:v2\n"), b...))
	return hex.EncodeToString(h[:])
}
func encodeOwnershipV2(p Policy) ([]byte, error) {
	w := ownershipWireV2(p)
	b, err := json.Marshal(ownershipRecordV2{2, "ownership", w, ownershipChecksumV2(w)})
	if err != nil {
		return nil, ErrStateRecord
	}
	if len(b) > maxStateRecordBytes {
		return nil, ErrLimit
	}
	return b, nil
}
func decodeOwnershipV2(data []byte) (Policy, error) {
	if !ownershipShape(data, ownershipRecordV2{}) {
		return Policy{}, ErrStateRecord
	}
	var r ownershipRecordV2
	if json.Unmarshal(data, &r) != nil || r.Version != 2 || r.Kind != "ownership" || len(r.Policy.Members) == 0 {
		return Policy{}, ErrStateRecord
	}
	p := Policy{Revision: r.Policy.Revision, PoolGUID: r.Policy.PoolGUID, PoolName: r.Policy.PoolName, Scopes: make([]Scope, len(r.Policy.Scopes)), Members: make([]PoolMember, len(r.Policy.Members))}
	for i, s := range r.Policy.Scopes {
		p.Scopes[i] = Scope{s.GUID, s.Name, s.ParentGUID, s.Properties}
	}
	for i, m := range r.Policy.Members {
		p.Members[i] = PoolMember{m.VdevGUID, m.Serial, m.SerialSource, m.MediaSize, m.SectorSize, m.StripeSize, m.StripeOffset}
	}
	p, err := normalizeOwnership(p)
	if err != nil {
		return Policy{}, err
	}
	if len(r.Checksum) != 64 || r.Checksum != ownershipChecksumV2(ownershipWireV2(p)) {
		return Policy{}, ErrStateRecord
	}
	return p, nil
}
