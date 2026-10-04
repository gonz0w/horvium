// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

type processUsage struct {
	RSSBytes uint64
	CPU      time.Duration
}

func parseUsage(b []byte, pids [2]int) (processUsage, error) {
	var out processUsage
	seen := map[int]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 {
			return out, errors.New("invalid fixture process sample")
		}
		pid, e := strconv.Atoi(f[0])
		if e != nil || seen[pid] || pid != pids[0] && pid != pids[1] {
			return out, errors.New("unexpected fixture process")
		}
		seen[pid] = true
		rss, e := strconv.ParseUint(f[1], 10, 64)
		if e != nil || rss > math.MaxUint64/1024 {
			return out, errors.New("invalid RSS")
		}
		rss *= 1024
		cpu, e := cpuTime(f[2])
		if e != nil || rss > math.MaxUint64-out.RSSBytes || cpu > time.Duration(math.MaxInt64)-out.CPU {
			return out, errors.New("invalid fixture usage")
		}
		out.RSSBytes += rss
		out.CPU += cpu
	}
	if len(seen) != 2 {
		return out, errors.New("missing fixture process")
	}
	return out, nil
}
func digits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
func cpuTime(s string) (time.Duration, error) {
	p := strings.Split(s, ":")
	if len(p) != 2 && len(p) != 3 {
		return 0, errors.New("invalid CPU time")
	}
	sec := strings.Split(p[len(p)-1], ".")
	if len(sec) > 2 || !digits(sec[0]) || len(sec) == 2 && (!digits(sec[1]) || len(sec[1]) > 9) {
		return 0, errors.New("invalid CPU seconds")
	}
	n, e := strconv.Atoi(sec[0])
	if e != nil || n >= 60 {
		return 0, errors.New("invalid CPU seconds")
	}
	for _, part := range p[:len(p)-1] {
		if !digits(part) {
			return 0, errors.New("invalid CPU time")
		}
	}
	prefix := p[0] + "m"
	if len(p) == 3 {
		minute, e := strconv.Atoi(p[1])
		if e != nil || minute >= 60 {
			return 0, errors.New("invalid CPU minutes")
		}
		prefix = p[0] + "h" + p[1] + "m"
	}
	return time.ParseDuration(prefix + p[len(p)-1] + "s")
}
func idlePercent(start, end, elapsed time.Duration) (float64, error) {
	if elapsed <= 0 || start < 0 || end < start {
		return 0, errors.New("invalid CPU measurement")
	}
	return 100 * float64(end-start) / float64(elapsed), nil
}
func performanceSnapshot(count, padding int) inventory.Snapshot {
	now := time.Now().UTC()
	mem := uint64(6144 << 20)
	s := inventory.Snapshot{StartedAt: now, FinishedAt: now}
	s.Host = inventory.Section[inventory.Host]{Status: inventory.OK, StartedAt: now, FinishedAt: now, Data: inventory.Host{InstalledKernel: "synthetic-performance", RunningKernel: "synthetic-performance", Userland: "synthetic-performance", Architecture: "amd64", PhysicalMemory: &mem, ZFSUserland: "synthetic-performance", ZFSKernel: "synthetic-performance"}}
	s.GEOM = inventory.Section[inventory.GEOM]{Status: inventory.OK, StartedAt: now, FinishedAt: now, Data: inventory.GEOM{Classes: []inventory.Class{{ID: "1", Name: "DISK"}}}}
	for i := 0; i < 4; i++ {
		gid, pid := strconv.Itoa(2+i), strconv.Itoa(6+i)
		name := fmt.Sprintf("fixture%d", i)
		s.GEOM.Data.Geoms = append(s.GEOM.Data.Geoms, inventory.Geom{ID: gid, ClassID: "1", Name: name})
		s.GEOM.Data.Providers = append(s.GEOM.Data.Providers, inventory.Provider{ID: pid, GeomID: gid, Name: name, Class: "DISK", Kind: inventory.DiskCandidate, MediaSize: 3e12, SectorSize: 512, Identity: inventory.IdentityMissing})
	}
	s.Pools = inventory.Section[[]inventory.Pool]{Status: inventory.OK, StartedAt: now, FinishedAt: now, Data: []inventory.Pool{{Name: "vault", GUID: "10", Size: 12e12, Free: 12e12, Health: "ONLINE"}}}
	s.Datasets = inventory.Section[[]inventory.Dataset]{Status: inventory.OK, StartedAt: now, FinishedAt: now, Data: make([]inventory.Dataset, count)}
	suffix := strings.Repeat("x", padding)
	for i := range s.Datasets.Data {
		s.Datasets.Data[i] = inventory.Dataset{Name: fmt.Sprintf("vault/dataset-%06d", i) + suffix, GUID: strconv.Itoa(100 + i), Used: uint64(i), Referenced: uint64(i), Available: 1 << 40}
	}
	return s
}
