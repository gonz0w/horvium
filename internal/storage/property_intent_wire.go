// SPDX-License-Identifier: BSD-3-Clause
package storage

// Required, explicit version-one fields. Planning Go field names are not the schema.
type propertyValueWire struct {
	Effective     string `json:"effective"`
	Source        Source `json:"source"`
	SourceGUID    string `json:"source_guid"`
	ReceivedValue string `json:"received_value"`
}
type propertyPropertiesWire struct {
	Quota       propertyValueWire `json:"quota"`
	RefQuota    propertyValueWire `json:"refquota"`
	Compression propertyValueWire `json:"compression"`
}
type propertyFilesystemWire struct {
	GUID       string                 `json:"guid"`
	Name       string                 `json:"name"`
	ParentGUID string                 `json:"parent_guid"`
	Properties propertyPropertiesWire `json:"properties"`
}
type propertyStateWire struct {
	GUID               string            `json:"guid"`
	Name               string            `json:"name"`
	EncryptionRootGUID string            `json:"encryption_root_guid"`
	Mountpoint         propertyValueWire `json:"mountpoint"`
	CanMount           propertyValueWire `json:"canmount"`
	Readonly           propertyValueWire `json:"readonly"`
	Mounted            propertyValueWire `json:"mounted"`
	Encryption         propertyValueWire `json:"encryption"`
	EncryptionRoot     propertyValueWire `json:"encryption_root"`
	KeyStatus          propertyValueWire `json:"keystatus"`
}
type propertyResourceWire struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
}
type propertyVersionWire struct {
	GUID      string      `json:"guid"`
	Name      string      `json:"name"`
	Type      DatasetType `json:"type"`
	OwnerGUID string      `json:"owner_guid"`
}
type propertyPoolValueWire struct {
	Effective string `json:"effective"`
	Source    Source `json:"source"`
}
type propertyPoolWire struct {
	GUID           string                `json:"guid"`
	LoadGUID       string                `json:"load_guid"`
	Name           string                `json:"name"`
	GUIDSource     Source                `json:"guid_source"`
	LoadGUIDSource Source                `json:"load_guid_source"`
	Health         propertyPoolValueWire `json:"health"`
	Readonly       propertyPoolValueWire `json:"readonly"`
	BootFS         propertyPoolValueWire `json:"bootfs"`
	AltRoot        propertyPoolValueWire `json:"altroot"`
	CacheFile      propertyPoolValueWire `json:"cachefile"`
	Compatibility  propertyPoolValueWire `json:"compatibility"`
	MultiHost      propertyPoolValueWire `json:"multihost"`
}
type propertyFeatureWire struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Source Source `json:"source"`
}
type propertyFeaturesWire struct {
	PoolGUID string                `json:"pool_guid"`
	PoolName string                `json:"pool_name"`
	Format   string                `json:"format"`
	Values   []propertyFeatureWire `json:"values"`
}
type propertyVdevWire struct {
	GUID           string `json:"guid"`
	ParentGUID     string `json:"parent_guid"`
	Address        string `json:"address"`
	Kind           string `json:"kind"`
	Health         string `json:"health"`
	Annotation     string `json:"annotation"`
	Ordinal        uint64 `json:"ordinal"`
	ReadErrors     uint64 `json:"read_errors"`
	WriteErrors    uint64 `json:"write_errors"`
	ChecksumErrors uint64 `json:"checksum_errors"`
}
type propertyTopologyWire struct {
	ScanReported   bool               `json:"scan_reported"`
	PoolGUID       string             `json:"pool_guid"`
	PoolName       string             `json:"pool_name"`
	Health         string             `json:"health"`
	Scan           string             `json:"scan"`
	Diagnostics    string             `json:"diagnostics"`
	ReadErrors     uint64             `json:"read_errors"`
	WriteErrors    uint64             `json:"write_errors"`
	ChecksumErrors uint64             `json:"checksum_errors"`
	DataErrors     uint64             `json:"data_errors"`
	Nodes          []propertyVdevWire `json:"nodes"`
}
type propertyReviewWire struct {
	ID             string                   `json:"id"`
	CreatedAt      string                   `json:"created_at"`
	ExpiresAt      string                   `json:"expires_at"`
	PolicyRevision uint64                   `json:"policy_revision"`
	PoolGUID       string                   `json:"pool_guid"`
	PoolName       string                   `json:"pool_name"`
	Change         jobChange                `json:"change"`
	Context        []propertyFilesystemWire `json:"context"`
	Affected       []propertyResourceWire   `json:"affected"`
	Versions       []propertyVersionWire    `json:"versions"`
	States         []propertyStateWire      `json:"states"`
	Pool           propertyPoolWire         `json:"pool"`
	Features       propertyFeaturesWire     `json:"features"`
	Topology       propertyTopologyWire     `json:"topology"`
}
type propertyIntentWire struct {
	CapturedAt    string             `json:"captured_at"`
	PolicyDigest  string             `json:"policy_digest"`
	SessionDigest string             `json:"session_digest"`
	Review        propertyReviewWire `json:"review"`
}
type propertyIntentContent struct {
	Version uint64             `json:"version"`
	Kind    string             `json:"kind"`
	Intent  propertyIntentWire `json:"intent"`
}
type propertyIntentRecord struct {
	Version  uint64             `json:"version"`
	Kind     string             `json:"kind"`
	Intent   propertyIntentWire `json:"intent"`
	Checksum string             `json:"checksum"`
}
