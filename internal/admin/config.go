// SPDX-License-Identifier: BSD-3-Clause
// Package admin supplies internal M1 configuration and administration contracts.
// It installs no service and executes no native commands.
package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/netip"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gonz0w/horvium/internal/helper"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

var (
	ErrConfig       = errors.New("invalid administration configuration")
	ErrUnauthorized = errors.New("administration authorization required")
	ErrThrottled    = errors.New("administration request throttled")
	ErrCapacity     = errors.New("administration capacity reached")
	ErrClosed       = errors.New("administration stopped")
	ErrState        = errors.New("unsafe or unavailable credential state")
	ErrUnsupported  = errors.New("credential files require FreeBSD")
)

const MaxConfigBytes = 16 << 10

// Config is an internal version-1 format, not a release configuration promise.
// All values are explicit; HTTP callers cannot change safety budgets.
type Config struct {
	Version             int      `json:"version"`
	Listen              string   `json:"listen"`
	AllowedHosts        []string `json:"allowed_hosts"`
	ManagementUID       uint32   `json:"management_uid"`
	SocketGID           uint32   `json:"socket_gid"`
	HelperSocket        string   `json:"helper_socket"`
	StateDirectory      string   `json:"state_directory"`
	MaxBytes            int      `json:"max_bytes"`
	MaxRecords          int      `json:"max_records"`
	MaxFieldBytes       int      `json:"max_field_bytes"`
	MaxXMLDepth         int      `json:"max_xml_depth"`
	MaxXMLElements      int      `json:"max_xml_elements"`
	MaxFrameBytes       int      `json:"max_frame_bytes"`
	MaxJSONBytes        int      `json:"max_json_bytes"`
	MaxTransportRecords int      `json:"max_transport_records"`
	MaxJSONDepth        int      `json:"max_json_depth"`
	IntervalSeconds     int      `json:"interval_seconds"`
	TimeoutSeconds      int      `json:"timeout_seconds"`
	StaleSeconds        int      `json:"stale_seconds"`
	MaxReaders          int      `json:"max_readers"`
	MaxHelperClients    int      `json:"max_helper_clients"`
}

// ParseConfig bounds bytes before decoding, requires every exact key once, and
// rejects unknown/case-variant keys, nulls, trailing documents and bad types.
func ParseConfig(r io.Reader) (Config, error) {
	b, e := io.ReadAll(io.LimitReader(r, MaxConfigBytes+1))
	if e != nil || len(b) > MaxConfigBytes || !utf8.Valid(b) {
		return Config{}, ErrConfig
	}
	d := json.NewDecoder(bytes.NewReader(b))
	tok, e := d.Token()
	if e != nil || tok != json.Delim('{') {
		return Config{}, ErrConfig
	}
	fields := map[string]bool{}
	typ := reflect.TypeFor[Config]()
	for i := 0; i < typ.NumField(); i++ {
		fields[typ.Field(i).Tag.Get("json")] = false
	}
	for d.More() {
		tok, e = d.Token()
		key, ok := tok.(string)
		seen, known := fields[key]
		if e != nil || !ok || !known || seen {
			return Config{}, ErrConfig
		}
		fields[key] = true
		var raw json.RawMessage
		if d.Decode(&raw) != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return Config{}, ErrConfig
		}
	}
	if tok, e = d.Token(); e != nil || tok != json.Delim('}') {
		return Config{}, ErrConfig
	}
	if _, e = d.Token(); e != io.EOF {
		return Config{}, ErrConfig
	}
	for _, seen := range fields {
		if !seen {
			return Config{}, ErrConfig
		}
	}
	var c Config
	if json.Unmarshal(b, &c) != nil || c.Validate() != nil {
		return Config{}, ErrConfig
	}
	return c, nil
}
func (c Config) Validate() error {
	host, port, e := net.SplitHostPort(c.Listen)
	ip, ie := netip.ParseAddr(host)
	if c.Version != 1 || e != nil || ie != nil || !ip.IsLoopback() || ip.Is4In6() || ip.Zone() != "" || !validPort(port) || c.Listen != net.JoinHostPort(ip.String(), port) {
		return ErrConfig
	}
	if len(c.AllowedHosts) < 1 || len(c.AllowedHosts) > 16 {
		return ErrConfig
	}
	seen := map[string]bool{}
	for _, h := range c.AllowedHosts {
		if !validHost(h) || seen[h] {
			return ErrConfig
		}
		seen[h] = true
	}
	if c.ManagementUID == 0 || c.ManagementUID == ^uint32(0) || c.SocketGID == 0 || c.SocketGID == ^uint32(0) || !cleanPath(c.HelperSocket) || len(c.HelperSocket) > 103 || !cleanPath(c.StateDirectory) {
		return ErrConfig
	}
	if c.InventoryLimits().Validate() != nil || c.MaxFrameBytes != 64<<10 || c.MaxJSONBytes < 1 || c.MaxTransportRecords < 2 || c.MaxJSONDepth < 1 || c.MaxHelperClients < 1 || c.MaxHelperClients > 64 {
		return ErrConfig
	}
	if c.IntervalSeconds < 1 || c.IntervalSeconds > 86400 || c.TimeoutSeconds < 1 || c.TimeoutSeconds > 300 || c.StaleSeconds < 1 || c.StaleSeconds > 86400 || c.MaxReaders < 1 || c.MaxReaders > 64 {
		return ErrConfig
	}
	return nil
}
func validPort(s string) bool {
	n, e := strconv.Atoi(s)
	return e == nil && n > 0 && n <= 65535 && strconv.Itoa(n) == s
}
func validHost(s string) bool {
	if len(s) > 260 || strings.ContainsAny(s, "/\\?#@ \t\r\n") {
		return false
	}
	host, port, e := net.SplitHostPort(s)
	if e != nil || !validPort(port) {
		return false
	}
	if ip, e := netip.ParseAddr(host); e == nil {
		return ip.Zone() == "" && !ip.Is4In6() && s == net.JoinHostPort(ip.String(), port)
	}
	if len(host) < 1 || len(host) > 253 || host != strings.ToLower(host) {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}
func cleanPath(s string) bool {
	if len(s) > 4096 || !utf8.ValidString(s) || !filepath.IsAbs(s) || filepath.Clean(s) != s || s == "/" {
		return false
	}
	for _, r := range s {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
func (c Config) InventoryLimits() inventory.Limits {
	return inventory.Limits{MaxBytes: c.MaxBytes, MaxRecords: c.MaxRecords, MaxFieldBytes: c.MaxFieldBytes, MaxXMLDepth: c.MaxXMLDepth, MaxXMLElements: c.MaxXMLElements}
}
func (c Config) HelperConfig() helper.Config {
	return helper.Config{Socket: c.HelperSocket, ManagementUID: c.ManagementUID, SocketGID: c.SocketGID, Limits: helper.Limits{MaxBytes: c.MaxJSONBytes, MaxRecords: c.MaxTransportRecords, MaxFieldBytes: c.MaxFieldBytes, MaxDepth: c.MaxJSONDepth}, Timeout: time.Duration(c.TimeoutSeconds) * time.Second, MaxClients: c.MaxHelperClients}
}
func (c Config) ObservationConfig() observation.Config {
	return observation.Config{Interval: time.Duration(c.IntervalSeconds) * time.Second, Timeout: time.Duration(c.TimeoutSeconds) * time.Second, StaleAfter: time.Duration(c.StaleSeconds) * time.Second, Limits: c.InventoryLimits(), MaxReaders: c.MaxReaders}
}
