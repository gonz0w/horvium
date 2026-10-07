// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"reflect"
	"sort"
	"strings"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

// PoolProviderSource independently correlates every direct pool leaf with its
// complete GEOM graph, diskinfo geometry/serial and qualified attachments. It
// provides observations only: no locality, enrollment or execution authority.
type PoolProviderSource struct{ source *SystemUseSource }

func NewPoolProviderSource() (*PoolProviderSource, error) {
	s, err := NewSystemUseSource()
	if err != nil {
		return nil, err
	}
	return &PoolProviderSource{source: s}, nil
}

// directProviderName deliberately excludes partition/label paths and options.
// Addresses are accepted only from the already validated native topology and
// must also match one direct uniquely identified DISK in the fresh whole graph.
func directProviderName(address string) (string, bool) {
	n, ok := strings.CutPrefix(address, "/dev/")
	if !ok || n == "" || n == "." || n == ".." || n[0] == '-' {
		return "", false
	}
	for _, c := range n {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.') {
			return "", false
		}
	}
	return n, true
}

func (s *PoolProviderSource) Collect(ctx context.Context, o storage.Observation) ([]inventory.DiskObservation, error) {
	if ctx == nil {
		return nil, storage.ErrInvalid
	}
	if err := storage.ValidatePoolSelection(o.PoolGUID, o.PoolName); err != nil {
		return nil, err
	}
	if !o.Complete {
		return nil, storage.ErrUnavailable
	}
	if err := storage.VerifyPoolTopology(o.PoolGUID, o.PoolName, o.Topology); err != nil {
		return nil, err
	}
	var names []string
	for _, v := range o.Topology.Nodes {
		if v.Kind == "leaf" {
			n, ok := directProviderName(v.Address)
			if !ok {
				return nil, inventory.ErrDiskCorrelation
			}
			names = append(names, n)
		}
	}
	if s == nil || s.source == nil || s.source.read == nil || s.source.commandTimeout <= 0 || s.source.totalTimeout <= 0 {
		return nil, ErrUnavailable
	}
	r := s.source
	if r.bytes <= 0 || r.bytes > 64<<20 || r.records < 8 || r.records > 100000 {
		return nil, ErrLimit
	}
	sort.Strings(names)
	ctx, cancel := context.WithTimeout(ctx, r.totalTimeout)
	defer cancel()
	c := systemCycle{source: r, bytes: r.bytes, lines: r.records}
	before, release, err := providerPass(ctx, &c, o, names)
	if err != nil {
		return nil, err
	}
	after, next, err := providerPass(ctx, &c, o, names)
	if err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if release != next || !reflect.DeepEqual(before, after) {
		return nil, storage.ErrDrift
	}
	return after, nil
}

func providerPass(ctx context.Context, c *systemCycle, o storage.Observation, names []string) ([]inventory.DiskObservation, string, error) {
	rows := map[systemProbe][]byte{}
	for _, p := range []systemProbe{systemVersions, systemPoolTopologyGUID, systemPoolTopologyPath, systemGEOM} {
		data, err := c.capture(ctx, p, o.PoolName)
		if err != nil {
			return nil, "", err
		}
		rows[p] = data
	}
	versions, err := sourceLines(rows[systemVersions], 3, inventory.DefaultLimits())
	if err != nil {
		return nil, "", err
	}
	format := ""
	for _, v := range versions {
		m := propertyReleasePattern.FindStringSubmatch(v)
		if len(m) != 2 || format != "" && format != m[1] {
			return nil, "", ErrUnavailable
		}
		format = m[1]
	}
	topology, err := storage.ParsePoolTopology(rows[systemPoolTopologyGUID], rows[systemPoolTopologyPath], o.PoolName, o.PoolGUID)
	if err != nil {
		return nil, "", err
	}
	if !reflect.DeepEqual(topology, o.Topology) {
		return nil, "", storage.ErrDrift
	}
	// Bound all repeated complete-graph parses together, across both passes.
	// One initial parse plus two parses per leaf receive disjoint allowances.
	limits := inventory.DefaultLimits()
	limits.MaxBytes = c.source.bytes / (2 * (1 + 2*len(names)))
	limits.MaxRecords = c.source.records / (2 * (1 + 2*len(names)))
	limits.MaxXMLElements /= 2 * (1 + 2*len(names))
	if limits.MaxRecords < 1 {
		return nil, "", ErrLimit
	}
	graph, err := inventory.ParseGEOM(rows[systemGEOM], limits)
	if err != nil {
		return nil, "", err
	}
	providers := map[string]inventory.Provider{}
	for _, p := range graph.Providers {
		if _, ok := providers[p.Name]; ok {
			return nil, "", inventory.ErrDiskCorrelation
		}
		providers[p.Name] = p
	}
	result := make([]inventory.DiskObservation, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		p, ok := providers[name]
		if !ok || seen[p.ID] || p.Kind != inventory.DiskCandidate || p.Class != "DISK" || p.Identity != inventory.IdentityUnique {
			return nil, "", inventory.ErrDiskCorrelation
		}
		seen[p.ID] = true
		info, e := c.capture(ctx, systemDiskInfo, name)
		if e != nil {
			return nil, "", e
		}
		serial, e := c.capture(ctx, systemDiskSerial, name)
		if e != nil {
			return nil, "", e
		}
		disk, e := inventory.ParseDiskObservation(rows[systemGEOM], info, serial, limits)
		if e != nil {
			return nil, "", e
		}
		if !reflect.DeepEqual(disk.Provider, p) {
			return nil, "", inventory.ErrDiskCorrelation
		}
		attachments, e := inventory.ParseDiskAttachments(rows[systemGEOM], name, limits)
		if e != nil {
			return nil, "", e
		}
		if !reflect.DeepEqual(attachments.Disk, p) {
			return nil, "", inventory.ErrDiskAttachments
		}
		result = append(result, disk)
	}
	return result, string(rows[systemVersions]), nil
}
