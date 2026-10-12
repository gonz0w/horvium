// SPDX-License-Identifier: BSD-3-Clause
package console

import (
	"fmt"
	"math/bits"

	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

const maxCapacityRows = 20

type capacityMeasure struct {
	Width   uint64
	Percent string
}
type poolComparison struct {
	Name, GUID, URL string
	Allocated, Free capacityMeasure
	ZeroSize        bool
}
type poolCapacity struct {
	Rows              []poolComparison
	Start, End, Total int
}

// Coordinates are rounded to tenths of a percent using exact integer arithmetic.
// No byte value or identity passes through a floating-point/browser number.
func capacityRatio(value, size uint64) capacityMeasure {
	if size == 0 || value > size {
		return capacityMeasure{Percent: "Unavailable"}
	}
	hi, lo := bits.Mul64(value, 1000)
	width, remainder := bits.Div64(hi, lo, size)
	// Equivalent to 2*remainder >= size without overflowing at uint64 maximum.
	if remainder >= size-remainder {
		width++
	}
	return capacityMeasure{Width: width, Percent: fmt.Sprintf("%d.%d%%", width/10, width%10)}
}

// Compare only detached rows already selected for the exact table. No extra
// manager query, native collection or whole-inventory materialization occurs.
func poolComparisons(p observation.Page[inventory.Pool], q observation.Query) *poolCapacity {
	if p.Freshness.Status != inventory.OK {
		return nil
	}
	n := min(len(p.Rows), maxCapacityRows)
	c := &poolCapacity{Rows: make([]poolComparison, n), Total: p.Total}
	if n > 0 {
		c.Start = p.Offset + 1
		c.End = p.Offset + n
	}
	q.Generation = p.Generation
	for i, pool := range p.Rows[:n] {
		c.Rows[i] = poolComparison{Name: pool.Name, GUID: pool.GUID, URL: queryURL("/storage/pools", q, pool.GUID), Allocated: capacityRatio(pool.Allocated, pool.Size), Free: capacityRatio(pool.Free, pool.Size), ZeroSize: pool.Size == 0}
	}
	return c
}

type datasetComparison struct {
	Name, GUID, URL string
	Used            capacityMeasure
}
type datasetCapacity struct {
	Rows              []datasetComparison
	Start, End, Total int
	Maximum           uint64
}

// Used values can overlap across datasets. This scale is the largest individual
// Used on this detached page, not a sum, pool utilization or quota headroom.
func datasetComparisons(p observation.Page[inventory.Dataset], q observation.Query) *datasetCapacity {
	if p.Freshness.Status != inventory.OK {
		return nil
	}
	n := min(len(p.Rows), maxCapacityRows)
	c := &datasetCapacity{Rows: make([]datasetComparison, n), Total: p.Total}
	for _, d := range p.Rows {
		c.Maximum = max(c.Maximum, d.Used)
	}
	if n > 0 {
		c.Start = p.Offset + 1
		c.End = p.Offset + n
	}
	q.Generation = p.Generation
	for i, d := range p.Rows[:n] {
		c.Rows[i] = datasetComparison{Name: d.Name, GUID: d.GUID, URL: queryURL("/storage/datasets", q, d.GUID), Used: capacityRatio(d.Used, c.Maximum)}
	}
	return c
}
