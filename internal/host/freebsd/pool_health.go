// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/storage"
)

// PoolHealthObservation contains fresh reported facts, including unhealthy ones.
// It supplies neither write eligibility nor scan ownership/durable job outcomes.
type PoolHealthObservation struct {
	Pool     storage.PoolState
	Topology storage.PoolTopology
	Scan     storage.PoolScan
}

// PoolHealthSource reads fixed native projections. It accepts no native output,
// privileges, policy, job/store or executable intent from a caller.
type PoolHealthSource struct {
	read                         poolReader
	bytes                        int
	commandTimeout, totalTimeout time.Duration
}

func NewPoolHealthSource() (*PoolHealthSource, error) {
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	return &PoolHealthSource{read: readPoolNative, bytes: 4 << 20, commandTimeout: 10 * time.Second, totalTimeout: 60 * time.Second}, nil
}

func healthRelease(data []byte) error {
	values, err := sourceLines(data, 3, inventory.DefaultLimits())
	if err != nil {
		return err
	}
	format := ""
	for _, value := range values {
		m := propertyReleasePattern.FindStringSubmatch(value)
		if len(m) != 2 || format != "" && format != m[1] {
			return ErrUnavailable
		}
		format = m[1]
	}
	return nil
}

// Collect brackets paired GUID/path status with pool import identity and release.
// Any failed, unsupported, incomplete or drifting read discards the whole result.
// Equality is a sequential drift check, never an atomic snapshot or execution fence.
func (s *PoolHealthSource) Collect(ctx context.Context, guid, name string) (PoolHealthObservation, error) {
	zero := PoolHealthObservation{}
	if ctx == nil {
		return zero, storage.ErrInvalid
	}
	if err := storage.ValidatePoolSelection(guid, name); err != nil {
		return zero, err
	}
	if s == nil || s.read == nil || s.commandTimeout <= 0 || s.totalTimeout <= 0 {
		return zero, ErrUnavailable
	}
	if s.bytes <= 0 || s.bytes > 4<<20 {
		return zero, ErrLimit
	}
	ctx, cancel := context.WithTimeout(ctx, s.totalTimeout)
	defer cancel()
	// Reuse the native source's consumed stdout/stderr budget and child reaping path.
	c := poolCycle{source: &PoolPropertySource{read: s.read, commandTimeout: s.commandTimeout}, bytes: s.bytes}
	beforeRelease, err := c.capture(ctx, poolVersions, name)
	if err != nil {
		return zero, err
	}
	if err = healthRelease(beforeRelease); err != nil {
		return zero, err
	}
	beforeData, err := c.capture(ctx, poolState, name)
	if err != nil {
		return zero, err
	}
	before, err := storage.ParsePoolState(beforeData, name)
	if err != nil {
		return zero, err
	}
	if before.GUID != guid {
		return zero, storage.ErrDrift
	}
	guids, err := c.capture(ctx, poolTopologyGUID, name)
	if err != nil {
		return zero, err
	}
	paths, err := c.capture(ctx, poolTopologyPath, name)
	if err != nil {
		return zero, err
	}
	topology, err := storage.ParsePoolTopology(guids, paths, name, guid)
	if err != nil {
		return zero, err
	}
	// Qualified complete legacy status may omit scan history. Preserve absence
	// explicitly; never infer idle, quiescence or an attributable outcome.
	scan := storage.PoolScan{Kind: "unreported", State: "unreported"}
	if topology.ScanReported {
		scan, err = storage.ParsePoolScan(topology.Scan)
		if err != nil {
			return zero, err
		}
	}
	afterData, err := c.capture(ctx, poolState, name)
	if err != nil {
		return zero, err
	}
	after, err := storage.ParsePoolState(afterData, name)
	if err != nil {
		return zero, err
	}
	afterRelease, err := c.capture(ctx, poolVersions, name)
	if err != nil {
		return zero, err
	}
	if err = healthRelease(afterRelease); err != nil {
		return zero, err
	}
	if err = ctx.Err(); err != nil {
		return zero, err
	}
	if before != after || string(beforeRelease) != string(afterRelease) || after.Health.Effective != topology.Health {
		return zero, storage.ErrDrift
	}
	return PoolHealthObservation{Pool: after, Topology: topology, Scan: scan}, nil
}
