// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type propertyRow struct{ value, received, source string }
type filesystemRows struct {
	fields [4]propertyRow
	seen   [4]bool
}

// ParseProperties accepts ONLY a successful complete fixed zfs get projection:
// -H -p -r -t filesystem -o name,property,value,received,source
// guid,quota,refquota,compression <validated pool name>.
// The caller must separately check command success, bounded capture and a fresh
// stable inventory before/after collection. A syntactically complete stream can
// still omit an entire filesystem; this parser cannot prove native completeness,
// freshness, pool identity or eligibility. It never grants enrollment or writes.
func ParseProperties(data []byte, poolName string) ([]Filesystem, error) {
	if len(data) > maxObservationBytes || bytes.Count(data, []byte{'\n'}) > 4*maxRecords {
		return nil, ErrLimit
	}
	if len(data) == 0 || !utf8.Valid(data) || !validName(poolName) || strings.Contains(poolName, "/") {
		return nil, ErrInvalid
	}
	groups := map[string]*filesystemRows{}
	for len(data) > 0 {
		end := bytes.IndexByte(data, '\n')
		if end > 5*maxFieldBytes+4 {
			return nil, ErrLimit
		}
		if end < 0 || bytes.Count(data[:end], []byte{'\t'}) != 4 {
			return nil, ErrInvalid
		}
		fields := strings.Split(string(data[:end]), "\t")
		for _, field := range fields {
			if len(field) > maxFieldBytes {
				return nil, ErrLimit
			}
			if field == "" {
				return nil, ErrInvalid
			}
			for _, c := range field {
				if unicode.IsControl(c) {
					return nil, ErrInvalid
				}
			}
		}
		name := fields[0]
		if !validName(name) || (name != poolName && !strings.HasPrefix(name, poolName+"/")) {
			return nil, ErrInvalid
		}
		n := -1
		switch fields[1] {
		case "guid":
			n = 0
		case "quota":
			n = 1
		case "refquota":
			n = 2
		case "compression":
			n = 3
		}
		if n < 0 {
			return nil, ErrInvalid
		}
		group := groups[name]
		if group == nil {
			if len(groups) >= maxRecords {
				return nil, ErrLimit
			}
			group = &filesystemRows{}
			groups[name] = group
		}
		if group.seen[n] {
			return nil, ErrInvalid
		}
		group.seen[n] = true
		group.fields[n] = propertyRow{fields[2], fields[3], fields[4]}
		data = data[end+1:]
	}
	// Resolve every native inherited source/parent NAME through this same complete
	// GUID table. Names never become persistent identity, even across row ordering.
	guids := map[string]string{}
	for name, group := range groups {
		for _, seen := range group.seen {
			if !seen {
				return nil, ErrInvalid
			}
		}
		guid := group.fields[0]
		if !validGUID(guid.value) || guid.received != "-" || guid.source != "-" {
			return nil, ErrInvalid
		}
		guids[name] = guid.value
	}
	result := make([]Filesystem, 0, len(groups))
	for name, group := range groups {
		f := Filesystem{GUID: guids[name], Name: name}
		if name != poolName {
			f.ParentGUID = guids[name[:strings.LastIndexByte(name, '/')]]
			if f.ParentGUID == "" {
				return nil, ErrInvalid
			}
		}
		for n, property := range []Property{Quota, RefQuota, Compression} {
			row := group.fields[n+1]
			value := Value{Effective: row.value}
			if property != Compression {
				if row.value == "0" {
					value.Effective = "none"
				} else if !validGUID(row.value) {
					return nil, ErrInvalid
				}
			}
			if row.received != "-" {
				value.ReceivedValue = row.received
				if property != Compression {
					if row.received == "0" {
						value.ReceivedValue = "none"
					} else if !validGUID(row.received) {
						return nil, ErrInvalid
					}
				}
			}
			switch row.source {
			case "default":
				value.Source = Default
			case "local":
				value.Source = Local
				value.SourceGUID = f.GUID
			case "received":
				value.Source = Received
				value.SourceGUID = f.GUID
			default:
				const prefix = "inherited from "
				if !strings.HasPrefix(row.source, prefix) {
					return nil, ErrInvalid
				}
				value.Source = Inherited
				value.SourceGUID = guids[strings.TrimPrefix(row.source, prefix)]
			}
			if !validValue(value, property, f.GUID) {
				return nil, ErrInvalid
			}
			switch property {
			case Quota:
				f.Properties.Quota = value
			case RefQuota:
				f.Properties.RefQuota = value
			case Compression:
				f.Properties.Compression = value
			}
		}
		result = append(result, f)
	}
	if _, err := indexFilesystems(poolName, result); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}
