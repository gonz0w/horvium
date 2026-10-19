// SPDX-License-Identifier: BSD-3-Clause
package freebsd

// Disposable research fixtures only. No metadata-derived signal authority and
// no product admission. Every case has its own self-reaper process, acquired
// before fixture birth. Unreaped original ancestry reserves the held guard PID
// until a serialized, irreversible signal-phase seal precedes every Wait gate.
import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
)

func runStableCutController(t *testing.T, mode, anchor string) {
	t.Helper()
	dir := t.TempDir()
	lease := "-"
	if anchor != "" {
		lease = filepath.Join(anchor, mode)
		if os.Mkdir(lease, 0700) != nil {
			t.Fatal("exclusive stable fixture lease")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNativeStableCutController$", "--", mode, dir, lease)
	c.Env = os.Environ()
	output := &capture{limit: 16384, cancel: cancel}
	c.Stdout, c.Stderr = sink{capture: output, keep: true}, sink{capture: output, keep: true}
	err := c.Run()
	output.Lock()
	defer output.Unlock()
	if err != nil || output.exceeded || ctx.Err() != nil {
		t.Fatal("stable fixture controller failed; no later native work", err, output.out.String())
	}
	if strings.Count(output.out.String(), "STABLE_FIXTURE_RESULT mode="+mode+" cleanup=verified\n") != 1 {
		t.Fatal("missing once-only actual stable fixture controller result")
	}
	fmt.Print(output.out.String())
}

// The original native prepare/ownedCommand remain in use. Only its final stop
// is held, before the sole Cmd.Wait. HOLD acknowledges live fixture custody; the born callback reports
// the born original Cmd's PID and blocks. On EOF/deadline it exits this creator,
// never returning into a consuming Wait. Its unreaped ancestor remains held by
// the isolated controller. SEAL is sent only after direct signal authority dies.
func stableHeldExecutor(parent context.Context, dir string, controlFD, statusFD uintptr) error {
	request, response := supervisorPipe(controlFD), supervisorPipe(statusFD)
	if request == nil || response == nil {
		os.Exit(2)
	}
	defer request.Close()
	defer response.Close()
	end := time.Now().Add(12 * time.Second)
	if d, ok := parent.Deadline(); ok && d.Before(end) {
		end = d
	}
	if request.SetReadDeadline(end) != nil || response.SetWriteDeadline(end) != nil {
		os.Exit(2)
	}
	ctx, cancel := context.WithDeadline(parent, end)
	defer cancel()
	path, args := os.Args[0], []string{"-test.run=^TestNativeStableProbe$", "--", dir, "probe"}
	prepare := prepareNativeCommand(ctx, path, args)
	waited := make(chan struct{})
	_, err := executePrepared(ctx, path, args, 128, func(c *exec.Cmd, p *processControl) func(error) error {
		classify := prepare(c, p)
		c.Env = append(c.Env, "HORVIUM_NATIVE_VM=1", "HORVIUM_PROPERTY_VM=1", "HORVIUM_RESEARCH_VM=1")
		stop, proof := p.stop, p.proof
		p.started = func() {
			var h [1]byte
			if _, e := io.ReadFull(request, h[:]); e != nil || h[0] != 'H' {
				os.Exit(2)
			}
			_ = fixturePublishBirth(response, c.Process.Pid, response.Close, func() { os.Exit(2) })
		}
		p.stop = func() error { return fixtureStopAfterSeal(request, stop, func() { os.Exit(2) }) }
		p.proof = func(e error, d time.Time) error { defer close(waited); return proof(e, d) }
		return classify
	})
	timer := time.NewTimer(max(0, time.Until(end)))
	defer timer.Stop()
	select {
	case <-waited:
		return err
	case <-timer.C:
		os.Exit(2)
	}
	return ErrCleanupDebt
}

func TestNativeStableLeaseOwner(t *testing.T) {
	a := flag.Args()
	if len(a) != 2 {
		return
	}
	recoveryOptIn(t)
	s, err := storage.OpenStateDirectory(a[0])
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if stableHeldExecutor(context.Background(), a[1], 3, 4) != nil {
		os.Exit(1)
	}
}

// Escaped fixture descendants have an independent, cooperative cleanup release.
// No saved child PID is ever signalled. They stay live through the actual cut;
// cleanup releases them only after the signal phase. The probe owns its one Wait.
func TestNativeStableProbe(t *testing.T) {
	a := flag.Args()
	if len(a) != 2 {
		return
	}
	recoveryOptIn(t)
	dir, mode := a[0], a[1]
	if mode != "probe" && mode != "child" {
		t.Fatal("fixed stable probe mode")
	}
	if os.WriteFile(filepath.Join(dir, mode), []byte(strconv.Itoa(os.Getpid())), 0600) != nil {
		os.Exit(2)
	}
	var child *exec.Cmd
	if mode == "probe" {
		child = exec.Command(os.Args[0], "-test.run=^TestNativeStableProbe$", "--", dir, "child")
		child.Env = os.Environ()
		child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if child.Start() != nil {
			os.Exit(2)
		}
	}
	end := time.Now().Add(12 * time.Second)
	for time.Now().Before(end) {
		if _, e := os.Stat(filepath.Join(dir, "cleanup-release")); e == nil {
			if child != nil && child.Wait() != nil {
				os.Exit(2)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	os.Exit(2)
}

func stableObserve(t *testing.T, ctx context.Context, pid int) recoveryProcess {
	t.Helper()
	out, e := execute(ctx, recoveryObserver, []string{strconv.Itoa(pid)}, 4096)
	var p recoveryProcess
	if e != nil || json.Unmarshal(out.data, &p) != nil || p.PID != pid || p.BootSec <= 0 ||
		p.Found && (p.StructBytes <= 0 || p.StartSec <= 0) {
		t.Fatal("stable fixture read-only observation", e)
	}
	return p
}
func stableBoundary(t *testing.T, ctx context.Context, check func() bool) {
	t.Helper()
	for ctx.Err() == nil {
		if check() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("stable fixture original work bound")
}

func TestNativeStableCutController(t *testing.T) {
	a := flag.Args()
	if len(a) != 3 {
		return
	}
	recoveryOptIn(t)
	mode, dir, lease := a[0], a[1], a[2]
	containment := mode == "containment-stopped-guard" || mode == "containment-guard-first"
	if !containment && mode != "supervision-guard-first" && mode != "supervision-supervisor-first" {
		t.Fatal("fixed stable cut mode")
	}
	// A fresh process has no earlier fixture children. No broad reap is allowed
	// until all original command/observer Waits actually join after sealing.
	if acquireNativeReaper() != nil {
		t.Fatal("stable self reaper acquisition")
	}
	s, e := nativeReaperObservation()
	if e != nil || s.Children != 0 || s.Descendants != 0 {
		t.Fatal("nonempty stable self reaper")
	}
	end := time.Now().Add(12 * time.Second)
	work, cancel := context.WithDeadline(context.Background(), end.Add(-4*time.Second))
	defer cancel()
	cr, cw, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	defer cr.Close()
	defer cw.Close()
	sr, sw, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	defer sr.Close()
	defer sw.Close()
	if cw.SetWriteDeadline(end) != nil || sr.SetReadDeadline(end.Add(-4*time.Second)) != nil {
		t.Fatal("bounded stable pipes")
	}
	var top *exec.Cmd
	var cap *fixtureReservation
	var waitGate = make(chan struct{})
	var releaseOnce sync.Once
	actualJoined := make(chan struct{})
	exchangeDone := make(chan error, 1)
	born := make(chan *exec.Cmd, 1)
	cleanupDone := false
	// All cleanup shares end. Missing original joins are debt and forbid raw
	// Wait4(-1); VM teardown alone can never satisfy cleanup_verified.
	cleanup := func() {
		if cleanupDone {
			return
		}
		cleanupDone = true
		if cap != nil {
			cap.seal()
		}
		cancel()
		_, _ = cw.Write([]byte{'S'})
		_ = cw.Close()
		releaseOnce.Do(func() { close(waitGate) })
		if top == nil && !containment {
			select {
			case top = <-born:
			case <-time.After(max(0, time.Until(end))):
				t.Error("original supervisor birth/join unknown")
				return
			}
		}
		if top != nil && mode != "supervision-guard-first" {
			if e := top.Process.Kill(); e != nil && !errors.Is(e, os.ErrProcessDone) {
				t.Error("original fixture cleanup stop", e)
			}
		}
		if containment && top != nil {
			go func() { _ = top.Wait(); close(actualJoined) }()
		}
		if top != nil {
			select {
			case <-actualJoined:
			case <-time.After(max(0, time.Until(end)-time.Second)):
				t.Error("original fixture Wait pending")
				_ = top.Process.Kill() // original handle, failed cleanup only
				select {
				case <-actualJoined:
				case <-time.After(max(0, time.Until(end))):
					return // unknown Wait forbids broad reap
				}
			}
		}
		if mode == "supervision-guard-first" && (top.ProcessState == nil || !top.ProcessState.Exited() || top.ProcessState.ExitCode() != 1) {
			t.Error("original supervisor did not naturally deny killed guard")
		}
		if !containment {
			select {
			case e := <-exchangeDone:
				if e == nil {
					t.Error("cut supervisor exchange succeeded")
				}
			case <-time.After(max(0, time.Until(end))):
				t.Error("original exchange pending")
				return
			}
		}
		if os.WriteFile(filepath.Join(dir, "cleanup-release"), nil, 0600) != nil {
			t.Error("private fixture release")
			return
		}
		// This is isolated fixture disposal, not successful production retirement.
		// Keep drainNativeReaper's residual-denial semantics unchanged.
		e := probeCommands.drainIdle(func() error {
			for time.Now().Before(end) {
				for {
					if !time.Now().Before(end) {
						return ErrCleanupDebt
					}
					var status unix.WaitStatus
					pid, e := unix.Wait4(-1, &status, unix.WNOHANG, nil)
					if e == unix.EINTR {
						continue
					}
					if e != nil && e != unix.ECHILD {
						return e
					}
					if pid <= 0 {
						break
					}
				}
				s, e := nativeReaperObservation()
				if e != nil {
					return e
				}
				if s.Children == 0 && s.Descendants == 0 {
					return nil
				}
				k := nativeReaperKill{Signal: int32(syscall.SIGKILL)}
				if e := ownProcctl(6, unsafe.Pointer(&k)); e != nil {
					return e
				}
				time.Sleep(5 * time.Millisecond)
			}
			return ErrCleanupDebt
		})
		if e != nil {
			t.Error("isolated fixture self-scope cleanup unknown", e)
		}
	}
	t.Cleanup(cleanup)
	if containment {
		top = exec.Command(os.Args[0], "-test.run=^TestNativeStableLeaseOwner$", "--", lease, dir)
		top.Env = os.Environ()
		top.ExtraFiles = []*os.File{cr, sw}
		top.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if top.Start() != nil {
			top = nil
			t.Fatal("original stable lease owner birth")
		}
	} else {
		intent := supervisorLiveIntent(t)
		encoded, _ := storage.EncodePropertyIntent(intent)
		owner, e := bindBootOwner(readNativeBoot)
		if e != nil {
			t.Fatal(e)
		}
		go func() {
			exchangeDone <- exchangePropertySupervisorWait(work, intent, encoded, owner, func(c *exec.Cmd) {
				c.Args = []string{os.Args[0], supervisorFixtureArgument, "stable-escaped", dir}
				c.Env = os.Environ()
				c.ExtraFiles = append(c.ExtraFiles, cr, sw)
			}, func(c *exec.Cmd) error { born <- c; <-waitGate; e := c.Wait(); close(actualJoined); return e })
		}()
		select {
		case top = <-born:
		case <-work.Done():
			t.Fatal("original supervisor birth unavailable")
		}
	}
	_ = cr.Close()
	_ = sw.Close()
	var probe, leaf recoveryProcess
	stableBoundary(t, work, func() bool {
		p, e := os.ReadFile(filepath.Join(dir, "probe"))
		if e != nil {
			return false
		}
		q, e := os.ReadFile(filepath.Join(dir, "child"))
		if e != nil {
			return false
		}
		pid, e := strconv.Atoi(string(p))
		if e != nil || pid <= 1 {
			t.Fatal("bounded probe witness")
		}
		cid, e := strconv.Atoi(string(q))
		if e != nil || cid <= 1 {
			t.Fatal("bounded child witness")
		}
		probe = stableObserve(t, work, pid)
		leaf = stableObserve(t, work, cid)
		return probe.Found && leaf.Found && !probe.Zombie && !leaf.Zombie && leaf.PPID == probe.PID && probe.UID == 0 && probe.RUID == 0 && leaf.UID == 0 && leaf.RUID == 0
	})
	if _, e := cw.Write([]byte{'H'}); e != nil {
		t.Fatal("held original stop request")
	}
	var frame [8]byte
	if _, e := io.ReadFull(sr, frame[:]); e != nil {
		t.Fatal("born original guard stop is not held", e)
	}
	var tail [1]byte
	if n, e := sr.Read(tail[:]); n != 0 || e != io.EOF {
		t.Fatal("duplicate/partial original birth frame")
	}
	guardPID := fixtureBirthPID(frame)
	if guardPID < 2 || guardPID > 1<<31-1 {
		t.Fatal("original guard birth frame")
	}
	// The only direct-PID capability originates in the original born callback.
	// Every retirement must cross the still-closed stop gate before sole Wait.
	// Original top Wait also remains gated, including on death.
	// There is no other consuming ancestor waiter in this fresh self-reaper scope.
	cap = &fixtureReservation{end: end.Add(-4 * time.Second), positive: func(signal int) error { return unix.Kill(int(guardPID), syscall.Signal(signal)) }}
	guard := stableObserve(t, work, int(guardPID))
	parent := stableObserve(t, work, top.Process.Pid)
	if probe.Zombie || leaf.Zombie || leaf.PPID != probe.PID || probe.RUID != 0 || leaf.RUID != 0 || !guard.Found || guard.PID != probe.PPID || guard.PPID != parent.PID || !parent.Found || parent.Zombie ||
		guard.ReaperError != 0 || guard.Reaper != guard.PID || guard.Descendants < 2 || leaf.PGID == probe.PGID || probe.UID != 0 || leaf.UID != 0 {
		t.Fatal("held original lineage differs")
	}
	if !containment && (parent.Reaper != parent.PID || parent.ReaperFlags != 1) {
		t.Fatal("original supervisor not reaper")
	}
	if containment {
		if other, e := storage.OpenStateDirectory(lease); e == nil {
			_ = other.Close()
			t.Fatal("live singleton not retained")
		}
	}
	if mode == "containment-stopped-guard" {
		if cap.signal(int(syscall.SIGSTOP)) != nil {
			t.Fatal("actual held guard stop")
		}
		stableBoundary(t, work, func() bool { p := stableObserve(t, work, guard.PID); return recoverySame(p, guard) && p.Stopped })
	} else {
		if mode == "containment-guard-first" || mode == "supervision-supervisor-first" {
			if top.Process.Signal(syscall.SIGSTOP) != nil {
				t.Fatal("original parent stop")
			}
			stableBoundary(t, work, func() bool { p := stableObserve(t, work, parent.PID); return recoverySame(p, parent) && p.Stopped })
		}
		if cap.signal(int(syscall.SIGKILL)) != nil {
			t.Fatal("actual held guard kill")
		}
		stableBoundary(t, work, func() bool {
			p := stableObserve(t, work, guard.PID)
			return recoverySame(p, guard) && p.Zombie
		})
	}
	if containment || mode == "supervision-supervisor-first" {
		if top.Process.Kill() != nil {
			t.Fatal("original owner death cut")
		}
		// Non-consuming death evidence. Do not Cmd.Wait/reap an ancestry anchor yet.
		stableBoundary(t, work, func() bool { p := stableObserve(t, work, parent.PID); return recoverySame(p, parent) && p.Zombie })
	}
	probeLive, childLive := false, false
	if containment {
		store, e := storage.OpenStateDirectory(lease)
		if e != nil {
			t.Fatal("singleton reacquisition after actual owner death", e)
		}
		defer store.Close()
		p, c := stableObserve(t, work, probe.PID), stableObserve(t, work, leaf.PID)
		for _, v := range [][2]recoveryProcess{{p, probe}, {c, leaf}} {
			if v[0].Found && !v[0].Zombie && !recoverySame(v[0], v[1]) {
				t.Fatal("original descendant attribution changed")
			}
		}
		probeLive, childLive = recoverySame(p, probe) && !p.Zombie, recoverySame(c, leaf) && !c.Zombie
		if mode == "containment-guard-first" && !probeLive && !childLive {
			t.Fatal("missing actual surviving descendant counterexample")
		}
		if mode == "containment-stopped-guard" && cap.signal(int(syscall.SIGCONT)) != nil {
			t.Fatal("held original guard resume")
		}
		if mode == "containment-stopped-guard" {
			// Qualify the actual resumed guard's own drain before fixture disposal.
			stableBoundary(t, work, func() bool {
				quiet := true
				for _, original := range []recoveryProcess{guard, probe, leaf} {
					p := stableObserve(t, work, original.PID)
					if p.Found && !p.Zombie {
						if !recoverySame(p, original) {
							t.Fatal("post-resume attribution changed")
						}
						quiet = false
					}
				}
				return quiet
			})
		}
	}
	cap.seal() // irreversible: no direct PID operation after this point
	cleanup()
	if t.Failed() {
		return
	}
	fmt.Println("STABLE_FIXTURE_RESULT mode=" + mode + " cleanup=verified")
	if containment {
		name := "guard-first"
		if mode == "containment-stopped-guard" {
			name = "stopped-guard"
		}
		b, _ := json.Marshal(struct {
			Mode  string `json:"mode"`
			Lock  bool   `json:"singleton_reacquired"`
			Probe bool   `json:"original_probe_live"`
			Child bool   `json:"original_child_live"`
			Clean bool   `json:"cleanup_verified"`
		}{name, true, probeLive, childLive, true})
		fmt.Println("EXECUTOR_CONTAINMENT " + string(b))
	}
}
