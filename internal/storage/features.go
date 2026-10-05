// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxFeatures = 128
const commonProfileSHA256 = "8b5a635bf7609a0f521b08fe0e389c651dd6a266d33bd53a2e253b99089c3bf7"

func featureNames(format string) ([]string, error) {
	names := commonFeatureNames
	switch format {
	case "freebsd-14.5":
	case "freebsd-15.1":
		names += " " + newerFeatureNames
	default:
		return nil, ErrUnavailable
	}
	return strings.Fields(names), nil
}

// CommonFeatureProfile verifies exact approved bytes only. It neither opens a
// path nor proves installed ownership, lookup precedence or import compatibility.
func CommonFeatureProfile(data []byte) ([]string, error) {
	if len(data) != 584 {
		return nil, ErrUnavailable
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != commonProfileSHA256 {
		return nil, ErrUnavailable
	}
	return strings.Fields(commonFeatureNames), nil
}

func featureToken(s string, guid bool) bool {
	if len(s) == 0 || len(s) > 255 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || guid && (c == '.' || c == ':' || c == '-')) {
			return false
		}
	}
	if guid {
		parts := strings.Split(s, ":")
		return len(parts) == 2 && parts[0] != "" && parts[1] != ""
	}
	return true
}
func validFeature(v PoolFeature) bool {
	if v.Source != Local {
		return false
	}
	if strings.HasPrefix(v.Name, "feature@") {
		return featureToken(strings.TrimPrefix(v.Name, "feature@"), false) && (v.State == "disabled" || v.State == "enabled" || v.State == "active")
	}
	if strings.HasPrefix(v.Name, "unsupported@") {
		return featureToken(strings.TrimPrefix(v.Name, "unsupported@"), true) && (v.State == "inactive" || v.State == "readonly")
	}
	return false
}

// ParsePoolFeatures consumes the complete successful `zpool get -H -p -o
// name,property,value,source all <pool>` projection. Non-feature properties are
// shape-checked but supply no eligibility evidence; GUID and all qualified
// format feature names are mandatory. Unknown/unsupported features are retained.
func ParsePoolFeatures(data []byte, name, format string) (PoolFeatures, error) {
	if len(data) > maxPlanBytes || bytes.Count(data, []byte{'\n'}) > 256 {
		return PoolFeatures{}, ErrLimit
	}
	if len(data) == 0 || !utf8.Valid(data) || !validName(name) || strings.Contains(name, "/") {
		return PoolFeatures{}, ErrInvalid
	}
	if len(format) > maxFieldBytes {
		return PoolFeatures{}, ErrLimit
	}
	names, err := featureNames(format)
	if err != nil {
		return PoolFeatures{}, err
	}
	f := PoolFeatures{PoolName: name, Format: format}
	seen := make(map[string]bool)
	for len(data) > 0 {
		end := bytes.IndexByte(data, '\n')
		if end > 4*maxFieldBytes+3 {
			return PoolFeatures{}, ErrLimit
		}
		if end < 0 || bytes.Count(data[:end], []byte{'\t'}) != 3 {
			return PoolFeatures{}, ErrInvalid
		}
		fields := strings.Split(string(data[:end]), "\t")
		for _, field := range fields {
			if len(field) > maxFieldBytes {
				return PoolFeatures{}, ErrLimit
			}
			if field == "" {
				return PoolFeatures{}, ErrInvalid
			}
			for _, c := range field {
				if unicode.IsControl(c) {
					return PoolFeatures{}, ErrInvalid
				}
			}
		}
		if fields[0] != name || seen[fields[1]] {
			return PoolFeatures{}, ErrInvalid
		}
		seen[fields[1]] = true
		switch {
		case fields[1] == "guid":
			if !validGUID(fields[2]) || fields[3] != "-" {
				return PoolFeatures{}, ErrInvalid
			}
			f.PoolGUID = fields[2]
		case strings.HasPrefix(fields[1], "feature@") || strings.HasPrefix(fields[1], "unsupported@"):
			v := PoolFeature{fields[1], fields[2], Source(fields[3])}
			if !validFeature(v) {
				return PoolFeatures{}, ErrInvalid
			}
			if len(f.Values) >= maxFeatures {
				return PoolFeatures{}, ErrLimit
			}
			f.Values = append(f.Values, v)
		default:
			// No unknown namespace can masquerade as an ordinary ignored pool property.
			if strings.Contains(fields[1], "@") || !poolSource(Source(fields[3])) {
				return PoolFeatures{}, ErrInvalid
			}
		}
		data = data[end+1:]
	}
	if f.PoolGUID == "" {
		return PoolFeatures{}, ErrUnavailable
	}
	for _, n := range names {
		if !seen["feature@"+n] {
			return PoolFeatures{}, ErrUnavailable
		}
	}
	sort.Slice(f.Values, func(i, j int) bool { return f.Values[i].Name < f.Values[j].Name })
	return f, nil
}

// VerifyPoolFeatures only recognizes a complete qualified feature projection;
// known enabled/active features are preserved, never disabled or upgraded. This
// does not establish provider/topology/system safety or receiver compatibility.
func VerifyPoolFeatures(guid, name string, f PoolFeatures) error {
	if f.PoolGUID == "" && f.PoolName == "" && f.Format == "" && len(f.Values) == 0 {
		return ErrUnavailable
	}
	if err := preflight(Policy{}, Observation{Features: f}); err != nil {
		return err
	}
	if !validGUID(guid) || !validGUID(f.PoolGUID) || !validName(name) || strings.Contains(name, "/") || !validName(f.PoolName) || strings.Contains(f.PoolName, "/") {
		return ErrInvalid
	}
	if f.PoolGUID != guid || f.PoolName != name {
		return ErrDrift
	}
	names, err := featureNames(f.Format)
	if err != nil {
		return err
	}
	expected := make(map[string]bool, len(names))
	for _, n := range names {
		expected["feature@"+n] = true
	}
	seen := make(map[string]bool, len(f.Values))
	unknown := false
	for _, v := range f.Values {
		if !validFeature(v) || seen[v.Name] {
			return ErrInvalid
		}
		seen[v.Name] = true
		if !expected[v.Name] {
			unknown = true
		}
	}
	if unknown || len(seen) != len(expected) {
		return ErrUnavailable
	}
	return nil
}
