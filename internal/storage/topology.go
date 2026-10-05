// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxVdevs = 4096

func statusHealth(s string) bool {
	switch s {
	case "ONLINE", "DEGRADED", "FAULTED", "OFFLINE", "REMOVED", "UNAVAIL", "AVAIL", "INUSE":
		return true
	}
	return false
}
func statusText(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, c := range s {
		if unicode.IsControl(c) && c != '\n' && c != '\t' {
			return false
		}
	}
	return true
}
func counter(s string) (uint64, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || strconv.FormatUint(n, 10) != s {
		return 0, ErrInvalid
	}
	return n, nil
}
func groupName(address, prefix string) bool {
	if !strings.HasPrefix(address, prefix+"-") {
		return false
	}
	_, err := counter(strings.TrimPrefix(address, prefix+"-"))
	return err == nil
}
func vdevKind(address string) string {
	if strings.HasPrefix(address, "/") {
		return "leaf"
	}
	if groupName(address, "mirror") {
		return "mirror"
	}
	if groupName(address, "raidz2") {
		return "raidz2"
	}
	return "unsupported"
}
func vdevAddress(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("/_-.:", c)) {
			return false
		}
	}
	if strings.HasPrefix(s, "/") {
		for _, part := range strings.Split(s[1:], "/") {
			if part == "" || part == "." || part == ".." {
				return false
			}
		}
	} else if strings.Contains(s, "/") {
		return false
	}
	return true
}
func statusAnnotation(s string) bool {
	switch s {
	case "", "(resilvering)", "(repairing)", "(awaiting resilver)", "(removing)", "(non-allocating)":
		return true
	}
	return false
}

type statusRow struct {
	depth                    int
	name, health, annotation string
	read, write, checksum    uint64
}
type statusView struct {
	topology PoolTopology
	rows     []statusRow
}

func parseStatus(data []byte, name string) (statusView, error) {
	zero := statusView{}
	if len(data) > 1<<20 || bytes.Count(data, []byte{'\n'}) > 8192 {
		return zero, ErrLimit
	}
	if len(data) == 0 || data[len(data)-1] != '\n' || !statusText(string(data)) {
		return zero, ErrInvalid
	}
	lines := strings.Split(string(data), "\n")
	seen := make(map[string]bool)
	var result statusView
	result.topology.PoolName = name
	section := 0
	active := ""
	rootSeen := false
	errorsSeen := false
	for _, line := range lines {
		if len(line) > maxFieldBytes {
			return zero, ErrLimit
		}
		body := strings.TrimSpace(line)
		if body == "" {
			continue
		}
		if errorsSeen {
			return zero, ErrInvalid
		}
		switch section {
		case 0:
			if body == "config:" {
				if !seen["pool"] || !seen["state"] || !seen["scan"] {
					return zero, ErrUnavailable
				}
				section = 1
				continue
			}
			if active != "" && (strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "        ")) {
				if active == "scan" {
					result.topology.Scan += "\n" + body
				} else {
					result.topology.Diagnostics += "\n" + body
				}
				continue
			}
			key, value, ok := strings.Cut(body, ":")
			value = strings.TrimSpace(value)
			if !ok || value == "" || seen[key] {
				return zero, ErrInvalid
			}
			if !seen["pool"] && key != "pool" {
				return zero, ErrInvalid
			}
			seen[key] = true
			active = ""
			switch key {
			case "pool":
				if value != name {
					return zero, ErrDrift
				}
			case "state":
				if !statusHealth(value) {
					return zero, ErrInvalid
				}
				result.topology.Health = value
			case "scan":
				result.topology.Scan = value
				active = key
			case "status", "action", "see":
				if result.topology.Diagnostics != "" {
					result.topology.Diagnostics += "\n"
				}
				result.topology.Diagnostics += key + ": " + value
				active = key
			default:
				return zero, ErrUnavailable
			}
		case 1:
			if strings.Join(strings.Fields(body), " ") != "NAME STATE READ WRITE CKSUM" || !strings.HasPrefix(line, "\t") {
				return zero, ErrInvalid
			}
			section = 2
		case 2:
			if strings.HasPrefix(body, "errors:") {
				if !rootSeen || len(result.rows) < 2 {
					return zero, ErrUnavailable
				}
				if body != "errors: No known data errors" {
					value, ok := strings.CutPrefix(body, "errors: ")
					if !ok {
						return zero, ErrInvalid
					}
					value, ok = strings.CutSuffix(value, " data errors, use '-v' for a list")
					if !ok {
						return zero, ErrUnavailable
					}
					n, err := counter(value)
					if err != nil || n == 0 {
						return zero, ErrInvalid
					}
					result.topology.DataErrors = n
				}
				errorsSeen = true
				continue
			}
			if !strings.HasPrefix(line, "\t") {
				return zero, ErrInvalid
			}
			rest := line[1:]
			depth := len(rest) - len(strings.TrimLeft(rest, " "))
			if depth > 64 || depth%2 != 0 {
				return zero, ErrInvalid
			}
			fields := strings.Fields(rest)
			if len(fields) < 5 {
				return zero, ErrUnavailable
			}
			annotation := strings.Join(fields[5:], " ")
			if !statusHealth(fields[1]) || !statusAnnotation(annotation) {
				return zero, ErrUnavailable
			}
			row := statusRow{depth: depth, name: fields[0], health: fields[1], annotation: annotation}
			var err error
			row.read, err = counter(fields[2])
			if err != nil {
				return zero, err
			}
			row.write, err = counter(fields[3])
			if err != nil {
				return zero, err
			}
			row.checksum, err = counter(fields[4])
			if err != nil {
				return zero, err
			}
			if !rootSeen {
				if depth != 0 || row.name != name || row.health != result.topology.Health || annotation != "" {
					return zero, ErrInvalid
				}
				rootSeen = true
				result.topology.ReadErrors = row.read
				result.topology.WriteErrors = row.write
				result.topology.ChecksumErrors = row.checksum
			} else if depth == 0 {
				return zero, ErrInvalid
			}
			if len(result.rows) > maxVdevs {
				return zero, ErrLimit
			}
			result.rows = append(result.rows, row)
		}
		if len(result.topology.Scan) > maxFieldBytes || len(result.topology.Diagnostics) > maxFieldBytes {
			return zero, ErrLimit
		}
	}
	if !errorsSeen {
		return zero, ErrUnavailable
	}
	return result, nil
}

