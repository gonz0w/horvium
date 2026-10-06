// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/sha256"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/gonz0w/horvium/internal/inventory"
)

// PoolLabelCapture is complete successful stdout from zdb -l on the exact
// absolute direct-disk argument. Execution, target, freshness and provenance
// must be established by the trusted caller. No native operation runs here.
type PoolLabelCapture struct {
	Device string
	Output []byte
}

// PoolLabelConfiguration retains positive on-disk metadata. State is the native
// decimal pool state (0 active, 1 exported, 2 destroyed), not current import,
// ownership or health. Labels identifies a deduplicated configuration group;
// zdb prints a checksum warning only for its representative label. No all-label
// checksum integrity, complete topology or absence inference is supplied.
type PoolLabelConfiguration struct {
	Name, PoolGUID, TopGUID, VdevGUID, State, TXG string
	Labels                                        []int
	ConfigurationHash                             [32]byte
}
type PoolLabels struct {
	Disk           inventory.DiskObservation
	Configurations []PoolLabelConfiguration
}

var labelKey = regexp.MustCompile(`^[A-Za-z0-9_.:\-]+(?:\[[0-9]+\])?$`)

const labelSeparator = "------------------------------------"

// ParsePoolLabels accepts positive, completely represented four-label captures
// in the qualified default -l grammar. It rejects diagnostics, reported bad
// representative checksums, incomplete coverage and contradictory identities.
// No successful result means an unused disk or authority to import/overwrite it.
func ParsePoolLabels(geom, info, serial []byte, c PoolLabelCapture, l inventory.Limits) (PoolLabels, error) {
	zero := PoolLabels{}
	if l.Validate() != nil {
		return zero, ErrInvalid
	}
	remaining := l.MaxBytes
	for _, n := range []int{len(geom), len(info), len(serial), len(c.Device), len(c.Output)} {
		if n > remaining {
			return zero, ErrLimit
		}
		remaining -= n
	}
	if err := systemField(c.Device, l); err != nil {
		return zero, err
	}
	if !devicePath(c.Device) {
		return zero, ErrInvalid
	}
	if len(c.Output) == 0 || c.Output[len(c.Output)-1] != '\n' {
		return zero, ErrUnavailable
	}
	rows := strings.Split(strings.TrimSuffix(string(c.Output), "\n"), "\n")
	// Every header/body row and two independent scalar captures share the graph
	// record budget. Refuse before allocating a graph or admitting partial rows.
	if len(rows) > l.MaxRecords-3 {
		return zero, ErrLimit
	}
	for _, row := range rows {
		if err := systemField(row, l); err != nil {
			return zero, err
		}
	}
	graphLimits := l
	graphLimits.MaxRecords -= len(rows) + 2
	disk, err := inventory.ParseDiskObservation(geom, info, serial, graphLimits)
	if err != nil {
		if errors.Is(err, inventory.ErrLimit) {
			return zero, ErrLimit
		}
		if errors.Is(err, inventory.ErrDiskCorrelation) {
			return zero, ErrDrift
		}
		return zero, ErrInvalid
	}
	if c.Device != "/dev/"+disk.Provider.Name {
		return zero, ErrDrift
	}
	result := PoolLabels{Disk: disk}
	covered := [4]bool{}
	for i := 0; i < len(rows); {
		if i+3 >= len(rows) || rows[i] != labelSeparator || rows[i+2] != labelSeparator {
			return zero, ErrUnavailable
		}
		representative := -1
		for n := 0; n < 4; n++ {
			if rows[i+1] == "LABEL "+strconv.Itoa(n)+" " {
				representative = n
			}
		}
		if representative < 0 || covered[representative] {
			return zero, ErrUnavailable
		}
		i += 3
		start := i
		values := map[string]string{}
		// Each level owns a distinct key set. A deeper row must immediately follow
		// a container; siblings cannot turn scalar values into parent sections.
		keys := []map[string]bool{{}}
		previousDepth := 1
		previousContainer := false
		for i < len(rows) && !strings.HasPrefix(rows[i], "    labels = ") {
			row := rows[i]
			indent := len(row) - len(strings.TrimLeft(row, " "))
			if indent < 4 || indent%4 != 0 || indent/4 > l.MaxXMLDepth {
				return zero, ErrUnavailable
			}
			depth := indent / 4
			if depth > previousDepth && (depth != previousDepth+1 || !previousContainer) {
				return zero, ErrUnavailable
			}
			if depth > len(keys) {
				keys = append(keys, map[string]bool{})
			} else {
				keys = keys[:depth]
			}
			text := row[indent:]
			key, value, container := "", "", false
			if strings.HasSuffix(text, ":") {
				key = strings.TrimSuffix(text, ":")
				container = true
			} else {
				cut := strings.LastIndex(text, ": ")
				if cut < 1 {
					return zero, ErrUnavailable
				}
				key, value = text[:cut], text[cut+2:]
			}
			if !labelKey.MatchString(key) || keys[depth-1][key] {
				return zero, ErrUnavailable
			}
			keys[depth-1][key] = true
			if !container && !labelValue(value) {
				return zero, ErrUnavailable
			}
			if depth == 1 {
				switch key {
				case "vdev_tree", "features_for_read":
					if !container {
						return zero, ErrUnavailable
					}
				case "version", "state", "txg", "pool_guid", "top_guid", "guid", "vdev_children", "errata", "hostid":
					if container || !labelNumber(value, true) {
						return zero, ErrUnavailable
					}
				case "name", "hostname":
					if container || !labelString(value) {
						return zero, ErrUnavailable
					}
				default:
					return zero, ErrUnavailable
				}
				values[key] = value
			}
			previousDepth, previousContainer = depth, container
			i++
		}
		if i == len(rows) {
			return zero, ErrUnavailable
		}
		config := PoolLabelConfiguration{Name: strings.Trim(values["name"], "'"), PoolGUID: values["pool_guid"], TopGUID: values["top_guid"], VdevGUID: values["guid"], State: values["state"], TXG: values["txg"]}
		if !validName(config.Name) || strings.ContainsAny(config.Name, "/@#") || values["version"] != "5000" || (config.State != "0" && config.State != "1" && config.State != "2") || !labelNumber(config.PoolGUID, false) || !labelNumber(config.TopGUID, false) || !labelNumber(config.VdevGUID, false) || !labelNumber(config.TXG, false) {
			return zero, ErrUnavailable
		}
		if _, ok := values["vdev_tree"]; !ok {
			return zero, ErrUnavailable
		}
		group := strings.TrimPrefix(rows[i], "    labels = ")
		if !strings.HasSuffix(group, " ") {
			return zero, ErrUnavailable
		}
		for _, part := range strings.Split(strings.TrimSuffix(group, " "), " ") {
			n, err := strconv.Atoi(part)
			if err != nil || part != strconv.Itoa(n) || n < 0 || n > 3 || covered[n] || (len(config.Labels) > 0 && n <= config.Labels[len(config.Labels)-1]) {
				return zero, ErrUnavailable
			}
			covered[n] = true
			config.Labels = append(config.Labels, n)
		}
		if len(config.Labels) == 0 || config.Labels[0] != representative {
			return zero, ErrUnavailable
		}
		config.ConfigurationHash = sha256.Sum256([]byte(strings.Join(rows[start:i], "\n")))
		if len(result.Configurations) > 0 {
			first := result.Configurations[0]
			if first.Name != config.Name || first.PoolGUID != config.PoolGUID || first.TopGUID != config.TopGUID || first.VdevGUID != config.VdevGUID || first.State != config.State {
				return zero, ErrDrift
			}
		}
		result.Configurations = append(result.Configurations, config)
		i++
	}
	for _, seen := range covered {
		if !seen {
			return zero, ErrUnavailable
		}
	}
	return result, nil
}

func labelNumber(value string, zero bool) bool {
	if value == "" || (len(value) > 1 && value[0] == '0') {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	n, err := strconv.ParseUint(value, 10, 64)
	return err == nil && (zero || n != 0)
}
func labelString(value string) bool {
	return len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' && !strings.Contains(value[1:len(value)-1], "'")
}
func labelValue(value string) bool {
	return labelString(value) || labelNumber(value, true) || value == "true" || value == "false"
}
