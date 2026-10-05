// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"
)

var poolProperties = [...]string{"guid", "load_guid", "health", "readonly", "bootfs", "altroot", "cachefile", "compatibility", "multihost"}

func poolValues(p PoolState) [7]PoolValue {
	return [7]PoolValue{p.Health, p.Readonly, p.BootFS, p.AltRoot, p.CacheFile, p.Compatibility, p.MultiHost}
}

// ParsePoolState accepts only a successful fixed single-pool projection:
// zpool get -H -p -o name,property,value,source
// guid,load_guid,health,readonly,bootfs,altroot,cachefile,compatibility,multihost <pool>.
// It neither checks feature files nor establishes topology/system/device safety.
func ParsePoolState(data []byte, poolName string) (PoolState, error) {
	if len(data) > maxPlanBytes || bytes.Count(data, []byte{'\n'}) > len(poolProperties) {
		return PoolState{}, ErrLimit
	}
	if len(data) == 0 || !utf8.Valid(data) || !validName(poolName) || strings.Contains(poolName, "/") {
		return PoolState{}, ErrInvalid
	}
	var values [9]PoolValue
	var seen [9]bool
	for len(data) > 0 {
		end := bytes.IndexByte(data, '\n')
		if end > 4*maxFieldBytes+3 {
			return PoolState{}, ErrLimit
		}
		if end < 0 || bytes.Count(data[:end], []byte{'\t'}) != 3 {
			return PoolState{}, ErrInvalid
		}
		fields := strings.Split(string(data[:end]), "\t")
		for _, field := range fields {
			if len(field) > maxFieldBytes {
				return PoolState{}, ErrLimit
			}
			if field == "" {
				return PoolState{}, ErrInvalid
			}
			for _, c := range field {
				if unicode.IsControl(c) {
					return PoolState{}, ErrInvalid
				}
			}
		}
		if fields[0] != poolName {
			return PoolState{}, ErrInvalid
		}
		n := -1
		for at, name := range poolProperties {
			if name == fields[1] {
				n = at
				break
			}
		}
		if n < 0 || seen[n] {
			return PoolState{}, ErrInvalid
		}
		seen[n] = true
		values[n] = PoolValue{fields[2], Source(fields[3])}
		data = data[end+1:]
	}
	for _, present := range seen {
		if !present {
			return PoolState{}, ErrInvalid
		}
	}
	p := PoolState{GUID: values[0].Effective, LoadGUID: values[1].Effective, Name: poolName, GUIDSource: values[0].Source, LoadGUIDSource: values[1].Source,
		Health: values[2], Readonly: values[3], BootFS: values[4], AltRoot: values[5], CacheFile: values[6], Compatibility: values[7], MultiHost: values[8]}
	if err := validatePoolState(p); err != nil {
		return PoolState{}, err
	}
	return p, nil
}

func poolSource(s Source) bool { return s == NoSource || s == Default || s == Local }

func validatePoolState(p PoolState) error {
	if err := preflight(Policy{}, Observation{Pool: p}); err != nil {
		return err
	}
	if !validName(p.Name) || strings.Contains(p.Name, "/") || !validGUID(p.GUID) || !validGUID(p.LoadGUID) || !poolSource(p.GUIDSource) || !poolSource(p.LoadGUIDSource) {
		return ErrInvalid
	}
	for _, v := range poolValues(p) {
		if v.Effective == "" || !utf8.ValidString(v.Effective) || !poolSource(v.Source) {
			return ErrInvalid
		}
		for _, c := range v.Effective {
			if unicode.IsControl(c) {
				return ErrInvalid
			}
		}
	}
	switch p.Health.Effective {
	case "ONLINE", "DEGRADED", "FAULTED", "OFFLINE", "REMOVED", "UNAVAIL", "SUSPENDED", "SPLIT", "UNKNOWN":
	default:
		return ErrInvalid
	}
	if p.MultiHost.Effective != "on" && p.MultiHost.Effective != "off" {
		return ErrInvalid
	}
	if p.Readonly.Effective != "on" && p.Readonly.Effective != "off" {
		return ErrInvalid
	}
	boot := p.BootFS.Effective
	if boot != "-" && (!validName(boot) || (boot != p.Name && !strings.HasPrefix(boot, p.Name+"/"))) {
		return ErrInvalid
	}
	if p.AltRoot.Effective != "-" && !strings.HasPrefix(p.AltRoot.Effective, "/") {
		return ErrInvalid
	}
	if p.CacheFile.Effective != "-" && p.CacheFile.Effective != "none" && !strings.HasPrefix(p.CacheFile.Effective, "/") {
		return ErrInvalid
	}
	// Compatibility filenames are observed text, never read or resolved here.
	// Do not reinterpret their presence as an installed/qualified feature profile.
	for _, part := range strings.Split(p.Compatibility.Effective, ",") {
		if part == "" || strings.TrimSpace(part) != part {
			return ErrInvalid
		}
	}
	return nil
}

// VerifyPoolState supplies identity/import-incarnation and coarse refusal checks
// only. An unset bootfs and ONLINE health are not system/topology/feature proofs.
func VerifyPoolState(guid, name string, pool PoolState) error {
	if pool == (PoolState{}) {
		return ErrUnavailable
	}
	if err := validatePoolState(pool); err != nil {
		return err
	}
	if !validGUID(guid) || !validName(name) || strings.Contains(name, "/") {
		return ErrInvalid
	}
	if pool.GUID != guid || pool.Name != name {
		return ErrDrift
	}
	if pool.Readonly.Effective != "off" || pool.BootFS.Effective != "-" || pool.Health.Effective != "ONLINE" || pool.MultiHost.Effective != "off" {
		return ErrUnavailable
	}
	return nil
}