// ParsePoolTopology pairs fixed successful legacy projections:
// zpool status -g -p <pool> and zpool status -L -P -p <pool>.
// The independent fresh pool GUID is a trusted collector input. Status text does
// not expose it. Matching rows do not make separate native reads atomic.
func ParsePoolTopology(guids, paths []byte, name, guid string) (PoolTopology, error) {
	if !validName(name) || strings.Contains(name, "/") || !validGUID(guid) {
		return PoolTopology{}, ErrInvalid
	}
	a, err := parseStatus(guids, name)
	if err != nil {
		return PoolTopology{}, err
	}
	b, err := parseStatus(paths, name)
	if err != nil {
		return PoolTopology{}, err
	}
	if !reflect.DeepEqual(a.topology, b.topology) || len(a.rows) != len(b.rows) {
		return PoolTopology{}, ErrDrift
	}
	result := a.topology
	result.PoolGUID = guid
	stack := make(map[int]string)
	ordinals := make(map[string]int)
	previousDepth := 0
	for i, row := range a.rows {
		peer := b.rows[i]
		if row.depth != peer.depth || row.health != peer.health || row.annotation != peer.annotation || row.read != peer.read || row.write != peer.write || row.checksum != peer.checksum {
			return PoolTopology{}, ErrDrift
		}
		if i == 0 {
			continue
		}
		if !validGUID(row.name) || !vdevAddress(peer.name) || row.depth > previousDepth+2 {
			return PoolTopology{}, ErrInvalid
		}
		parent := ""
		if row.depth > 2 {
			parent = stack[row.depth-2]
			if parent == "" {
				return PoolTopology{}, ErrInvalid
			}
		}
		for depth := range stack {
			if depth >= row.depth {
				delete(stack, depth)
			}
		}
		stack[row.depth] = row.name
		previousDepth = row.depth
		node := Vdev{GUID: row.name, ParentGUID: parent, Address: peer.name, Kind: vdevKind(peer.name), Health: row.health, Annotation: row.annotation, Ordinal: ordinals[parent], ReadErrors: row.read, WriteErrors: row.write, ChecksumErrors: row.checksum}
		ordinals[parent]++
		result.Nodes = append(result.Nodes, node)
	}
	if err := validateTopology(result); err != nil {
		return PoolTopology{}, err
	}
	return normalizedTopology(result), nil
}
func normalizedTopology(t PoolTopology) PoolTopology {
	t.Nodes = append([]Vdev(nil), t.Nodes...)
	sort.Slice(t.Nodes, func(i, j int) bool { return t.Nodes[i].GUID < t.Nodes[j].GUID })
	return t
}
func validateTopology(t PoolTopology) error {
	if t.PoolGUID == "" && t.PoolName == "" && len(t.Nodes) == 0 {
		return ErrUnavailable
	}
	if err := preflight(Policy{}, Observation{Topology: t}); err != nil {
		return err
	}
	if !validGUID(t.PoolGUID) || !validName(t.PoolName) || strings.Contains(t.PoolName, "/") || !statusHealth(t.Health) || t.Scan == "" || !statusText(t.Scan) || !statusText(t.Diagnostics) || len(t.Nodes) == 0 {
		return ErrInvalid
	}
	nodes := make(map[string]Vdev, len(t.Nodes))
	addresses := make(map[string]bool, len(t.Nodes))
	children := make(map[string]int)
	orders := make(map[string]map[int]bool)
	for _, node := range t.Nodes {
		if !validGUID(node.GUID) || node.GUID == t.PoolGUID || nodes[node.GUID].GUID != "" || !vdevAddress(node.Address) || addresses[node.Address] || vdevKind(node.Address) != node.Kind || !statusHealth(node.Health) || !statusAnnotation(node.Annotation) || node.Ordinal < 0 || node.Ordinal >= maxVdevs {
			return ErrInvalid
		}
		nodes[node.GUID] = node
		addresses[node.Address] = true
		children[node.ParentGUID]++
		if orders[node.ParentGUID] == nil {
			orders[node.ParentGUID] = make(map[int]bool)
		}
		if orders[node.ParentGUID][node.Ordinal] {
			return ErrInvalid
		}
		orders[node.ParentGUID][node.Ordinal] = true
	}
	for _, node := range t.Nodes {
		if node.Ordinal >= children[node.ParentGUID] {
			return ErrInvalid
		}
		if node.Kind == "leaf" && children[node.GUID] != 0 {
			return ErrInvalid
		}
		parent := node.ParentGUID
		for depth := 0; parent != ""; depth++ {
			ancestor, ok := nodes[parent]
			if !ok || ancestor.Kind == "leaf" || parent == node.GUID || depth >= 32 {
				return ErrInvalid
			}
			parent = ancestor.ParentGUID
		}
	}
	return nil
}

