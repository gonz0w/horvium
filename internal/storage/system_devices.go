// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"math"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gonz0w/horvium/internal/inventory"
)

// PoolUseCapture contains complete successful native projections for one mounted
// pool. Names are current addresses. Captures must be fresh; they are not atomic.
type PoolUseCapture struct {
	Name                           string
	GUID, StatusGUIDs, StatusPaths []byte
}

// MountUse retains positive backing facts, including unowned/unidentified root
// disks. Empty Disks is permitted only for the explicitly recognized non-block
// mounts; it is never a declaration that any selected device is unused.
type MountUse struct {
	Source, Target, Type, PoolGUID string
	Disks                          []inventory.Provider
}
type SwapUse struct {
	Source                                  string
	SizeKiB, UsedKiB, AvailableKiB, Percent uint64
	Disks                                   []inventory.Provider
}

// SystemDeviceUse maps observed mount/swap use to direct GEOM disks. It grants
// neither complete use clearance nor any mutation/enrollment authority.
type SystemDeviceUse struct {
	Mounts []MountUse
	Swap   []SwapUse
}

type systemBudget struct {
	limits    inventory.Limits
	remaining int
}

func (b *systemBudget) spend(n int) error {
	if n > b.remaining {
		return ErrLimit
	}
	b.remaining -= n
	return nil
}
func systemField(s string, l inventory.Limits) error {
	if len(s) > l.MaxFieldBytes {
		return ErrLimit
	}
	if s == "" || !utf8.ValidString(s) {
		return ErrInvalid
	}
	for _, c := range s {
		if unicode.IsControl(c) {
			return ErrInvalid
		}
	}
	return nil
}
func systemRows(data []byte, count int, b *systemBudget, visit func([]string) error) error {
	if len(data) == 0 || data[len(data)-1] != '\n' || !statusText(string(data)) {
		return ErrInvalid
	}
	if strings.Count(string(data), "\n") > b.remaining {
		return ErrLimit
	}
	lines := strings.Split(string(data[:len(data)-1]), "\n")
	for _, line := range lines {
		if err := b.spend(1); err != nil {
			return err
		}
		// Native mount -p prints raw names, without an escape encoding. Do not
		// reinterpret backslashes, shell quoting or Unicode as field separators.
		fields := make([]string, 0, count)
		for i := 0; i < len(line); {
			if line[i] == ' ' || line[i] == '\t' {
				i++
				continue
			}
			start := i
			for i < len(line) && line[i] != ' ' && line[i] != '\t' {
				i++
			}
			if len(fields) == count {
				return ErrInvalid
			}
			if err := systemField(line[start:i], b.limits); err != nil {
				return err
			}
			fields = append(fields, line[start:i])
		}
		if len(fields) != count {
			return ErrInvalid
		}
		if err := visit(fields); err != nil {
			return err
		}
	}
	return nil
}
func systemPath(s string) bool {
	return strings.HasPrefix(s, "/") && path.Clean(s) == s && !strings.Contains(s, "\\")
}
func devicePath(s string) bool { return strings.HasPrefix(s, "/dev/") && s != "/dev/" && systemPath(s) }
func mountTable(data []byte, b *systemBudget) ([]MountUse, error) {
	var mounts []MountUse
	targets := make(map[string]bool)
	roots := 0
	err := systemRows(data, 6, b, func(f []string) error {
		m := MountUse{Source: f[0], Target: f[1], Type: f[2]}
		// Inspected statfs on/from buffers are 1024 bytes on both targets.
		// A full buffer can be a truncated name that happens to match another
		// provider. Refuse it even when that spelling exists in GEOM.
		if len(m.Source) >= 1023 || len(m.Target) >= 1023 {
			return ErrUnavailable
		}
		if !systemPath(m.Target) || targets[m.Target] {
			return ErrInvalid
		}
		targets[m.Target] = true
		ro, rw := 0, 0
		for _, option := range strings.Split(f[3], ",") {
			if option == "ro" {
				ro++
			}
			if option == "rw" {
				rw++
			}
			if option == "" {
				return ErrInvalid
			}
		}
		if ro+rw != 1 {
			return ErrInvalid
		}
		for _, n := range f[4:] {
			v, err := strconv.ParseUint(n, 10, 32)
			if err != nil || strconv.FormatUint(v, 10) != n {
				return ErrInvalid
			}
		}
		if m.Target == "/" {
			roots++
			if m.Type != "ufs" && m.Type != "zfs" {
				return ErrUnavailable
			}
		}
		mounts = append(mounts, m)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if roots != 1 {
		return nil, ErrUnavailable
	}
	return mounts, nil
}
func swapTable(data []byte, b *systemBudget) ([]SwapUse, error) {
	var swaps []SwapUse
	seen := make(map[string]bool)
	header, total := false, false
	var size, used, available uint64
	err := systemRows(data, 5, b, func(f []string) error {
		if !header {
			header = true
			if strings.Join(f, " ") != "Device 1K-blocks Used Avail Capacity" {
				return ErrInvalid
			}
			return nil
		}
		if total || seen[f[0]] {
			return ErrInvalid
		}
		seen[f[0]] = true
		var values [4]uint64
		for i, s := range f[1:] {
			if i == 3 {
				var ok bool
				s, ok = strings.CutSuffix(s, "%")
				if !ok {
					return ErrInvalid
				}
			}
			v, err := counter(s)
			if err != nil {
				return err
			}
			values[i] = v
		}
		if values[0] == 0 || values[1] > values[0] || values[2] != values[0]-values[1] || values[3] > 100 {
			return ErrInvalid
		}
		if f[0] == "Total" {
			total = true
			if len(swaps) < 2 || values[0] != size || values[1] != used || values[2] != available {
				return ErrInvalid
			}
			return nil
		}
		// The inspected kvm_swap.ksw_devname buffer is 32 bytes; the live
		// swapinfo producer uses snprintf into it even without /dev/kmem.
		if len(f[0]) >= 31 || !devicePath(f[0]) {
			return ErrUnavailable
		}
		if values[0] > math.MaxUint64-size || values[1] > math.MaxUint64-used || values[2] > math.MaxUint64-available {
			return ErrInvalid
		}
		size += values[0]
		used += values[1]
		available += values[2]
		swaps = append(swaps, SwapUse{Source: f[0], SizeKiB: values[0], UsedKiB: values[1], AvailableKiB: values[2], Percent: values[3]})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if (len(swaps) > 1) != total {
		return nil, ErrUnavailable
	}
	return swaps, nil
}

type systemResolver struct {
	providers map[string]inventory.Provider
	geoms     map[string]inventory.Geom
	names     map[string][]string
	outputs   map[string]int
	budget    *systemBudget
}

func (r *systemResolver) disks(addresses []string) ([]inventory.Provider, error) {
	roots := make(map[string]inventory.Provider)
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var follow func(string, int) error
	follow = func(id string, depth int) error {
		if visiting[id] || depth > r.budget.limits.MaxXMLDepth {
			return ErrUnavailable
		}
		if visited[id] {
			return nil
		}
		if err := r.budget.spend(1); err != nil {
			return err
		}
		p, ok := r.providers[id]
		if !ok {
			return ErrUnavailable
		}
		g, ok := r.geoms[p.GeomID]
		if !ok {
			return ErrUnavailable
		}
		if p.Kind == inventory.DiskCandidate {
			if p.Class != "DISK" || g.Name != p.Name || len(g.Consumers) != 0 || r.outputs[g.ID] != 1 || len(r.names[p.Name]) != 1 {
				return ErrUnavailable
			}
			roots[id] = p
			visited[id] = true
			return nil
		}
		if len(g.Consumers) == 0 {
			return ErrUnavailable
		}
		visiting[id] = true
		for _, c := range g.Consumers {
			if err := follow(c.ProviderID, depth+1); err != nil {
				return err
			}
		}
		delete(visiting, id)
		visited[id] = true
		return nil
	}
	if len(addresses) == 0 {
		return nil, ErrUnavailable
	}
	for _, address := range addresses {
		if !devicePath(address) {
			return nil, ErrUnavailable
		}
		ids := r.names[strings.TrimPrefix(address, "/dev/")]
		if len(ids) != 1 {
			return nil, ErrUnavailable
		}
		if err := follow(ids[0], 0); err != nil {
			return nil, err
		}
	}
	disks := make([]inventory.Provider, 0, len(roots))
	for _, p := range roots {
		disks = append(disks, p)
	}
	sort.Slice(disks, func(i, j int) bool { return disks[i].Name < disks[j].Name })
	return disks, nil
}

// ParseSystemDeviceUse correlates complete GEOM, mount -p, swapinfo -k and paired
// status/GUID captures for exactly the mounted ZFS pools. Unsupported or partial
// source/backing means unavailable, never absent use. No native commands run.
func ParseSystemDeviceUse(geom, mount, swap []byte, pools []PoolUseCapture, limits inventory.Limits) (SystemDeviceUse, error) {
	zero := SystemDeviceUse{}
	if limits.Validate() != nil {
		return zero, ErrInvalid
	}
	remaining := limits.MaxBytes
	take := func(data []byte) bool {
		if len(data) > remaining {
			return false
		}
		remaining -= len(data)
		return true
	}
	if !take(geom) || !take(mount) || !take(swap) {
		return zero, ErrLimit
	}
	if len(pools) > limits.MaxRecords {
		return zero, ErrLimit
	}
	for _, p := range pools {
		if !take([]byte(p.Name)) || !take(p.GUID) || !take(p.StatusGUIDs) || !take(p.StatusPaths) {
			return zero, ErrLimit
		}
	}
	graph, err := inventory.ParseGEOM(geom, limits)
	if err != nil {
		if errors.Is(err, inventory.ErrLimit) {
			return zero, ErrLimit
		}
		return zero, ErrInvalid
	}
	b := systemBudget{limits: limits, remaining: limits.MaxRecords}
	if err := b.spend(len(graph.Classes) + len(graph.Geoms) + len(graph.Providers)); err != nil {
		return zero, err
	}
	r := systemResolver{providers: make(map[string]inventory.Provider), geoms: make(map[string]inventory.Geom), names: make(map[string][]string), outputs: make(map[string]int), budget: &b}
	for _, g := range graph.Geoms {
		if err := b.spend(len(g.Consumers)); err != nil {
			return zero, err
		}
		r.geoms[g.ID] = g
	}
	for _, p := range graph.Providers {
		r.providers[p.ID] = p
		r.names[p.Name] = append(r.names[p.Name], p.ID)
		r.outputs[p.GeomID]++
	}
	mounts, err := mountTable(mount, &b)
	if err != nil {
		return zero, err
	}
	swaps, err := swapTable(swap, &b)
	if err != nil {
		return zero, err
	}
	required := make(map[string]bool)
	for _, m := range mounts {
		if m.Type == "zfs" {
			if !validName(m.Source) || strings.ContainsAny(m.Source, "@#") {
				return zero, ErrUnavailable
			}
			required[strings.Split(m.Source, "/")[0]] = true
		}
	}
	if len(required) != len(pools) {
		return zero, ErrUnavailable
	}
	type poolBacking struct {
		guid  string
		disks []inventory.Provider
	}
	backing := make(map[string]poolBacking)
	for _, p := range pools {
		if err := b.spend(1); err != nil {
			return zero, err
		}
		if err := systemField(p.Name, limits); err != nil {
			return zero, err
		}
		if !required[p.Name] || backing[p.Name].guid != "" {
			return zero, ErrUnavailable
		}
		if len(p.GUID) < 2 || p.GUID[len(p.GUID)-1] != '\n' {
			return zero, ErrInvalid
		}
		guid := string(p.GUID[:len(p.GUID)-1])
		if err := systemField(guid, limits); err != nil {
			return zero, err
		}
		if !validGUID(guid) {
			return zero, ErrInvalid
		}
		for _, capture := range [][]byte{p.StatusGUIDs, p.StatusPaths} {
			if len(capture) > 1<<20 || strings.Count(string(capture), "\n") > 8192 {
				return zero, ErrLimit
			}
			for _, line := range strings.Split(string(capture), "\n") {
				if len(line) > limits.MaxFieldBytes {
					return zero, ErrLimit
				}
			}
		}
		topology, err := ParsePoolTopology(p.StatusGUIDs, p.StatusPaths, p.Name, guid)
		if err != nil {
			return zero, err
		}
		if err := b.spend(len(topology.Nodes)); err != nil {
			return zero, err
		}
		parents := make(map[string]bool)
		for _, node := range topology.Nodes {
			parents[node.ParentGUID] = true
		}
		var leaves []string
		for _, node := range topology.Nodes {
			if !parents[node.GUID] {
				if node.Kind != "leaf" || !devicePath(node.Address) {
					return zero, ErrUnavailable
				}
				leaves = append(leaves, node.Address)
			}
		}
		disks, err := r.disks(leaves)
		if err != nil {
			return zero, err
		}
		backing[p.Name] = poolBacking{guid, disks}
	}
	directCache := make(map[string][]inventory.Provider)
	resolve := func(address string) ([]inventory.Provider, error) {
		if d, ok := directCache[address]; ok {
			return d, nil
		}
		d, err := r.disks([]string{address})
		if err == nil {
			directCache[address] = d
		}
		return d, err
	}
	for i, m := range mounts {
		var disks []inventory.Provider
		switch {
		case m.Type == "zfs":
			p := backing[strings.Split(m.Source, "/")[0]]
			mounts[i].PoolGUID = p.guid
			disks = p.disks
		case devicePath(m.Source):
			disks, err = resolve(m.Source)
			if err != nil {
				return zero, err
			}
		case m.Target != "/" && m.Source == m.Type && (m.Type == "devfs" || m.Type == "tmpfs" || m.Type == "procfs" || m.Type == "fdescfs"):
		case m.Target != "/" && m.Type == "nfs" && strings.Contains(m.Source, ":"):
		default:
			return zero, ErrUnavailable
		}
		if m.Target == "/" && len(disks) == 0 {
			return zero, ErrUnavailable
		}
		if err := b.spend(len(disks)); err != nil {
			return zero, err
		}
		mounts[i].Disks = append([]inventory.Provider(nil), disks...)
	}
	for i, s := range swaps {
		disks, err := resolve(s.Source)
		if err != nil {
			return zero, err
		}
		if err := b.spend(len(disks)); err != nil {
			return zero, err
		}
		swaps[i].Disks = append([]inventory.Provider(nil), disks...)
	}
	sort.Slice(mounts, func(i, j int) bool { return mounts[i].Target < mounts[j].Target })
	sort.Slice(swaps, func(i, j int) bool { return swaps[i].Source < swaps[j].Source })
	return SystemDeviceUse{mounts, swaps}, nil
}
