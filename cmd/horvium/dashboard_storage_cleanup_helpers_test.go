// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"errors"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type dashboardCleanupIdentity struct {
	Pool, Unit, File, GUID string
	Created                bool
}

type dashboardCleanupReceipt struct {
	PoolCreated bool `json:"pool_created"`
	PoolRemoved bool `json:"pool_removed"`
	MDRemoved   bool `json:"md_removed"`
}

// Test-controller policy only. A failed observation is uncertainty, never absence.
// Every command uses the identity captured during this disposable fixture's setup.
func dashboardCleanupStorage(id dashboardCleanupIdentity, run func(string, ...string) ([]byte, error)) (dashboardCleanupReceipt, error) {
	fail := func() (dashboardCleanupReceipt, error) {
		return dashboardCleanupReceipt{}, errors.New("owned storage cleanup uncertain; discard guest")
	}
	if run == nil || !regexp.MustCompile(`^horvium_ui_[0-9a-f]{16}$`).MatchString(id.Pool) || !regexp.MustCompile(`^md(0|[1-9][0-9]{0,9})$`).MatchString(id.Unit) || !filepath.IsAbs(id.File) || filepath.Clean(id.File) != id.File || filepath.Base(id.File) != "disk.img" || len(id.File) > 4096 || strings.ContainsAny(id.File, "\x00\r\n\t") {
		return fail()
	}
	if id.Created {
		guid, err := strconv.ParseUint(id.GUID, 10, 64)
		if err != nil || guid == 0 || strconv.FormatUint(guid, 10) != id.GUID {
			return fail()
		}
	} else if id.GUID != "" {
		return fail()
	}
	command := func(path string, args ...string) (string, error) {
		data, err := run(path, args...)
		if err != nil || len(data) > 65536 {
			return "", errors.New("cleanup command unavailable")
		}
		return string(data), nil
	}
	backing := func() bool {
		data, err := command("/sbin/mdconfig", "-l", "-n", "-f", id.File)
		return err == nil && (data == strings.TrimPrefix(id.Unit, "md")+"\n" || data == strings.TrimPrefix(id.Unit, "md")+" \n")
	}
	if !backing() {
		return fail()
	}
	if id.Created {
		data, err := command("/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", id.Pool)
		if err != nil || data != id.GUID+"\n" {
			return fail()
		}
		data, err = command("/sbin/zpool", "destroy", id.Pool)
		if err != nil || data != "" {
			return fail()
		}
		data, err = command("/sbin/zpool", "list", "-H", "-o", "name")
		if err != nil || !dashboardCleanupAbsent(data, id.Pool, false) {
			return fail()
		}
	}
	// Recheck immediately before detaching; pool destruction may take time.
	if !backing() {
		return fail()
	}
	data, err := command("/sbin/mdconfig", "-d", "-u", strings.TrimPrefix(id.Unit, "md"))
	if err != nil || data != "" {
		return fail()
	}
	data, err = command("/sbin/mdconfig", "-l")
	if err != nil || !dashboardCleanupAbsent(data, id.Unit, true) {
		return fail()
	}
	return dashboardCleanupReceipt{PoolCreated: id.Created, PoolRemoved: id.Created, MDRemoved: true}, nil
}

func dashboardCleanupAbsent(data, owned string, devices bool) bool {
	if len(data) > 65536 || strings.ContainsAny(data, "\x00\r") || (data != "" && !strings.HasSuffix(data, "\n")) {
		return false
	}
	pattern := regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]{0,254}$`)
	if devices {
		pattern = regexp.MustCompile(`^md(0|[1-9][0-9]{0,9})$`)
	}
	var fields []string
	if devices {
		fields = strings.Fields(strings.TrimSuffix(data, "\n"))
	} else if data != "" {
		fields = strings.Split(strings.TrimSuffix(data, "\n"), "\n")
	}
	if len(fields) > 4096 {
		return false
	}
	seen := map[string]bool{}
	for _, field := range fields {
		if !pattern.MatchString(field) || field == owned || seen[field] {
			return false
		}
		seen[field] = true
	}
	return true
}

// Prevent an untrusted command from allocating or retaining an unbounded inventory.
type dashboardCleanupOutput struct {
	data     []byte
	overflow bool
}

func (w *dashboardCleanupOutput) Write(p []byte) (int, error) {
	if len(p) > 65536-len(w.data) {
		w.overflow = true
		return 0, io.ErrShortBuffer
	}
	w.data = append(w.data, p...)
	return len(p), nil
}