// VerifyPoolTopology recognizes healthy homogeneous mirror or RAIDZ2 groups
// only, with no leaf/root/data errors or annotations and no requested scan.
// It establishes neither provider provenance nor system/device-use safety.
func VerifyPoolTopology(guid, name string, t PoolTopology) error {
	if err := validateTopology(t); err != nil {
		return err
	}
	if !validGUID(guid) || !validName(name) || strings.Contains(name, "/") {
		return ErrInvalid
	}
	if t.PoolGUID != guid || t.PoolName != name {
		return ErrDrift
	}
	if t.Health != "ONLINE" || t.Scan != "none requested" || t.ReadErrors != 0 || t.WriteErrors != 0 || t.ChecksumErrors != 0 || t.DataErrors != 0 {
		return ErrUnavailable
	}
	children := make(map[string]int)
	for _, node := range t.Nodes {
		children[node.ParentGUID]++
	}
	layout := ""
	width := 0
	for _, node := range t.Nodes {
		if node.Health != "ONLINE" || node.ReadErrors != 0 || node.WriteErrors != 0 || node.ChecksumErrors != 0 || node.Annotation != "" {
			return ErrUnavailable
		}
		if node.ParentGUID == "" {
			minimum := 2
			if node.Kind == "raidz2" {
				minimum = 4
			} else if node.Kind != "mirror" {
				return ErrUnavailable
			}
			if children[node.GUID] < minimum {
				return ErrUnavailable
			}
			if layout != "" && (node.Kind != layout || children[node.GUID] != width) {
				return ErrUnavailable
			}
			layout = node.Kind
			width = children[node.GUID]
		} else if node.Kind != "leaf" || !strings.HasPrefix(node.Address, "/dev/") {
			return ErrUnavailable
		}
	}
	return nil
}

// VerifyTopologyStability detects observed changes in two complete projections;
// no repeated read or advisory check fences another root writer.
func VerifyTopologyStability(a, b PoolTopology) error {
	if err := validateTopology(a); err != nil {
		return err
	}
	if err := validateTopology(b); err != nil {
		return err
	}
	if !reflect.DeepEqual(normalizedTopology(a), normalizedTopology(b)) {
		return ErrDrift
	}
	return nil
}
