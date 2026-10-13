// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
)

// Fixed roles already produced by the reviewed host builder. No operator paths.
var dashboardArtifactNames = map[string]string{"management": "horvium", "helper": "reader-fixture.test", "fixture": "dashboard-native.test"}
var dashboardArtifactRoles = []string{"management", "helper", "fixture"}

type dashboardArtifactObservation struct {
	Artifacts map[string]string `json:"artifacts"`
	GoVersion string            `json:"go_version"`
}
type dashboardOpenedArtifact struct {
	file *os.File
	info os.FileInfo
	hash string
}

func dashboardArtifactStat(info os.FileInfo) bool {
	return info != nil && info.Mode().IsRegular() && info.Size() > 0 && info.Size() <= 128<<20 && info.Mode().Perm()&0111 != 0 && info.Mode().Perm()&0022 == 0
}
func dashboardArtifactUnchanged(before, after os.FileInfo) bool {
	return dashboardArtifactStat(after) && os.SameFile(before, after) && before.Size() == after.Size() && before.Mode() == after.Mode() && before.ModTime().Equal(after.ModTime())
}
func dashboardArtifactDigest(file *os.File, size int64) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(file, size+1))
	if err != nil || n != size {
		return "", errors.New("incomplete artifact bytes")
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
func dashboardArtifactCompiler(b *debug.BuildInfo) (string, error) {
	if b == nil || !regexp.MustCompile(`^go[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(b.GoVersion) {
		return "", errors.New("unidentified artifact compiler")
	}
	expected := map[string]string{"GOOS": "freebsd", "GOARCH": "amd64", "GOAMD64": "v1", "CGO_ENABLED": "0"}
	seen := map[string]bool{}
	for _, s := range b.Settings {
		if value, ok := expected[s.Key]; ok {
			if seen[s.Key] || s.Value != value {
				return "", errors.New("incorrect artifact target")
			}
			seen[s.Key] = true
		}
	}
	if len(seen) != len(expected) {
		return "", errors.New("incomplete artifact target")
	}
	return b.GoVersion, nil
}

// Hash and decode the same opened objects, then recheck the whole original set.
// decode is injected only for portable malformed/mutation tests; native uses
// debug/buildinfo.Read. Observations confer no execution or acceptance authority.
func dashboardObserveArtifacts(root *os.Root, executable string, decode func(io.ReaderAt) (*debug.BuildInfo, error)) (result dashboardArtifactObservation, err error) {
	if root == nil || decode == nil {
		return result, errors.New("missing artifact observer")
	}
	opened := map[string]dashboardOpenedArtifact{}
	defer func() {
		for _, a := range opened {
			err = errors.Join(err, a.file.Close())
		}
		if err != nil {
			result = dashboardArtifactObservation{}
		}
	}()
	for _, role := range dashboardArtifactRoles {
		name := dashboardArtifactNames[role]
		before, e := root.Lstat(name)
		if e != nil || !dashboardArtifactStat(before) {
			return result, errors.New("unsafe original artifact")
		}
		f, e := root.Open(name)
		if e != nil {
			return result, errors.New("cannot open original artifact")
		}
		opened[role] = dashboardOpenedArtifact{file: f, info: before}
		actual, e := f.Stat()
		if e != nil || !dashboardArtifactUnchanged(before, actual) {
			return result, errors.New("original artifact replaced")
		}
		hash, e := dashboardArtifactDigest(f, before.Size())
		if e != nil {
			return result, e
		}
		opened[role] = dashboardOpenedArtifact{f, before, hash}
	}
	self, e := os.Lstat(executable)
	if e != nil || !dashboardArtifactUnchanged(opened["fixture"].info, self) {
		return result, errors.New("controller is not observed fixture executable")
	}
	result.Artifacts = map[string]string{}
	for _, role := range dashboardArtifactRoles {
		a := opened[role]
		b, e := decode(io.NewSectionReader(a.file, 0, a.info.Size()))
		if e != nil {
			return result, errors.New("unreadable artifact build metadata")
		}
		version, e := dashboardArtifactCompiler(b)
		if e != nil {
			return result, e
		}
		if result.GoVersion != "" && result.GoVersion != version {
			return result, errors.New("artifact compiler disagreement")
		}
		result.GoVersion = version
		result.Artifacts[role] = a.hash
	}
	for _, role := range dashboardArtifactRoles {
		a := opened[role]
		hash, he := dashboardArtifactDigest(a.file, a.info.Size())
		after, e := a.file.Stat()
		leaf, le := root.Lstat(dashboardArtifactNames[role])
		if e != nil || le != nil || he != nil || !dashboardArtifactUnchanged(a.info, after) || !dashboardArtifactUnchanged(a.info, leaf) || hash != a.hash {
			return result, errors.New("original artifact changed during observation")
		}
	}
	self, e = os.Lstat(executable)
	if e != nil || !dashboardArtifactUnchanged(opened["fixture"].info, self) {
		return result, errors.New("controller identity changed")
	}
	return result, nil
}

type dashboardGuestFacts struct{ Kernel, Userland, ZFS, PhysicalBytes string }
type dashboardGuestFactObservation struct {
	Kernel        string `json:"kernel"`
	Userland      string `json:"userland"`
	ZFS           string `json:"zfs"`
	PhysicalBytes uint64 `json:"physical_bytes"`
}

func dashboardNormalizeGuestFacts(f dashboardGuestFacts) (dashboardGuestFactObservation, error) {
	var result dashboardGuestFactObservation
	for _, s := range []string{f.Kernel, f.Userland, f.ZFS, f.PhysicalBytes} {
		if len(s) == 0 || len(s) > 512 {
			return result, errors.New("incomplete/oversized native fact")
		}
		for _, c := range s {
			if c < 32 && c != '\n' || c == 127 {
				return result, errors.New("invalid native fact")
			}
		}
	}
	kernel, userland := strings.TrimSpace(f.Kernel), strings.TrimSpace(f.Userland)
	version := regexp.MustCompile(`^(15\.1|14\.5)-RELEASE(?:-p[0-9]+)?$`)
	k, u := version.FindStringSubmatch(kernel), version.FindStringSubmatch(userland)
	if k == nil || u == nil || k[1] != u[1] {
		return result, errors.New("unexpected actual guest release")
	}
	zfs := strings.Join(strings.Fields(f.ZFS), " ")
	if len(zfs) == 0 {
		return result, errors.New("missing ZFS version")
	}
	physical := strings.TrimSpace(f.PhysicalBytes)
	n, e := strconv.ParseUint(physical, 10, 64)
	if e != nil || n == 0 || strconv.FormatUint(n, 10) != physical {
		return result, errors.New("invalid actual physical memory")
	}
	return dashboardGuestFactObservation{kernel, userland, zfs, n}, nil
}

type dashboardFactBuffer struct{ data []byte }

func (b *dashboardFactBuffer) Write(p []byte) (int, error) {
	if len(p) > 512-len(b.data) {
		return 0, errors.New("oversized native fact output")
	}
	b.data = append(b.data, p...)
	return len(p), nil
}
