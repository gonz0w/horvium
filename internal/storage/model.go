// SPDX-License-Identifier: BSD-3-Clause
// Package storage validates prospective management scope and property reviews.
// It has no executor, persistence, transport or connection to the M1 inventory.
package storage

import (
	"errors"
	"time"
)

var (
	ErrInvalid     = errors.New("invalid storage planning input")
	ErrDenied      = errors.New("storage scope denied")
	ErrUnavailable = errors.New("complete eligible storage observation required")
	ErrLimit       = errors.New("storage planning budget exceeded")
	ErrDrift       = errors.New("storage plan requires new review")
	ErrExpired     = errors.New("storage plan expired")
)

const (
	Quota       Property = "quota"
	RefQuota    Property = "refquota"
	Compression Property = "compression"
	Default     Source   = "default"
	Local       Source   = "local"
	Received    Source   = "received"
	Inherited   Source   = "inherited"
)

type Property string
type Source string

// Value describes an effective property and its native source. SourceGUID is the
// supplying filesystem for local/received/inherited values; empty for default.
// ReceivedValue retains a shadowed received property as well as an active one.
// This is an internal model, not a wire or persisted format.
type Value struct {
	Effective     string
	Source        Source
	SourceGUID    string
	ReceivedValue string // Empty when native received column is "-".
}
type Properties struct{ Quota, RefQuota, Compression Value }
type Filesystem struct {
	GUID, Name, ParentGUID string
	Properties             Properties
}

type DatasetType string

const (
	FilesystemType DatasetType = "filesystem"
	VolumeType     DatasetType = "volume"
	SnapshotType   DatasetType = "snapshot"
	BookmarkType   DatasetType = "bookmark"
)

// Version GUIDs can be replicated; OwnerGUID binds them to this local dataset.
// Observed versions never grant snapshot/bookmark mutation authority.
type DatasetIdentity struct {
	GUID, Name string
	Type       DatasetType
	OwnerGUID  string
}

// Observation must come from a fresh complete, bounded helper-side collection.
// Eligible means native system/encryption/topology/feature gates ALL passed.
// No implemented adapter supplies this proof yet. M1 inventory cannot supply it.
type Observation struct {
	Complete           bool
	PoolGUID, PoolName string
	Eligible           bool
	Filesystems        []Filesystem
	Identities         []DatasetIdentity
}

// Policy is trusted local enrollment, never request-supplied authority. GUIDs
// identify resources; names/parents are the explicitly reviewed drift baseline.
// Each scope grants only its listed properties, never implicit child ownership.
// File ownership/schema/enrollment and persistence are later slice gates.
type Policy struct {
	Revision           uint64
	PoolGUID, PoolName string
	Scopes             []Scope
}
type Scope struct {
	GUID, Name, ParentGUID string
	Properties             []Property
}
type Intent struct {
	FilesystemGUID string
	Property       Property
	Value          string
}
type Resource struct{ GUID, Name string }

// Review includes the full relevant tree and ancestors, with before values and
// sources. Context includes branches protected by an override, so removing that
// override or adding a foreign child still invalidates a previously built plan.
type Review struct {
	ID                   string
	CreatedAt, ExpiresAt time.Time
	PolicyRevision       uint64
	PoolGUID, PoolName   string
	Intent               Intent
	Context              []Filesystem
	Affected             []Resource
	Versions             []DatasetIdentity
}

// Plan contains only private state and returns independent review copies. It is
// not a submission/admission token: repeated validation can never execute work.
type Plan struct {
	review     Review
	session    string
	policyHash [32]byte
}
