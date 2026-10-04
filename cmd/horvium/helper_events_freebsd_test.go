//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

type nativeHelperEventSink struct {
	sync.Mutex
	bytes.Buffer
}

func (s *nativeHelperEventSink) Write(b []byte) (int, error) {
	s.Lock()
	defer s.Unlock()
	return s.Buffer.Write(b)
}
func (s *nativeHelperEventSink) records(t *testing.T) []map[string]any {
	t.Helper()
	s.Lock()
	defer s.Unlock()
	var rows []map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(s.Buffer.Bytes()), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var row map[string]any
		if json.Unmarshal(line, &row) != nil {
			t.Fatal("invalid built-helper event")
		}
		if len(row) != 8 {
			t.Fatal("unexpected exported event fields")
		}
		rows = append(rows, row)
	}
	return rows
}
func TestNativeBuiltHelperDenialLogging(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" {
		t.Skip("explicit disposable native opt-in required")
	}
	if os.Getuid() != 0 {
		t.Fatal("fixture parent requires root")
	}
	c, path, _ := nativeRuntimeConfig(t)
	built, err := filepath.Abs("../../.tools/horvium")
	if err != nil {
		t.Fatal("built fixture path")
	}
	binary := filepath.Join(filepath.Dir(path), "horvium")
	copyPerformanceExecutable(t, built, binary)
	sink := &nativeHelperEventSink{}
	process := startNativeProcessWithStderr(t, binary, []string{"helper", "-config", path}, 0, 0, sink)
	nativeWait(t, func() bool { info, err := os.Stat(c.HelperSocket); return err == nil && info.Mode().Perm() == 0660 })
	// Root has filesystem access but no exact-UID exemption. Send no payload:
	// kernel authorization must reject the connection before request reads.
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: c.HelperSocket, Net: "unix"})
	if err != nil {
		t.Fatal("denial connection")
	}
	conn.SetDeadline(time.Now().Add(time.Second))
	var b [1]byte
	n, err := conn.Read(b[:])
	conn.Close()
	if n != 0 || err != io.EOF {
		t.Fatal("root denial did not close before reading input")
	}
	nativeWait(t, func() bool {
		for _, row := range sink.records(t) {
			if row["operation"] == "helper" && row["section"] == "authentication" && row["outcome"] == "rejected" && row["error_class"] == "permission" {
				return true
			}
		}
		return false
	})
	process.stop(t, syscall.SIGTERM, true)
	if _, err := os.Lstat(c.HelperSocket); !os.IsNotExist(err) {
		t.Fatal("denial fixture socket leaked")
	}
}
