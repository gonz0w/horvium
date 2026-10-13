//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"debug/buildinfo"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/gonz0w/horvium/internal/admin"
	"golang.org/x/sys/unix"
)

// Qualified amd64 layout already used by the native supervisor. Own process only;
// never signal saved PIDs. Adopted zombies must be reaped, not called removed.
type dashboardReaperStatus struct {
	Flags, Children, Descendants uint32
	Reaper, PID                  int32
	Pad                          [15]uint32
}

func dashboardProcctl(command uintptr, data unsafe.Pointer) error {
	if runtime.GOARCH != "amd64" {
		return errors.New("unsupported dashboard reaper ABI")
	}
	_, _, e := unix.Syscall6(unix.SYS_PROCCTL, 0, 0, command, uintptr(data), 0, 0)
	if e != 0 {
		return e
	}
	return nil
}
func dashboardReaperEmpty() error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for {
			var status unix.WaitStatus
			pid, err := unix.Wait4(-1, &status, unix.WNOHANG, nil)
			if err == syscall.EINTR {
				continue
			}
			if err != nil && err != syscall.ECHILD {
				return err
			}
			if pid <= 0 {
				break
			}
		}
		var s dashboardReaperStatus
		if err := dashboardProcctl(4, unsafe.Pointer(&s)); err != nil {
			return err
		}
		if s.Flags != 1 || s.Reaper != int32(os.Getpid()) {
			return errors.New("original dashboard reaper lost")
		}
		if dashboardEmptyReaper(s.Flags, s.Children, s.Descendants, int(s.Reaper), os.Getpid()) {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return errors.New("dashboard descendants remain; discard guest")
}

type dashboardGuestOwner struct {
	artifacts       dashboardArtifactObservation
	files           map[string]*dashboardOriginalFile
	directories     map[string]string
	sockets         map[string]string
	storageCleanups int
}

func newDashboardGuestOwner(t *testing.T) *dashboardGuestOwner {
	t.Helper()
	if err := dashboardProcctl(2, nil); err != nil {
		t.Fatal("dashboard controller reaper acquisition failed")
	}
	owner := &dashboardGuestOwner{files: map[string]*dashboardOriginalFile{}, directories: map[string]string{}, sockets: map[string]string{}}
	t.Cleanup(func() {
		// Successful parent publication already observed an empty reaper. On any
		// failure still reap departed children; do not kill arbitrary descendants.
		if err := dashboardReaperEmpty(); err != nil {
			t.Error("dashboard final descendant cleanup uncertain")
			return
		}
		if err := dashboardProcctl(3, nil); err != nil {
			t.Error("dashboard reaper release failed")
		}
	})
	owner.artifacts = observeDashboardParentArtifacts(t)
	return owner
}
func observeDashboardParentArtifacts(t *testing.T) dashboardArtifactObservation {
	t.Helper()
	root, err := os.OpenRoot("../../.tools")
	if err != nil {
		t.Fatal("original dashboard artifacts unavailable")
	}
	exe, ee := os.Executable()
	observed, oe := dashboardObserveArtifacts(root, exe, buildinfo.Read)
	ce := root.Close()
	if ee != nil || oe != nil || ce != nil {
		t.Fatal("original dashboard controller/artifacts changed")
	}
	return observed
}
func (o *dashboardGuestOwner) copy(t *testing.T, role, from, to string) {
	t.Helper()
	if err := dashboardCopyOriginalArtifact(from, to, o.artifacts.Artifacts[role]); err != nil {
		t.Fatal("original dashboard launch artifact changed")
	}
}
func (o *dashboardGuestOwner) original(t *testing.T, p *dashboardPreparedFixture) {
	t.Helper()
	if _, duplicate := o.files[p.mode]; duplicate {
		t.Fatal("duplicate original dashboard declaration")
	}
	file, err := dashboardSaveOriginal("../../.tools", "dashboard-original-"+p.mode+".json", dashboardOriginalFixture{1, p.mode, p.fixture})
	if err != nil {
		t.Fatal("original dashboard fixture persistence failed")
	}
	o.files[p.mode] = file
}
func (o *dashboardGuestOwner) pair(t *testing.T, mode, path string, c admin.Config) {
	t.Helper()
	if _, duplicate := o.directories[mode]; duplicate || path == "" || c.HelperSocket == "" {
		t.Fatal("duplicate or incomplete original dashboard pair")
	}
	o.directories[mode] = filepath.Dir(path)
	o.sockets[mode] = c.HelperSocket
}
func (o *dashboardGuestOwner) storage(r dashboardCleanupReceipt) {
	if r.PoolCreated && r.PoolRemoved && r.MDRemoved {
		o.storageCleanups++
	}
}
func (o *dashboardGuestOwner) finish(t *testing.T, record *dashboardParentControlReceipt) {
	t.Helper()
	expectedPairs, expectedStorage := 3, 0
	if record.Parent == "TestNativeReaderResourceBudgets" {
		expectedPairs, expectedStorage = 2, 1
	}
	if len(o.files) != 2 || len(o.directories) != expectedPairs || len(o.sockets) != expectedPairs || o.storageCleanups != expectedStorage {
		t.Fatal("incomplete original dashboard cleanup observations")
	}
	for mode, file := range o.files {
		if _, exists := record.Modes[mode]; !exists || file.check() != nil {
			t.Fatal("original dashboard fixture changed")
		}
	}
	for mode, directory := range o.directories {
		if _, err := os.Lstat(directory); !os.IsNotExist(err) {
			t.Fatal("dashboard temporary directory remains")
		}
		if _, err := os.Lstat(o.sockets[mode]); !os.IsNotExist(err) {
			t.Fatal("dashboard socket remains")
		}
	}
	if err := dashboardReaperEmpty(); err != nil {
		t.Fatal("dashboard descendants remain after original cleanup")
	}
	if !reflect.DeepEqual(observeDashboardParentArtifacts(t), o.artifacts) {
		t.Fatal("original dashboard artifacts changed during parent")
	}
	cleanup := dashboardGuestCleanup{true, true, true, o.storageCleanups, len(o.directories)}
	if _, err := dashboardSaveOriginal("../../.tools", "dashboard-original-"+record.Parent+".json", dashboardOriginalParent{1, record, cleanup, o.artifacts}); err != nil {
		t.Fatal("complete dashboard parent observation persistence failed")
	}
}
