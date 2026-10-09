// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

const supervisorFixtureArgument = "--horvium-test-property-supervisor"

func nativeSupervisorFixtureProcess(args []string) (bool, int) {
	if len(args) == 0 || args[0] != supervisorFixtureArgument {
		return false, 0
	}
	if len(args) != 3 || os.Getuid() != 0 || os.Geteuid() != 0 {
		return true, 1
	}
	mode, dir := args[1], args[2]
	if mode != "normal" && mode != "escaped" && mode != "normal-escaped" && mode != "residual" {
		return true, 1
	}
	request, response := supervisorPipe(3), supervisorPipe(4)
	if request == nil || response == nil {
		return true, 1
	}
	defer request.Close()
	defer response.Close()
	code := runPropertySupervisor(request, response, func(ctx context.Context, _ storage.PropertyIntent) error {
		if os.WriteFile(filepath.Join(dir, "launched"), []byte("owned fixture"), 0600) != nil {
			return ErrUnavailable
		}
		if mode == "normal" {
			return nil
		}
		if mode == "residual" {
			c := exec.Command(os.Args[0], "-test.run=^TestNativePropertyControlGuardChild$", "--", dir, "child")
			if c.Start() != nil {
				return ErrUnavailable
			}
			until := time.Now().Add(time.Second)
			for time.Now().Before(until) {
				if _, e := os.Stat(filepath.Join(dir, "child")); e == nil {
					return nil
				}
				time.Sleep(5 * time.Millisecond)
			}
			return ErrUnavailable
		}
		ctx, stop := context.WithTimeout(ctx, 4*time.Second)
		defer stop()
		if mode == "normal-escaped" {
			_, err := executeNative(ctx, os.Args[0], []string{"-test.run=^TestNativePropertyControlGuardChild$", "--", dir, "supervised-escaped"}, 128)
			return err
		}
		_, err := executeNative(ctx, os.Args[0], []string{"-test.run=^TestProbeTreeProcess$", "--", "escaped-child", filepath.Join(dir, "child"), filepath.Join(dir, "probe")}, 128)
		return err
	})
	return true, code
}
func supervisorLiveIntent(t testing.TB) storage.PropertyIntent {
	t.Helper()
	p, o, err := policySource(func(string) (storage.Policy, error) { return collectionPolicy(), nil }).Collect(context.Background(), "100")
	if err != nil {
		t.Fatal(err)
	}
	o.Eligible = true
	at := time.Now().UTC()
	plan, err := storage.Build(at, "owned-supervisor-fixture", p, o, storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "2097152"})
	if err != nil {
		t.Fatal(err)
	}
	i, err := plan.CapturePropertyIntent(at, "owned-supervisor-fixture", p, o)
	if err != nil {
		t.Fatal(err)
	}
	return i
}
func TestNativeSupervisedOwner(t *testing.T) {
	if len(flag.Args()) != 1 {
		return
	}
	recoveryOptIn(t)
	dir := flag.Args()[0]
	i := supervisorLiveIntent(t)
	data, _ := storage.EncodePropertyIntent(i)
	owner, err := bindBootOwner(readNativeBoot)
	if err != nil {
		t.Fatal(err)
	}
	_ = exchangePropertySupervisor(context.Background(), i, data, owner, func(c *exec.Cmd) {
		c.Args = []string{os.Args[0], supervisorFixtureArgument, "escaped", dir}
	})
}
func TestNativeExecutorSupervision(t *testing.T) {
	recoveryOptIn(t)
	t.Run("boot-identity", func(t *testing.T) {
		a, err := readNativeBoot()
		if err != nil {
			t.Fatal("actual raw kernel boot identity", err)
		}
		b, err := readNativeBoot()
		if err != nil || a != b {
			t.Fatal("kernel boot identity changed in one boot")
		}
		if os.WriteFile(".tools/supervisor-boot.bin", a[:], 0600) != nil {
			t.Fatal("owned boot witness")
		}
		output := recoveryCommand(t, "/root/work/.tools/executor-observer", "abi")
		if strings.TrimSpace(output) != "SUPERVISOR_NATIVE_ABI status=80 kill=80 pid=0 acquire=2 statuscmd=4 killcmd=6" || unsafe.Sizeof(nativeReaperStatus{}) != 80 || unsafe.Sizeof(nativeReaperKill{}) != 80 {
			t.Fatal("unsupported native reaper layout", output)
		}
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		defer w.Close()
		for _, f := range []*os.File{r, w} {
			var st unix.Stat_t
			fd := f.Fd()
			flags, e := unix.FcntlInt(fd, unix.F_GETFL, 0)
			if unix.Fstat(int(fd), &st) != nil || e != nil || !validSupervisorPipe(uint32(st.Mode), st.Uid, uint64(st.Nlink), flags) {
				t.Fatal("actual root anonymous pipe provenance", st.Mode, st.Uid, st.Nlink, flags, e)
			}
		}
		fmt.Print(output)
	})
	t.Run("clock-independent", func(t *testing.T) {
		original, err := readNativeBoot()
		if err != nil {
			t.Fatal(err)
		}
		before := time.Now()
		shifted := unix.NsecToTimeval(before.Add(time.Minute).UnixNano())
		if unix.Settimeofday(&shifted) != nil {
			t.Fatal("disposable guest clock shift")
		}
		defer func() {
			restored := unix.NsecToTimeval(time.Now().Add(-time.Minute).UnixNano())
			if unix.Settimeofday(&restored) != nil {
				t.Error("restore owned guest clock")
			}
		}()
		current, err := readNativeBoot()
		if err != nil || current != original {
			t.Fatal("boot identity depends on civil clock")
		}
	})
	for _, mode := range []string{"ready-success", "residual-descendant-denied", "peer-before-launch", "foreign-handshake", "normal-escaped", "guard-first", "supervisor-first", "owner-death"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			i := supervisorLiveIntent(t)
			data, _ := storage.EncodePropertyIntent(i)
			owner, err := bindBootOwner(readNativeBoot)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "peer-before-launch" || mode == "foreign-handshake" {
				r, w, _ := os.Pipe()
				sr, sw, _ := os.Pipe()
				defer r.Close()
				defer w.Close()
				defer sr.Close()
				defer sw.Close()
				c := exec.Command(os.Args[0], supervisorFixtureArgument, "normal", dir)
				c.ExtraFiles = []*os.File{r, sw}
				if c.Start() != nil {
					t.Fatal("owned child start")
				}
				r.Close()
				sw.Close()
				token := [16]byte{7}
				if writeSupervisorRequest(w, token, owner.identity, data) != nil || readSupervisorFrame(sr, 1, token, owner.identity) != nil {
					t.Fatal("actual ready")
				}
				if mode == "foreign-handshake" {
					f := supervisorFrame(2, [16]byte{8}, owner.identity)
					w.Write(f[:])
				}
				w.Close()
				if c.Wait() == nil {
					t.Fatal("peer loss/foreign handshake accepted")
				}
				if _, e := os.Stat(filepath.Join(dir, "launched")); !errors.Is(e, os.ErrNotExist) {
					t.Fatal("denied handshake launched fixture")
				}
				return
			}
			if mode == "ready-success" || mode == "residual-descendant-denied" {
				if e := exchangePropertySupervisor(context.Background(), i, data, owner, func(c *exec.Cmd) {
					m := "normal"
					if mode == "residual-descendant-denied" {
						m = "residual"
					}
					c.Args = []string{os.Args[0], supervisorFixtureArgument, m, dir}
				}); (e == nil) != (mode == "ready-success") {
					t.Fatal("actual reaper completion classification", e)
				}
				if _, e := os.Stat(filepath.Join(dir, "launched")); e != nil {
					t.Fatal("fixture consumer was not executed")
				}
				if mode == "residual-descendant-denied" {
					assertProbeStopped(t, fixturePID(t, filepath.Join(dir, "child")))
				}
				return
			}
			done := make(chan error, 1)
			var parent *exec.Cmd
			if mode == "owner-death" {
				parent = exec.Command(os.Args[0], "-test.run=^TestNativeSupervisedOwner$", "--", dir)
				parent.Env = os.Environ()
				if parent.Start() != nil {
					t.Fatal("original helper fixture")
				}
				go func() { done <- parent.Wait() }()
			} else {
				go func() {
					done <- exchangePropertySupervisor(context.Background(), i, data, owner, func(c *exec.Cmd) {
						m := "escaped"
						if mode == "normal-escaped" {
							m = mode
						}
						c.Args = []string{os.Args[0], supervisorFixtureArgument, m, dir}
					})
				}()
			}
			probeFile := "probe"
			if mode == "normal-escaped" {
				probeFile = "parent"
			}
			probe := recoveryInspect(t, fixturePID(t, filepath.Join(dir, probeFile)))
			child := recoveryInspect(t, fixturePID(t, filepath.Join(dir, "child")))
			guard := recoveryInspect(t, probe.PPID)
			supervisor := recoveryInspect(t, guard.PPID)
			if !probe.Found || !child.Found || !guard.Found || !supervisor.Found || supervisor.Reaper != supervisor.PID || supervisor.ReaperFlags != 1 || guard.Reaper != guard.PID || guard.Descendants < 2 || child.PGID == probe.PGID {
				t.Fatal("actual isolated supervisor/reaper/escaped lineage")
			}
			t.Cleanup(func() {
				for _, p := range []recoveryProcess{supervisor, guard, probe, child} {
					recoverySignal(t, p, syscall.SIGKILL)
				}
			})
			if mode == "normal-escaped" {
				if os.WriteFile(filepath.Join(dir, "release"), nil, 0600) != nil || os.WriteFile(filepath.Join(dir, "parent-release"), nil, 0600) != nil {
					t.Fatal("owned child release")
				}
			}
			if mode == "guard-first" {
				recoverySignal(t, guard, syscall.SIGKILL)
			}
			if mode == "supervisor-first" {
				recoveryStop(t, supervisor)
				recoverySignal(t, guard, syscall.SIGKILL)
				recoverySignal(t, supervisor, syscall.SIGKILL)
			}
			if mode == "owner-death" {
				original := recoveryInspect(t, parent.Process.Pid)
				recoverySignal(t, original, syscall.SIGKILL)
			}
			select {
			case e := <-done:
				if (e == nil) != (mode == "normal-escaped") {
					t.Fatal("actual supervisor completion differs", e)
				}
			case <-time.After(12 * time.Second):
				t.Fatal("owned original consumer did not join")
			}
			if mode == "supervisor-first" {
				recoverySignal(t, probe, syscall.SIGKILL)
				recoverySignal(t, child, syscall.SIGKILL)
			}
			// When the owner dies the independent supervisor keeps its bounded duties.
			until := time.Now().Add(7 * time.Second)
			for time.Now().Before(until) {
				p := recoveryInspect(t, supervisor.PID)
				if !recoverySame(supervisor, p) || p.Zombie {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			for _, p := range []recoveryProcess{supervisor, guard, probe, child} {
				assertProbeStopped(t, p.PID)
			}
		})
	}
	if !t.Failed() {
		fmt.Println("SUPERVISOR_RESULT before=pass")
	}
}
func TestNativeSupervisorBootAfter(t *testing.T) {
	recoveryOptIn(t)
	before, err := os.ReadFile(".tools/supervisor-boot.bin")
	if err != nil || len(before) != 16 {
		t.Fatal("owned pre-reboot native identity")
	}
	after, err := readNativeBoot()
	if err != nil || string(before) == string(after[:]) {
		t.Fatal("real reboot did not change kernel identity")
	}
	if os.Remove(".tools/supervisor-boot.bin") != nil {
		t.Fatal("owned boot witness teardown")
	}
	fmt.Println("SUPERVISOR_RESULT after=pass")
}

func TestNativeSupervisorPropertyControls(t *testing.T) {
	recoveryOptIn(t)
	f := recoverySetup(t)
	source, err := NewPoolPropertySource()
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithTimeout(context.Background(), 120*time.Second)
	defer stop()
	nativePropertyControl(t, ctx, source, f.Policy.PoolGUID, f.Policy.PoolName)
	t.Run("fixture-teardown", func(t *testing.T) {
		for _, pool := range []struct{ name, guid string }{{f.Policy.PoolName, f.Policy.PoolGUID}, {f.MetadataName, f.MetadataGUID}} {
			if strings.TrimSpace(recoveryCommand(t, "/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", pool.name)) != pool.guid {
				t.Fatal("owned retained-control pool identity")
			}
			recoveryCommand(t, "/sbin/zpool", "destroy", pool.name)
		}
		if os.RemoveAll(recoveryRoot) != nil {
			t.Fatal("owned retained-control fixture removal")
		}
	})
	if !t.Failed() {
		fmt.Println("SUPERVISOR_CONTROLS_RESULT pass")
	}
}
