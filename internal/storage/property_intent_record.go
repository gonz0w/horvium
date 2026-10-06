// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"reflect"
	"time"
	"unicode/utf8"
)

// EncodePropertyIntent returns bounded canonical baseline metadata only. No
// StateDirectory publisher accepts this kind; future job embedding/admission and
// root provenance need their own reviewed format and qualification.
func EncodePropertyIntent(i PropertyIntent) ([]byte, error) {
	normalized, err := normalizePropertyIntent(i)
	if err != nil {
		return nil, err
	}
	return encodePropertyIntent(normalized)
}

func encodePropertyIntent(i PropertyIntent) ([]byte, error) {
	w := propertyIntentToWire(i)
	data, err := json.Marshal(propertyIntentRecord{Version: 1, Kind: "property-intent", Intent: w, Checksum: propertyIntentChecksum(w)})
	if err != nil {
		return nil, ErrStateRecord
	}
	if len(data) > maxStateRecordBytes {
		return nil, ErrLimit
	}
	return data, nil
}

// DecodePropertyIntent returns independent consistency-checked metadata. It
// cannot restore a Plan/session, prove a native baseline was complete or authorize
// work. Fixed-shape traversal rejects unknown/duplicate/missing/null fields.
func DecodePropertyIntent(data []byte) (PropertyIntent, error) {
	if len(data) > maxStateRecordBytes {
		return PropertyIntent{}, ErrLimit
	}
	if !utf8.Valid(data) {
		return PropertyIntent{}, ErrStateRecord
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if ownershipJSONValue(d, reflect.TypeFor[propertyIntentRecord]()) != nil {
		return PropertyIntent{}, ErrStateRecord
	}
	if _, err := d.Token(); err != io.EOF {
		return PropertyIntent{}, ErrStateRecord
	}
	var w propertyIntentRecord
	if json.Unmarshal(data, &w) != nil || w.Version != 1 || w.Kind != "property-intent" {
		return PropertyIntent{}, ErrStateRecord
	}
	decoded, err := propertyIntentFromWire(w.Intent)
	if err != nil {
		return PropertyIntent{}, err
	}
	normalized, err := normalizePropertyIntent(decoded)
	if err != nil {
		return PropertyIntent{}, err
	}
	if !validJobDigest(w.Checksum) || w.Checksum != propertyIntentChecksum(propertyIntentToWire(normalized)) {
		return PropertyIntent{}, ErrStateRecord
	}
	return normalized, nil
}

func propertyIntentChecksum(w propertyIntentWire) string {
	data, _ := json.Marshal(propertyIntentContent{Version: 1, Kind: "property-intent", Intent: w})
	sum := sha256.Sum256(append([]byte("horvium:property-intent:v1\n"), data...))
	return hex.EncodeToString(sum[:])
}

func propertyValueToWire(v Value) propertyValueWire {
	return propertyValueWire{v.Effective, v.Source, v.SourceGUID, v.ReceivedValue}
}
func propertyValueFromWire(w propertyValueWire) Value {
	return Value{w.Effective, w.Source, w.SourceGUID, w.ReceivedValue}
}
func propertyPoolValueToWire(v PoolValue) propertyPoolValueWire {
	return propertyPoolValueWire{v.Effective, v.Source}
}
func propertyPoolValueFromWire(w propertyPoolValueWire) PoolValue {
	return PoolValue{w.Effective, w.Source}
}

func propertyIntentToWire(i PropertyIntent) propertyIntentWire {
	r := i.review
	w := propertyReviewWire{ID: r.ID, CreatedAt: r.CreatedAt.Format(time.RFC3339Nano), ExpiresAt: r.ExpiresAt.Format(time.RFC3339Nano), PolicyRevision: r.PolicyRevision, PoolGUID: r.PoolGUID, PoolName: r.PoolName, Change: jobChange{r.Intent.FilesystemGUID, r.Intent.Property, r.Intent.Value}, Context: make([]propertyFilesystemWire, len(r.Context)), Affected: make([]propertyResourceWire, len(r.Affected)), Versions: make([]propertyVersionWire, len(r.Versions)), States: make([]propertyStateWire, len(r.States))}
	for n, f := range r.Context {
		w.Context[n] = propertyFilesystemWire{f.GUID, f.Name, f.ParentGUID, propertyPropertiesWire{propertyValueToWire(f.Properties.Quota), propertyValueToWire(f.Properties.RefQuota), propertyValueToWire(f.Properties.Compression)}}
	}
	for n, a := range r.Affected {
		w.Affected[n] = propertyResourceWire{a.GUID, a.Name}
	}
	for n, v := range r.Versions {
		w.Versions[n] = propertyVersionWire{v.GUID, v.Name, v.Type, v.OwnerGUID}
	}
	for n, s := range r.States {
		w.States[n] = propertyStateWire{GUID: s.GUID, Name: s.Name, EncryptionRootGUID: s.EncryptionRootGUID, Mountpoint: propertyValueToWire(s.Mountpoint), CanMount: propertyValueToWire(s.CanMount), Readonly: propertyValueToWire(s.Readonly), Mounted: propertyValueToWire(s.Mounted), Encryption: propertyValueToWire(s.Encryption), EncryptionRoot: propertyValueToWire(s.EncryptionRoot), KeyStatus: propertyValueToWire(s.KeyStatus)}
	}
	p := r.Pool
	w.Pool = propertyPoolWire{GUID: p.GUID, LoadGUID: p.LoadGUID, Name: p.Name, GUIDSource: p.GUIDSource, LoadGUIDSource: p.LoadGUIDSource, Health: propertyPoolValueToWire(p.Health), Readonly: propertyPoolValueToWire(p.Readonly), BootFS: propertyPoolValueToWire(p.BootFS), AltRoot: propertyPoolValueToWire(p.AltRoot), CacheFile: propertyPoolValueToWire(p.CacheFile), Compatibility: propertyPoolValueToWire(p.Compatibility), MultiHost: propertyPoolValueToWire(p.MultiHost)}
	w.Features = propertyFeaturesWire{PoolGUID: r.Features.PoolGUID, PoolName: r.Features.PoolName, Format: r.Features.Format, Values: make([]propertyFeatureWire, len(r.Features.Values))}
	for n, v := range r.Features.Values {
		w.Features.Values[n] = propertyFeatureWire{v.Name, v.State, v.Source}
	}
	topo := r.Topology
	w.Topology = propertyTopologyWire{ScanReported: topo.ScanReported, PoolGUID: topo.PoolGUID, PoolName: topo.PoolName, Health: topo.Health, Scan: topo.Scan, Diagnostics: topo.Diagnostics, ReadErrors: topo.ReadErrors, WriteErrors: topo.WriteErrors, ChecksumErrors: topo.ChecksumErrors, DataErrors: topo.DataErrors, Nodes: make([]propertyVdevWire, len(topo.Nodes))}
	for n, v := range topo.Nodes {
		w.Topology.Nodes[n] = propertyVdevWire{v.GUID, v.ParentGUID, v.Address, v.Kind, v.Health, v.Annotation, uint64(v.Ordinal), v.ReadErrors, v.WriteErrors, v.ChecksumErrors}
	}
	return propertyIntentWire{CapturedAt: i.capturedAt.Format(time.RFC3339Nano), PolicyDigest: propertyDigestString(i.policyDigest), SessionDigest: propertyDigestString(i.sessionDigest), Review: w}
}

func propertyIntentTimeFromWire(value string) (time.Time, error) {
	at, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || !canonicalIntentTime(at) || at.UTC().Format(time.RFC3339Nano) != value {
		return time.Time{}, ErrStateRecord
	}
	return at.UTC(), nil
}

func propertyIntentFromWire(w propertyIntentWire) (PropertyIntent, error) {
	var i PropertyIntent
	if !validJobDigest(w.PolicyDigest) || !validJobDigest(w.SessionDigest) {
		return i, ErrStateRecord
	}
	pd, _ := hex.DecodeString(w.PolicyDigest)
	sd, _ := hex.DecodeString(w.SessionDigest)
	copy(i.policyDigest[:], pd)
	copy(i.sessionDigest[:], sd)
	var err error
	if i.capturedAt, err = propertyIntentTimeFromWire(w.CapturedAt); err != nil {
		return PropertyIntent{}, err
	}
	r := w.Review
	created, err := propertyIntentTimeFromWire(r.CreatedAt)
	if err != nil {
		return PropertyIntent{}, err
	}
	expires, err := propertyIntentTimeFromWire(r.ExpiresAt)
	if err != nil {
		return PropertyIntent{}, err
	}
	if len(r.Context) > maxRecords || len(r.Affected) > maxRecords || len(r.Versions) > maxRecords || len(r.States) > maxRecords || len(r.Features.Values) > maxFeatures || len(r.Topology.Nodes) > maxVdevs {
		return PropertyIntent{}, ErrLimit
	}
	i.review = Review{ID: r.ID, CreatedAt: created, ExpiresAt: expires, PolicyRevision: r.PolicyRevision, PoolGUID: r.PoolGUID, PoolName: r.PoolName, Intent: Intent{r.Change.FilesystemGUID, r.Change.Property, r.Change.Value}, Context: make([]Filesystem, len(r.Context)), Affected: make([]Resource, len(r.Affected)), Versions: make([]DatasetIdentity, len(r.Versions)), States: make([]FilesystemState, len(r.States))}
	for n, f := range r.Context {
		i.review.Context[n] = Filesystem{f.GUID, f.Name, f.ParentGUID, Properties{propertyValueFromWire(f.Properties.Quota), propertyValueFromWire(f.Properties.RefQuota), propertyValueFromWire(f.Properties.Compression)}}
	}
	for n, a := range r.Affected {
		i.review.Affected[n] = Resource{a.GUID, a.Name}
	}
	for n, v := range r.Versions {
		i.review.Versions[n] = DatasetIdentity{v.GUID, v.Name, v.Type, v.OwnerGUID}
	}
	for n, s := range r.States {
		i.review.States[n] = FilesystemState{GUID: s.GUID, Name: s.Name, EncryptionRootGUID: s.EncryptionRootGUID, Mountpoint: propertyValueFromWire(s.Mountpoint), CanMount: propertyValueFromWire(s.CanMount), Readonly: propertyValueFromWire(s.Readonly), Mounted: propertyValueFromWire(s.Mounted), Encryption: propertyValueFromWire(s.Encryption), EncryptionRoot: propertyValueFromWire(s.EncryptionRoot), KeyStatus: propertyValueFromWire(s.KeyStatus)}
	}
	p := r.Pool
	i.review.Pool = PoolState{GUID: p.GUID, LoadGUID: p.LoadGUID, Name: p.Name, GUIDSource: p.GUIDSource, LoadGUIDSource: p.LoadGUIDSource, Health: propertyPoolValueFromWire(p.Health), Readonly: propertyPoolValueFromWire(p.Readonly), BootFS: propertyPoolValueFromWire(p.BootFS), AltRoot: propertyPoolValueFromWire(p.AltRoot), CacheFile: propertyPoolValueFromWire(p.CacheFile), Compatibility: propertyPoolValueFromWire(p.Compatibility), MultiHost: propertyPoolValueFromWire(p.MultiHost)}
	i.review.Features = PoolFeatures{PoolGUID: r.Features.PoolGUID, PoolName: r.Features.PoolName, Format: r.Features.Format, Values: make([]PoolFeature, len(r.Features.Values))}
	for n, v := range r.Features.Values {
		i.review.Features.Values[n] = PoolFeature{v.Name, v.State, v.Source}
	}
	topo := r.Topology
	i.review.Topology = PoolTopology{ScanReported: topo.ScanReported, PoolGUID: topo.PoolGUID, PoolName: topo.PoolName, Health: topo.Health, Scan: topo.Scan, Diagnostics: topo.Diagnostics, ReadErrors: topo.ReadErrors, WriteErrors: topo.WriteErrors, ChecksumErrors: topo.ChecksumErrors, DataErrors: topo.DataErrors, Nodes: make([]Vdev, len(topo.Nodes))}
	for n, v := range topo.Nodes {
		if v.Ordinal >= maxVdevs {
			return PropertyIntent{}, ErrStateRecord
		}
		i.review.Topology.Nodes[n] = Vdev{v.GUID, v.ParentGUID, v.Address, v.Kind, v.Health, v.Annotation, int(v.Ordinal), v.ReadErrors, v.WriteErrors, v.ChecksumErrors}
	}
	return i, nil
}
