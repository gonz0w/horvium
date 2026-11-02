// SPDX-License-Identifier: BSD-3-Clause
// Package freebsd contains native read-only discovery, to be called by the helper.
package freebsd

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

var (
	ErrUnsupported = errors.New("native discovery requires FreeBSD")
	ErrUnavailable = errors.New("native source unavailable")
	ErrPermission  = errors.New("native source permission denied")
	ErrExit        = errors.New("native source failed")
	ErrLimit       = inventory.ErrLimit
)

type probe string

const (
	versions     probe = "freebsd-version"
	architecture probe = "architecture"
	memory       probe = "physical-memory"
	zfsVersion   probe = "zfs-version"
	geom         probe = "geom"
	pools        probe = "pools"
	datasets     probe = "datasets"
)

type readOutput struct {
	data     []byte
	consumed int
}

type reader func(context.Context, probe, int) (readOutput, error)

type Source struct {
	limits         inventory.Limits
	read           reader
	commandTimeout time.Duration
	totalTimeout   time.Duration
}

func New(limits inventory.Limits) (*Source, error) {
	if err := limits.Validate(); err != nil {
		return nil, err
	}
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	return &Source{limits: limits, read: readNative, commandTimeout: 10 * time.Second, totalTimeout: 45 * time.Second}, nil
}

type cycle struct {
	source         *Source
	bytes, records int
	debt           error
}

func (c *cycle) read(ctx context.Context, p probe) ([]byte, error) {
	if c.debt != nil {
		return nil, c.debt
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.bytes <= 0 {
		return nil, ErrLimit
	}
	commandCtx, cancel := context.WithTimeout(ctx, c.source.commandTimeout)
	defer cancel()
	output, err := c.source.read(commandCtx, p, c.bytes)
	if errors.Is(err, ErrCleanupDebt) {
		c.debt = err
	}
	if output.consumed < len(output.data) || output.consumed > c.bytes {
		c.bytes = 0
		return nil, ErrLimit
	}
	c.bytes -= output.consumed
	if err != nil {
		return nil, err
	}
	return output.data, nil
}

func (s *Source) Collect(ctx context.Context) inventory.Snapshot {
	ctx, cancel := context.WithTimeout(ctx, s.totalTimeout)
	defer cancel()
	result := inventory.Snapshot{StartedAt: time.Now().UTC()}
	cycle := &cycle{source: s, bytes: s.limits.MaxBytes, records: s.limits.MaxRecords}
	result.Host = collectHost(ctx, cycle)
	result.GEOM = collectSection(ctx, cycle, geom, inventory.ParseGEOM, func(g inventory.GEOM) int {
		n := len(g.Classes) + len(g.Geoms) + len(g.Providers)
		for _, geom := range g.Geoms {
			n += len(geom.Consumers)
		}
		return n
	})
	result.Pools = collectSection(ctx, cycle, pools, inventory.ParsePools, func(items []inventory.Pool) int { return len(items) })
	result.Datasets = collectSection(ctx, cycle, datasets, inventory.ParseDatasets, func(items []inventory.Dataset) int { return len(items) })
	result.FinishedAt = time.Now().UTC()
	return result
}

func collectSection[T any](ctx context.Context, c *cycle, p probe, parse func([]byte, inventory.Limits) (T, error), count func(T) int) inventory.Section[T] {
	result := inventory.Section[T]{Status: inventory.Unavailable, StartedAt: time.Now().UTC()}
	var data []byte
	var err error
	if c.records <= 0 {
		err = ErrLimit
	} else {
		data, err = c.read(ctx, p)
	}
	if err == nil {
		limits := c.source.limits
		limits.MaxRecords = c.records
		var value T
		value, err = parse(data, limits)
		if err == nil {
			result.Data = value
			result.Status = inventory.OK
			c.records -= count(value)
		}
	}
	if err != nil {
		result.Problems = []inventory.Problem{{Source: string(p), Code: problemCode(err)}}
	}
	result.FinishedAt = time.Now().UTC()
	return result
}

var (
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+-[A-Z]+(?:-[a-zA-Z0-9_.-]+)?$`)
	archPattern    = regexp.MustCompile(`^[a-z0-9_]+$`)
	zfsPattern     = regexp.MustCompile(`^zfs-(?:kmod-)?[0-9]+\.[0-9]+\.[0-9]+[a-zA-Z0-9_.-]*$`)
)

func sourceLines(data []byte, count int, limits inventory.Limits) ([]string, error) {
	if len(data) == 0 || data[len(data)-1] != '\n' {
		return nil, inventory.ErrMalformed
	}
	if strings.Count(string(data), "\n") != count {
		return nil, inventory.ErrMalformed
	}
	values := strings.Split(string(data[:len(data)-1]), "\n")
	for _, value := range values {
		if len(value) > limits.MaxFieldBytes {
			return nil, ErrLimit
		}
	}
	return values, nil
}

func collectHost(ctx context.Context, c *cycle) inventory.Section[inventory.Host] {
	result := inventory.Section[inventory.Host]{Status: inventory.OK, StartedAt: time.Now().UTC()}
	for _, p := range []probe{versions, architecture, memory, zfsVersion} {
		data, err := c.read(ctx, p)
		if err == nil {
			err = applyHost(&result.Data, p, data, c.source.limits)
		}
		if err != nil {
			result.Problems = append(result.Problems, inventory.Problem{Source: string(p), Code: problemCode(err)})
		}
	}
	if len(result.Problems) == 4 {
		result.Status = inventory.Unavailable
	} else if len(result.Problems) != 0 {
		result.Status = inventory.Partial
	}
	result.FinishedAt = time.Now().UTC()
	return result
}

func applyHost(host *inventory.Host, p probe, data []byte, limits inventory.Limits) error {
	count := 1
	if p == versions {
		count = 3
	}
	if p == zfsVersion {
		count = 2
	}
	values, err := sourceLines(data, count, limits)
	if err != nil {
		return err
	}
	switch p {
	case versions:
		for _, value := range values {
			if !versionPattern.MatchString(value) {
				return inventory.ErrMalformed
			}
		}
		host.InstalledKernel, host.RunningKernel, host.Userland = values[0], values[1], values[2]
	case architecture:
		if !archPattern.MatchString(values[0]) {
			return inventory.ErrMalformed
		}
		host.Architecture = values[0]
	case memory:
		for _, c := range values[0] {
			if c < '0' || c > '9' {
				return inventory.ErrMalformed
			}
		}
		n, err := strconv.ParseUint(values[0], 10, 64)
		if err != nil || n == 0 {
			return inventory.ErrMalformed
		}
		host.PhysicalMemory = &n
	case zfsVersion:
		if !zfsPattern.MatchString(values[0]) || !zfsPattern.MatchString(values[1]) || !strings.HasPrefix(values[0], "zfs-") || strings.HasPrefix(values[0], "zfs-kmod-") || !strings.HasPrefix(values[1], "zfs-kmod-") {
			return inventory.ErrMalformed
		}
		host.ZFSUserland, host.ZFSKernel = values[0], values[1]
	default:
		return inventory.ErrMalformed
	}
	return nil
}

func problemCode(err error) string {
	switch {
	case errors.Is(err, inventory.ErrLimit):
		return "limit"
	case errors.Is(err, inventory.ErrMalformed):
		return "malformed"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, ErrPermission):
		return "permission"
	case errors.Is(err, ErrUnavailable):
		return "unavailable"
	default:
		return "failed"
	}
}
