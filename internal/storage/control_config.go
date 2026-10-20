// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxControlConfigBytes = 16 << 10

// ControlTargets contains path metadata only, never a retained owner or receipt.
type ControlTargets struct {
	JournalDirectory string `json:"journal_directory"`
	LedgerDirectory  string `json:"ledger_directory"`
}

// ValidControlPath checks the admitted path grammar without normalization.
func ValidControlPath(path string) bool {
	return len(path) <= 4096 && utf8.ValidString(path) && filepath.IsAbs(path) && path != "/" && filepath.Clean(path) == path && strings.IndexFunc(path, func(r rune) bool { return r < 32 || r == 127 }) < 0
}

func controlPathsOverlap(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}

// ParseControlConfig parses only the separate storage-control selection format.
// It does not open objects, enroll storage or establish native custody.
func ParseControlConfig(r io.Reader) (ControlTargets, error) {
	if r == nil {
		return ControlTargets{}, ErrInvalid
	}
	data, e := io.ReadAll(io.LimitReader(r, maxControlConfigBytes+1))
	if e != nil || len(data) == 0 || len(data) > maxControlConfigBytes || !utf8.Valid(data) || !validControlUnicodeEscapes(data) {
		return ControlTargets{}, ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(data))
	if token, e := d.Token(); e != nil || token != json.Delim('{') {
		return ControlTargets{}, ErrInvalid
	}
	seen := make(map[string]bool, 4)
	var targets ControlTargets
	for d.More() {
		token, e := d.Token()
		key, ok := token.(string)
		if e != nil || !ok || seen[key] {
			return ControlTargets{}, ErrInvalid
		}
		seen[key] = true
		var raw json.RawMessage
		if d.Decode(&raw) != nil {
			return ControlTargets{}, ErrInvalid
		}
		var value string
		switch key {
		case "version":
			if string(bytes.TrimSpace(raw)) != "1" {
				return ControlTargets{}, ErrInvalid
			}
			continue
		case "format", "journal_directory", "ledger_directory":
			if bytes.Equal(raw, []byte("null")) || json.Unmarshal(raw, &value) != nil {
				return ControlTargets{}, ErrInvalid
			}
		default:
			return ControlTargets{}, ErrInvalid
		}
		switch key {
		case "format":
			if value != "horvium-storage-control" {
				return ControlTargets{}, ErrInvalid
			}
		case "journal_directory":
			targets.JournalDirectory = value
		case "ledger_directory":
			targets.LedgerDirectory = value
		}
	}
	if token, e := d.Token(); e != nil || token != json.Delim('}') {
		return ControlTargets{}, ErrInvalid
	}
	var trailing json.RawMessage
	if d.Decode(&trailing) != io.EOF || len(seen) != 4 || !ValidControlPath(targets.JournalDirectory) || !ValidControlPath(targets.LedgerDirectory) || controlPathsOverlap(targets.JournalDirectory, targets.LedgerDirectory) {
		return ControlTargets{}, ErrInvalid
	}
	return targets, nil
}

// encoding/json repairs unpaired UTF-16 surrogate escapes. Selection bytes
// cannot acquire a different path through that replacement normalization.
func validControlUnicodeEscapes(data []byte) bool {
	for i := 0; i < len(data); i++ {
		if data[i] != '\\' {
			continue
		}
		i++
		if i >= len(data) {
			return false
		}
		if data[i] != 'u' {
			continue
		}
		if i+4 >= len(data) {
			return false
		}
		n, e := strconv.ParseUint(string(data[i+1:i+5]), 16, 16)
		if e != nil {
			return false
		}
		i += 4
		if n >= 0xdc00 && n <= 0xdfff {
			return false
		}
		if n >= 0xd800 && n <= 0xdbff {
			if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
				return false
			}
			low, e := strconv.ParseUint(string(data[i+3:i+7]), 16, 16)
			if e != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}
