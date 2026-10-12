// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

type poolProbe string

const (
	poolVersions        poolProbe = "versions"
	poolState           poolProbe = "pool-state"
	poolIdentities      poolProbe = "identities"
	poolProperties      poolProbe = "properties"
	poolFilesystemState poolProbe = "filesystem-state"
	poolFeatures        poolProbe = "features"
	poolTopologyGUID    poolProbe = "topology-guid"
	poolTopologyPath    poolProbe = "topology-path"
)

var poolProbes = [...]poolProbe{poolVersions, poolState, poolIdentities, poolProperties, poolFilesystemState, poolFeatures, poolTopologyGUID, poolTopologyPath}

type poolReader func(context.Context, poolProbe, string, int) (readOutput, error)

// PoolPropertySource assembles fixed read-only projections for a trusted pool
// selection. It has no root policy, device/system proof, store or writer. Even
// complete supported results leave Observation.Eligible false; they cannot
// supply the PropertyBackend authority or an executable review on their own.
type PoolPropertySource struct {
	read                         poolReader
	bytes, records               int
	commandTimeout, totalTimeout time.Duration
}

func NewPoolPropertySource() (*PoolPropertySource, error) {
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	return &PoolPropertySource{read: readPoolNative, bytes: 64 << 20, records: 100000, commandTimeout: 10 * time.Second, totalTimeout: 45 * time.Second}, nil
}

type poolCycle struct {
	source         *PoolPropertySource
	bytes, records int
}

func (c *poolCycle) capture(ctx context.Context, p poolProbe, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, storage.AnnotateDeadlineReturn(nil, ctx, err, storage.DeadlinePoolCommand, storage.DeadlineGuard, 0)
	}
	if c.bytes <= 0 {
		return nil, ErrLimit
	}
	child, cancel := context.WithTimeout(ctx, c.source.commandTimeout)
	defer cancel()
	out, err := c.source.read(child, p, name, c.bytes)
	if out.consumed < len(out.data) || out.consumed > c.bytes {
		return nil, ErrLimit
	}
	c.bytes -= out.consumed
	if err != nil {
		return nil, storage.AnnotateDeadlineReturn(child, ctx, err, storage.DeadlinePoolCommand, storage.DeadlineCapture, c.source.commandTimeout)
	}
	if err = child.Err(); err != nil {
		return nil, storage.AnnotateDeadlineReturn(child, ctx, err, storage.DeadlinePoolCommand, storage.DeadlineCapture, c.source.commandTimeout)
	}
	return out.data, nil
}

var propertyReleasePattern = regexp.MustCompile(`^(14\.5|15\.1)-RELEASE(?:-p[0-9]+)?$`)

// Collect uses one shared stdout/stderr byte budget and record budget across
// two full independent passes. Any failed/unsupported/mixed/drifting result is
// discarded whole. Repeated equality detects ordinary drift; it is not an atomic
// ZFS snapshot, root fence or protection against changes between/after reads.
func (s *PoolPropertySource) Collect(ctx context.Context, guid, name string) (storage.Observation, error) {
	if ctx == nil {
		return storage.Observation{}, storage.ErrInvalid
	}
	if err := storage.ValidatePoolSelection(guid, name); err != nil {
		return storage.Observation{}, err
	}
	if s == nil || s.read == nil || s.commandTimeout <= 0 || s.totalTimeout <= 0 {
		return storage.Observation{}, ErrUnavailable
	}
	if s.bytes <= 0 || s.bytes > 64<<20 || s.records <= 0 || s.records > 100000 {
		return storage.Observation{}, ErrLimit
	}
	parent := ctx
	ctx, cancel := context.WithTimeout(ctx, s.totalTimeout)
	defer cancel()
	c := poolCycle{source: s, bytes: s.bytes, records: s.records}
	before, release, err := c.pass(ctx, guid, name)
	if err != nil {
		return storage.Observation{}, storage.AnnotateDeadlineReturn(ctx, parent, err, storage.DeadlinePoolCollector, storage.DeadlineCollection, s.totalTimeout)
	}
	after, nextRelease, err := c.pass(ctx, guid, name)
	if err != nil {
		return storage.Observation{}, storage.AnnotateDeadlineReturn(ctx, parent, err, storage.DeadlinePoolCollector, storage.DeadlineCollection, s.totalTimeout)
	}
	if err = ctx.Err(); err != nil {
		return storage.Observation{}, storage.AnnotateDeadlineReturn(ctx, parent, err, storage.DeadlinePoolCollector, storage.DeadlineCollection, s.totalTimeout)
	}
	if release != nextRelease || !reflect.DeepEqual(before, after) {
		return storage.Observation{}, storage.ErrDrift
	}
	after.Complete = true
	// Missing native root/system/device-use and root-policy gates must stay false.
	return after, nil
}

func (c *poolCycle) pass(ctx context.Context, guid, name string) (storage.Observation, string, error) {
	o := storage.Observation{PoolGUID: guid, PoolName: name}
	var release, format string
	var guidRows, pathRows []byte
	for _, p := range poolProbes {
		if c.records <= 0 {
			return storage.Observation{}, "", ErrLimit
		}
		data, err := c.capture(ctx, p, name)
		if err != nil {
			return storage.Observation{}, "", err
		}
		count := 0
		switch p {
		case poolVersions:
			values, e := sourceLines(data, 3, inventory.DefaultLimits())
			err = e
			if err == nil {
				for _, v := range values {
					m := propertyReleasePattern.FindStringSubmatch(v)
					if len(m) != 2 || format != "" && format != "freebsd-"+m[1] {
						err = ErrUnavailable
						break
					}
					format = "freebsd-" + m[1]
				}
				release = string(data)
			}
			count = 3
		case poolState:
			o.Pool, err = storage.ParsePoolState(data, name)
			if err == nil {
				err = storage.VerifyPoolState(guid, name, o.Pool)
			}
			count = 1
		case poolIdentities:
			o.Identities, err = storage.ParseIdentities(data, name)
			count = len(o.Identities)
		case poolProperties:
			o.Filesystems, err = storage.ParseProperties(data, name)
			count = len(o.Filesystems)
			if err == nil {
				err = storage.VerifyFilesystemIdentities(name, o.Filesystems, o.Identities)
			}
		case poolFilesystemState:
			o.States, err = storage.ParseFilesystemStates(data, name)
			count = len(o.States)
			if err == nil {
				err = storage.VerifyFilesystemStates(name, o.Identities, o.States)
			}
		case poolFeatures:
			o.Features, err = storage.ParsePoolFeatures(data, name, format)
			count = len(o.Features.Values)
			if err == nil {
				err = storage.VerifyPoolFeatures(guid, name, o.Features)
			}
		case poolTopologyGUID:
			guidRows = data
		case poolTopologyPath:
			pathRows = data
			o.Topology, err = storage.ParsePoolTopology(guidRows, pathRows, name, guid)
			count = len(o.Topology.Nodes)
			if err == nil {
				err = storage.VerifyPoolTopology(guid, name, o.Topology)
			}
		}
		if err != nil {
			return storage.Observation{}, "", err
		}
		if count > c.records {
			return storage.Observation{}, "", ErrLimit
		}
		c.records -= count
	}
	return o, strings.TrimSuffix(release, "\n"), nil
}
