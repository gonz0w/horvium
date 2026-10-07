// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

type systemProbe string

const (
	systemVersions         systemProbe = "versions"
	systemGEOM             systemProbe = "geom"
	systemMounts           systemProbe = "mounts"
	systemSwap             systemProbe = "swap"
	systemPools            systemProbe = "pools"
	systemPoolGUID         systemProbe = "pool-guid"
	systemPoolTopologyGUID systemProbe = "topology-guid"
	systemPoolTopologyPath systemProbe = "topology-path"
)

type systemReader func(context.Context, systemProbe, string, int) (readOutput, error)

// SystemUseSource collects every imported pool and mount/swap backing using
// fixed guarded reads. It has no request selectors, unused-device declaration,
// enrollment authority or mutation methods.
type SystemUseSource struct {
	read                         systemReader
	bytes, records               int
	commandTimeout, totalTimeout time.Duration
}

func NewSystemUseSource() (*SystemUseSource, error) {
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	return &SystemUseSource{read: readSystemNative, bytes: 64 << 20, records: 100000, commandTimeout: 10 * time.Second, totalTimeout: 45 * time.Second}, nil
}

type systemCycle struct {
	source       *SystemUseSource
	bytes, lines int
}

func (c *systemCycle) capture(ctx context.Context, p systemProbe, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.bytes <= 0 || c.lines <= 0 {
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
		return nil, err
	}
	if err = child.Err(); err != nil {
		return nil, err
	}
	lines := strings.Count(string(out.data), "\n")
	if lines > c.lines {
		return nil, ErrLimit
	}
	c.lines -= lines
	return out.data, nil
}
func (s *SystemUseSource) Collect(ctx context.Context) (storage.PoolSystemUse, error) {
	if ctx == nil {
		return storage.PoolSystemUse{}, storage.ErrInvalid
	}
	if s == nil || s.read == nil || s.commandTimeout <= 0 || s.totalTimeout <= 0 {
		return storage.PoolSystemUse{}, ErrUnavailable
	}
	if s.bytes <= 0 || s.bytes > 64<<20 || s.records < 8 || s.records > 100000 {
		return storage.PoolSystemUse{}, ErrLimit
	}
	ctx, cancel := context.WithTimeout(ctx, s.totalTimeout)
	defer cancel()
	c := systemCycle{source: s, bytes: s.bytes, lines: s.records}
	before, release, err := c.pass(ctx)
	if err != nil {
		return storage.PoolSystemUse{}, err
	}
	after, nextRelease, err := c.pass(ctx)
	if err != nil {
		return storage.PoolSystemUse{}, err
	}
	if err = ctx.Err(); err != nil {
		return storage.PoolSystemUse{}, err
	}
	if release != nextRelease || !reflect.DeepEqual(before, after) {
		return storage.PoolSystemUse{}, storage.ErrDrift
	}
	return after, nil
}
func (c *systemCycle) pass(ctx context.Context) (storage.PoolSystemUse, string, error) {
	rows := map[systemProbe][]byte{}
	for _, p := range []systemProbe{systemVersions, systemGEOM, systemMounts, systemSwap, systemPools} {
		data, err := c.capture(ctx, p, "")
		if err != nil {
			return storage.PoolSystemUse{}, "", err
		}
		rows[p] = data
	}
	versions, err := sourceLines(rows[systemVersions], 3, inventory.DefaultLimits())
	if err != nil {
		return storage.PoolSystemUse{}, "", err
	}
	format := ""
	for _, v := range versions {
		m := propertyReleasePattern.FindStringSubmatch(v)
		if len(m) != 2 || format != "" && format != m[1] {
			return storage.PoolSystemUse{}, "", ErrUnavailable
		}
		format = m[1]
	}
	limits := inventory.DefaultLimits()
	limits.MaxBytes = c.source.bytes
	limits.MaxRecords = c.source.records / 2
	catalog, err := inventory.ParsePools(rows[systemPools], limits)
	if err != nil {
		return storage.PoolSystemUse{}, "", err
	}
	if len(catalog) > limits.MaxRecords/4 {
		return storage.PoolSystemUse{}, "", ErrLimit
	}
	var captures []storage.PoolUseCapture
	for _, pool := range catalog {
		if err = storage.ValidatePoolSelection(pool.GUID, pool.Name); err != nil {
			return storage.PoolSystemUse{}, "", err
		}
		capture := storage.PoolUseCapture{Name: pool.Name}
		for _, entry := range []struct {
			probe  systemProbe
			target *[]byte
		}{{systemPoolGUID, &capture.GUID}, {systemPoolTopologyGUID, &capture.StatusGUIDs}, {systemPoolTopologyPath, &capture.StatusPaths}} {
			data, e := c.capture(ctx, entry.probe, pool.Name)
			if e != nil {
				return storage.PoolSystemUse{}, "", e
			}
			*entry.target = data
		}
		captures = append(captures, capture)
	}
	use, err := storage.ParsePoolSystemUse(rows[systemGEOM], rows[systemMounts], rows[systemSwap], rows[systemPools], captures, limits)
	return use, string(rows[systemVersions]), err
}
