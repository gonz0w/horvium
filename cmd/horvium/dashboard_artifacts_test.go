// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"crypto/sha256"
	"debug/buildinfo"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
)

func dashboardBuildInfoFixture() *debug.BuildInfo {
	return &debug.BuildInfo{GoVersion: "go1.27.1", Settings: []debug.BuildSetting{{Key: "GOOS", Value: "freebsd"}, {Key: "GOARCH", Value: "amd64"}, {Key: "GOAMD64", Value: "v1"}, {Key: "CGO_ENABLED", Value: "0"}}}
}
func dashboardArtifactFiles(t *testing.T) (*os.Root, string) {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"horvium", "reader-fixture.test", "dashboard-native.test"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("synthetic artifact "+name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	return root, filepath.Join(dir, "dashboard-native.test")
}
func TestDashboardArtifactOriginalBytesAndController(t *testing.T) {
	root, exe := dashboardArtifactFiles(t)
	calls := 0
	decode := func(r io.ReaderAt) (*debug.BuildInfo, error) {
		calls++
		var b [9]byte
		if _, e := r.ReadAt(b[:], 0); e != nil || string(b[:]) != "synthetic" {
			t.Fatal("not original opened bytes")
		}
		return dashboardBuildInfoFixture(), nil
	}
	got, err := dashboardObserveArtifacts(root, exe, decode)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 || got.GoVersion != "go1.27.1" || len(got.Artifacts) != 3 {
		t.Fatalf("incomplete observation: %+v", got)
	}
	for role, name := range dashboardArtifactNames {
		if got.Artifacts[role] != fmt.Sprintf("%x", sha256.Sum256([]byte("synthetic artifact "+name))) {
			t.Fatal("hash not original bytes")
		}
	}
	if _, err = dashboardObserveArtifacts(root, filepath.Join(root.Name(), "horvium"), decode); err == nil {
		t.Fatal("different controller accepted")
	}
}
func TestDashboardArtifactFileRefusals(t *testing.T) {
	for _, kind := range []string{"missing", "symlink", "directory", "empty", "oversized", "nonexecutable", "writable"} {
		t.Run(kind, func(t *testing.T) {
			root, exe := dashboardArtifactFiles(t)
			path := filepath.Join(root.Name(), "horvium")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "symlink":
				if err := os.Symlink("reader-fixture.test", path); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "empty", "oversized":
				f, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0700)
				if e != nil {
					t.Fatal(e)
				}
				if kind == "oversized" {
					if e = f.Truncate((128 << 20) + 1); e != nil {
						t.Fatal(e)
					}
				}
				if e = f.Close(); e != nil {
					t.Fatal(e)
				}
			case "nonexecutable", "writable":
				mode := os.FileMode(0600)
				if kind == "writable" {
					mode = 0777
				}
				if e := os.WriteFile(path, []byte("artifact"), mode); e != nil {
					t.Fatal(e)
				}
				if e := os.Chmod(path, mode); e != nil {
					t.Fatal(e)
				}
			}
			if got, e := dashboardObserveArtifacts(root, exe, func(io.ReaderAt) (*debug.BuildInfo, error) { return dashboardBuildInfoFixture(), nil }); e == nil || len(got.Artifacts) != 0 {
				t.Fatal("unsafe artifact returned partial/success observation")
			}
		})
	}
}
func TestDashboardArtifactMutationAndDecodeFailure(t *testing.T) {
	for _, kind := range []string{"rewrite", "replace", "later-artifact", "metadata", "decode"} {
		t.Run(kind, func(t *testing.T) {
			root, exe := dashboardArtifactFiles(t)
			calls := 0
			decode := func(r io.ReaderAt) (*debug.BuildInfo, error) {
				calls++
				name := "horvium"
				if kind == "later-artifact" {
					name = "reader-fixture.test"
				}
				if calls == 1 {
					path := filepath.Join(root.Name(), name)
					switch kind {
					case "rewrite", "later-artifact":
						if e := os.WriteFile(path, bytes.Repeat([]byte("x"), 100), 0700); e != nil {
							t.Fatal(e)
						}
					case "replace":
						if e := os.Remove(path); e != nil {
							t.Fatal(e)
						}
						if e := os.WriteFile(path, []byte("replacement"), 0700); e != nil {
							t.Fatal(e)
						}
					case "metadata":
						if e := os.Chmod(path, 0600); e != nil {
							t.Fatal(e)
						}
					case "decode":
						return nil, errors.New("decode failed")
					}
				}
				return dashboardBuildInfoFixture(), nil
			}
			if got, e := dashboardObserveArtifacts(root, exe, decode); e == nil || len(got.Artifacts) != 0 {
				t.Fatal("mutated/incomplete candidate observed")
			}
		})
	}
}
func TestDashboardArtifactTargetAndCompilerRefusals(t *testing.T) {
	for _, kind := range []string{"GOOS", "GOARCH", "GOAMD64", "CGO_ENABLED", "duplicate", "missing", "invalid-version", "version-disagreement", "nil"} {
		t.Run(kind, func(t *testing.T) {
			root, exe := dashboardArtifactFiles(t)
			calls := 0
			decode := func(io.ReaderAt) (*debug.BuildInfo, error) {
				calls++
				b := dashboardBuildInfoFixture()
				switch kind {
				case "nil":
					return nil, nil
				case "duplicate":
					b.Settings = append(b.Settings, b.Settings[0])
				case "missing":
					b.Settings = b.Settings[1:]
				case "invalid-version":
					b.GoVersion = "devel unpinned"
				case "version-disagreement":
					if calls == 2 {
						b.GoVersion = "go1.27.2"
					}
				default:
					for i := range b.Settings {
						if b.Settings[i].Key == kind {
							b.Settings[i].Value = "wrong"
						}
					}
				}
				return b, nil
			}
			if got, e := dashboardObserveArtifacts(root, exe, decode); e == nil || len(got.Artifacts) != 0 {
				t.Fatal("wrong target/compiler returned")
			}
		})
	}
}
func TestDashboardArtifactActualHostExecutable(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range dashboardArtifactNames {
		if e := os.WriteFile(filepath.Join(dir, name), data, 0700); e != nil {
			t.Fatal(e)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	// Inspect a real compiler-produced object. Hosts outside the target must be
	// denied; a native target must agree with the original metadata and bytes.
	metadata, err := buildinfo.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	version, targetErr := dashboardArtifactCompiler(metadata)
	got, observeErr := dashboardObserveArtifacts(root, filepath.Join(dir, "dashboard-native.test"), buildinfo.Read)
	if targetErr != nil {
		if observeErr == nil {
			t.Fatal("host binary claimed native target")
		}
		return
	}
	if observeErr != nil || got.GoVersion != version || got.Artifacts["fixture"] != fmt.Sprintf("%x", sha256.Sum256(data)) {
		t.Fatal("actual native object observation disagrees")
	}
}
func TestDashboardGuestFacts(t *testing.T) {
	valid := func() dashboardGuestFacts {
		return dashboardGuestFacts{Kernel: "15.1-RELEASE-p1\n", Userland: "15.1-RELEASE\n", ZFS: "zfs-2.4.0\nzfs-kmod-2.4.0\n", PhysicalBytes: "6442450944\n"}
	}
	got, e := dashboardNormalizeGuestFacts(valid())
	if e != nil || got.PhysicalBytes != 6442450944 || got.Kernel != "15.1-RELEASE-p1" {
		t.Fatalf("actual facts: %+v %v", got, e)
	}
	for _, kind := range []string{"wrong-release", "development", "mismatch", "kernel-lines", "zfs-empty", "zfs-control", "zfs-long", "memory-zero", "memory-sign", "memory-overflow", "memory-junk"} {
		t.Run(kind, func(t *testing.T) {
			f := valid()
			switch kind {
			case "wrong-release":
				f.Kernel = "13.5-RELEASE"
			case "development":
				f.Kernel = "15.1-STABLE"
			case "mismatch":
				f.Userland = "14.5-RELEASE"
			case "kernel-lines":
				f.Kernel += "extra\n"
			case "zfs-empty":
				f.ZFS = "\n"
			case "zfs-control":
				f.ZFS = "zfs\x00"
			case "zfs-long":
				f.ZFS = string(bytes.Repeat([]byte("a"), 513))
			case "memory-zero":
				f.PhysicalBytes = "0"
			case "memory-sign":
				f.PhysicalBytes = "+10"
			case "memory-overflow":
				f.PhysicalBytes = "18446744073709551616"
			case "memory-junk":
				f.PhysicalBytes = "10 bytes"
			}
			if _, e := dashboardNormalizeGuestFacts(f); e == nil {
				t.Fatal("malformed fact accepted")
			}
		})
	}
}

func TestDashboardFactOutputBoundAndMissingObserver(t *testing.T) {
	var buffer dashboardFactBuffer
	if n, e := buffer.Write(bytes.Repeat([]byte("a"), 512)); e != nil || n != 512 {
		t.Fatal("bounded original output lost")
	}
	if n, e := buffer.Write([]byte("b")); e == nil || n != 0 || len(buffer.data) != 512 {
		t.Fatal("overflow accepted/partially appended")
	}
	root, exe := dashboardArtifactFiles(t)
	if _, e := dashboardObserveArtifacts(nil, exe, buildinfo.Read); e == nil {
		t.Fatal("missing root accepted")
	}
	if _, e := dashboardObserveArtifacts(root, exe, nil); e == nil {
		t.Fatal("missing decoder accepted")
	}
	for _, release := range []string{"15.1", "14.5"} {
		if _, e := dashboardNormalizeGuestFacts(dashboardGuestFacts{release + "-RELEASE-p2", release + "-RELEASE-p3", "zfs-actual zfs-kmod-actual", "6442450944"}); e != nil {
			t.Fatal(e)
		}
	}
}

func TestDashboardArtifactDecoderCannotReadGrowth(t *testing.T) {
	root, exe := dashboardArtifactFiles(t)
	calls := 0
	decode := func(r io.ReaderAt) (*debug.BuildInfo, error) {
		calls++
		if calls == 1 {
			path := filepath.Join(root.Name(), "horvium")
			original, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(path, append(original, []byte("growth")...), 0700); e != nil {
				t.Fatal(e)
			}
			var b [1]byte
			n, e := r.ReadAt(b[:], int64(len(original)))
			if n != 0 || e != io.EOF {
				t.Error("decoder read beyond the originally bounded file")
			}
		}
		return dashboardBuildInfoFixture(), nil
	}
	if _, e := dashboardObserveArtifacts(root, exe, decode); e == nil {
		t.Fatal("grown original file accepted")
	}
}
