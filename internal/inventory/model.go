// SPDX-License-Identifier: BSD-3-Clause
// Package inventory models observations, never authority to mutate resources.
package inventory

import (
	"errors"
	"time"
)

var (
	ErrMalformed     = errors.New("incompatible inventory output")
	ErrLimit         = errors.New("inventory safety budget exceeded")
	ErrInvalidLimits = errors.New("invalid inventory limits")
)

// Limits bound source bytes, records, XML structure and field sizes. They are
// configurable safety budgets; reaching one is an error, not silent truncation.
type Limits struct {
	MaxBytes       int
	MaxRecords     int
	MaxFieldBytes  int
	MaxXMLDepth    int
	MaxXMLElements int
}

func DefaultLimits() Limits {
	return Limits{MaxBytes: 64 << 20, MaxRecords: 100000, MaxFieldBytes: 4096, MaxXMLDepth: 32, MaxXMLElements: 250000}
}

func (l Limits) Validate() error {
	if l.MaxBytes < 1 || l.MaxRecords < 1 || l.MaxFieldBytes < 1 || l.MaxXMLDepth < 1 || l.MaxXMLElements < 1 {
		return ErrInvalidLimits
	}
	return nil
}

type Pool struct {
	Name      string `json:"name"`
	GUID      string `json:"guid"`
	Size      uint64 `json:"size_bytes"`
	Allocated uint64 `json:"allocated_bytes"`
	Free      uint64 `json:"free_bytes"`
	Health    string `json:"health"`
}

type Dataset struct {
	Name       string `json:"name"`
	GUID       string `json:"guid"`
	Used       uint64 `json:"used_bytes"`
	Available  uint64 `json:"available_bytes"`
	Referenced uint64 `json:"referenced_bytes"`
}

type ProviderKind string

const (
	DiskCandidate ProviderKind = "disk_candidate"
	Partition     ProviderKind = "partition"
	Memory        ProviderKind = "memory"
	Logical       ProviderKind = "logical"
)

type IdentityState string

const (
	IdentityUnique        IdentityState = "unique_in_collection"
	IdentityMissing       IdentityState = "missing"
	IdentityAmbiguous     IdentityState = "ambiguous"
	IdentityNotApplicable IdentityState = "not_applicable"
)

type Identifier struct {
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	Source string `json:"source"`
}

type Class struct {
	ID   string `json:"local_id"`
	Name string `json:"name"`
}

type Consumer struct {
	ID         string `json:"local_id"`
	ProviderID string `json:"provider_local_id,omitempty"`
}

type Geom struct {
	ID        string     `json:"local_id"`
	ClassID   string     `json:"class_local_id"`
	Name      string     `json:"name"`
	Consumers []Consumer `json:"consumers"`
}

type Provider struct {
	ID          string        `json:"local_id"`
	GeomID      string        `json:"geom_local_id"`
	Name        string        `json:"name"`
	Class       string        `json:"class"`
	Kind        ProviderKind  `json:"kind"`
	MediaSize   uint64        `json:"media_size_bytes"`
	SectorSize  uint64        `json:"sector_size_bytes"`
	Identifiers []Identifier  `json:"identifiers"`
	Identity    IdentityState `json:"identity"`
}

// Graph IDs are valid only inside this observation; no durable disk ID is chosen.
type GEOM struct {
	Classes   []Class    `json:"classes"`
	Geoms     []Geom     `json:"geoms"`
	Providers []Provider `json:"providers"`
}

type Status string

const (
	OK          Status = "ok"
	Partial     Status = "partial"
	Unavailable Status = "unavailable"
)

// Problem carries a safe source/code, never command stderr or observed identifiers.
type Problem struct {
	Source string `json:"source"`
	Code   string `json:"code"`
}

type Section[T any] struct {
	Status     Status    `json:"status"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Data       T         `json:"data"`
	Problems   []Problem `json:"problems,omitempty"`
}

type Host struct {
	InstalledKernel string  `json:"installed_kernel,omitempty"`
	RunningKernel   string  `json:"running_kernel,omitempty"`
	Userland        string  `json:"userland,omitempty"`
	Architecture    string  `json:"architecture,omitempty"`
	PhysicalMemory  *uint64 `json:"physical_memory_bytes,omitempty"`
	ZFSUserland     string  `json:"zfs_userland,omitempty"`
	ZFSKernel       string  `json:"zfs_kernel,omitempty"`
}

type Snapshot struct {
	StartedAt  time.Time          `json:"started_at"`
	FinishedAt time.Time          `json:"finished_at"`
	Host       Section[Host]      `json:"host"`
	GEOM       Section[GEOM]      `json:"geom"`
	Pools      Section[[]Pool]    `json:"pools"`
	Datasets   Section[[]Dataset] `json:"datasets"`
}

func (s Snapshot) Complete() bool {
	return s.Host.Status == OK && s.GEOM.Status == OK && s.Pools.Status == OK && s.Datasets.Status == OK
}
