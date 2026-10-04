// SPDX-License-Identifier: BSD-3-Clause
package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func testConfig() Config {
	return Config{Version: 1, Listen: "127.0.0.1:8080", AllowedHosts: []string{"127.0.0.1:8080", "localhost:8080"}, ManagementUID: 65534, SocketGID: 65534, HelperSocket: "/var/run/horvium/inventory.sock", StateDirectory: "/var/db/horvium", MaxBytes: 64 << 20, MaxRecords: 100000, MaxFieldBytes: 4096, MaxXMLDepth: 32, MaxXMLElements: 250000, MaxFrameBytes: 64 << 10, MaxJSONBytes: 64 << 20, MaxTransportRecords: 100000, MaxJSONDepth: 32, IntervalSeconds: 30, TimeoutSeconds: 45, StaleSeconds: 90, MaxReaders: 16, MaxHelperClients: 8}
}
func configJSON(t *testing.T) []byte {
	t.Helper()
	b, e := json.Marshal(testConfig())
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestStrictConfiguration(t *testing.T) {
	b := configJSON(t)
	c, e := ParseConfig(bytes.NewReader(b))
	if e != nil || c.Listen != "127.0.0.1:8080" {
		t.Fatal("valid explicit config refused")
	}
	for _, bad := range []string{
		strings.Replace(string(b), `"version":1`, `"version":2`, 1),
		strings.Replace(string(b), `"version":1`, `"Version":1`, 1),
		strings.Replace(string(b), `"version":1`, `"version":1,"version":1`, 1),
		strings.Replace(string(b), `"version":1,`, "", 1),
		strings.Replace(string(b), `"version":1`, `"version":null`, 1),
		strings.Replace(string(b), `"version":1`, `"version":1,"future_option":true`, 1),
		string(b) + `{}`, string(b) + strings.Repeat(" ", MaxConfigBytes),
	} {
		if _, e := ParseConfig(strings.NewReader(bad)); !errors.Is(e, ErrConfig) {
			t.Fatal("ambiguous/oversized configuration accepted")
		}
	}
}
func TestConfigRejectsUnsafeBoundsAndAddresses(t *testing.T) {
	for _, edit := range []func(*Config){
		func(c *Config) { c.Listen = "0.0.0.0:8080" }, func(c *Config) { c.Listen = ":8080" }, func(c *Config) { c.Listen = "localhost:8080" }, func(c *Config) { c.Listen = "192.0.2.1:8080" }, func(c *Config) { c.Listen = "[::]:8080" }, func(c *Config) { c.Listen = "[::ffff:127.0.0.1]:8080" }, func(c *Config) { c.Listen = "127.0.0.1:0" },
		func(c *Config) { c.AllowedHosts = nil }, func(c *Config) { c.AllowedHosts = []string{"*.example:8080"} }, func(c *Config) { c.AllowedHosts = []string{"localhost:8080", "localhost:8080"} }, func(c *Config) { c.AllowedHosts = []string{"http://localhost:8080"} },
		func(c *Config) { c.ManagementUID = 0 }, func(c *Config) { c.SocketGID = 0 }, func(c *Config) { c.StateDirectory = "/var/db/../db/horvium" }, func(c *Config) { c.HelperSocket = "relative" }, func(c *Config) { c.HelperSocket = strings.Repeat("/a", 70) },
		func(c *Config) { c.MaxRecords = 0 }, func(c *Config) { c.MaxFrameBytes = 0 }, func(c *Config) { c.TimeoutSeconds = 301 }, func(c *Config) { c.MaxReaders = 65 }, func(c *Config) { c.MaxHelperClients = 65 }, func(c *Config) { c.IntervalSeconds = 0 }, func(c *Config) { c.StaleSeconds = 0 },
	} {
		c := testConfig()
		edit(&c)
		if !errors.Is(c.Validate(), ErrConfig) {
			t.Fatal("unsafe configuration accepted")
		}
	}
	c := testConfig()
	c.Listen = "[::1]:8080"
	c.AllowedHosts = []string{"[::1]:8080"}
	if c.Validate() != nil {
		t.Fatal("IPv6 loopback refused")
	}
}

func FuzzConfiguration(f *testing.F) {
	b, _ := json.Marshal(testConfig())
	f.Add(b)
	f.Add([]byte(`{"version":1,"version":1}`))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, b []byte) {
		c, e := ParseConfig(bytes.NewReader(b))
		if e == nil && c.Validate() != nil {
			t.Fatal("parser published unsafe configuration")
		}
		if e != nil && !errors.Is(e, ErrConfig) {
			t.Fatal("unsafe error detail")
		}
	})
}
