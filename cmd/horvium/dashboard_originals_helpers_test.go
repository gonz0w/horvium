// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type dashboardOriginalFixture struct {
	Schema  int                     `json:"schema"`
	Mode    string                  `json:"mode"`
	Fixture dashboardFixtureReceipt `json:"fixture"`
}
type dashboardGuestCleanup struct {
	DescendantsRemoved          bool `json:"descendants_removed"`
	SocketsRemoved              bool `json:"sockets_removed"`
	TemporaryDirectoriesRemoved bool `json:"temporary_directories_removed"`
	StorageCleanups             int  `json:"storage_cleanups"`
	Pairs                       int  `json:"pairs"`
}
type dashboardOriginalParent struct {
	Schema    int                            `json:"schema"`
	Controls  *dashboardParentControlReceipt `json:"controls"`
	Cleanup   dashboardGuestCleanup          `json:"cleanup"`
	Artifacts dashboardArtifactObservation   `json:"artifacts"`
}
type dashboardOriginalFile struct {
	directory, name string
	sum             [32]byte
	info            os.FileInfo
}

func dashboardOriginalName(name string) bool {
	for _, suffix := range []string{"synthetic-100", "synthetic-10000", "native-small", "projection-10000", "TestNativePerformanceBudgets", "TestNativeReaderResourceBudgets"} {
		if name == "dashboard-original-"+suffix+".json" {
			return true
		}
	}
	return false
}
func dashboardSaveOriginal(directory, name string, value any) (*dashboardOriginalFile, error) {
	fail := errors.New("original dashboard observation unavailable")
	b, err := json.Marshal(value)
	if err != nil || len(b) == 0 || len(b) > 65536 || !dashboardOriginalName(name) {
		return nil, fail
	}
	info, err := os.Lstat(directory)
	abs, re := filepath.EvalSymlinks(directory)
	if err != nil || re != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fail
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, fail
	}
	defer root.Close()
	f, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fail
	}
	n, we := f.Write(b)
	se := f.Sync()
	info, ie := f.Stat()
	ce := f.Close()
	if we != nil || n != len(b) || se != nil || ie != nil || ce != nil {
		return nil, fail
	}
	r := &dashboardOriginalFile{abs, name, sha256.Sum256(b), info}
	if r.check() != nil {
		return nil, fail
	}
	return r, nil
}
func (r *dashboardOriginalFile) check() error {
	fail := errors.New("original dashboard observation changed")
	if r == nil || !dashboardOriginalName(r.name) || r.info == nil {
		return fail
	}
	root, err := os.OpenRoot(r.directory)
	if err != nil {
		return fail
	}
	defer root.Close()
	path, err := root.Lstat(r.name)
	if err != nil || !path.Mode().IsRegular() || path.Mode().Perm() != 0600 || !os.SameFile(path, r.info) {
		return fail
	}
	f, err := root.Open(r.name)
	if err != nil {
		return fail
	}
	opened, oe := f.Stat()
	b, be := io.ReadAll(io.LimitReader(f, 65537))
	after, ae := f.Stat()
	ce := f.Close()
	if oe != nil || be != nil || ae != nil || ce != nil || len(b) > 65536 || !os.SameFile(opened, path) || !os.SameFile(after, path) || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) || sha256.Sum256(b) != r.sum {
		return fail
	}
	return nil
}
func dashboardEmptyReaper(flags, children, descendants uint32, owner, actual int) bool {
	return flags == 1 && children == 0 && descendants == 0 && owner > 1 && owner == actual
}

func dashboardCopyOriginalArtifact(from, to, expected string) error {
	fail := errors.New("original launch artifact changed")
	info, err := os.Lstat(from)
	if err != nil || !dashboardArtifactStat(info) {
		return fail
	}
	input, err := os.Open(from)
	if err != nil {
		return fail
	}
	defer input.Close()
	opened, err := input.Stat()
	before, de := dashboardArtifactDigest(input, info.Size())
	if err != nil || de != nil || !dashboardArtifactUnchanged(info, opened) || before != expected {
		return fail
	}
	if _, err := input.Seek(0, io.SeekStart); err != nil {
		return fail
	}
	output, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		return fail
	}
	h := sha256.New()
	n, we := io.Copy(io.MultiWriter(output, h), io.LimitReader(input, info.Size()+1))
	se := output.Sync()
	outInfo, ie := output.Stat()
	ce := output.Close()
	after, ae := input.Stat()
	repeated, re := dashboardArtifactDigest(input, info.Size())
	if we != nil || se != nil || ie != nil || ce != nil || ae != nil || re != nil || n != info.Size() || fmt.Sprintf("%x", h.Sum(nil)) != expected || repeated != expected || !dashboardArtifactUnchanged(info, after) {
		return fail
	}
	copy, err := os.Open(to)
	if err != nil {
		return fail
	}
	actual, de := dashboardArtifactDigest(copy, info.Size())
	ci, ie := copy.Stat()
	ce = copy.Close()
	if de != nil || ie != nil || ce != nil || !dashboardArtifactUnchanged(outInfo, ci) || actual != expected {
		return fail
	}
	return nil
}
