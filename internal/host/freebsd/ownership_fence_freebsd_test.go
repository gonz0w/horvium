// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// Each fault gets its own actual process and private slot. A sticky status-1
// fence must never be reset in the parent to make another case pass.
func TestNativeExecutorOwnershipFences(t *testing.T) {
	recoveryOptIn(t)
	for _, mode := range []string{"guard-status-one", "pending-proof"} {
		if !t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNativeExecutorOwnershipFenceChild$", "-test.timeout=7s", "--", mode, t.TempDir())
			cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin", "HORVIUM_NATIVE_VM=1", "HORVIUM_PROPERTY_VM=1", "HORVIUM_RESEARCH_VM=1"}
			output := &capture{limit: 16 << 10, cancel: cancel}
			cmd.Stdout, cmd.Stderr = sink{capture: output, keep: true}, sink{capture: output, keep: true}
			if err := cmd.Run(); err != nil {
				t.Fatalf("isolated native ownership fence failed: %v; bounded owned output=%q", err, output.out.String())
			}
			if output.exceeded {
				t.Fatal("isolated native ownership output cap")
			}
		}) {
			return
		}
	}
}

func TestNativeExecutorOwnershipFenceChild(t *testing.T) {
	args := flag.Args()
	if len(args) != 2 {
		return
	}
	recoveryOptIn(t)
	mode, dir := args[0], args[1]
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var initial error
	var release func()
	if mode == "guard-status-one" {
		type guardStatus struct {
			known, exited bool
			code          int
			proof         error
		}
		observed := make(chan guardStatus, 1)
		out, err := executePrepared(ctx, "/usr/bin/false", nil, 128, func(cmd *exec.Cmd, control *processControl) func(error) error {
			classify := prepareNativeCommand(ctx, "/usr/bin/false", nil)(cmd, control)
			actual := control.proof
			control.proof = func(waitErr error, end time.Time) error {
				proof := actual(waitErr, end)
				status := guardStatus{proof: proof}
				if cmd.ProcessState != nil { // Access only after the actual original Wait.
					status.known, status.exited, status.code = true, cmd.ProcessState.Exited(), cmd.ProcessState.ExitCode()
				}
				observed <- status
				return proof
			}
			return classify
		})
		initial = err
		select {
		case status := <-observed:
			if !status.known || !status.exited || status.code != 1 || !errors.Is(status.proof, ErrCleanupDebt) || !errors.Is(initial, ErrExit) || len(out.data) != 0 {
				t.Fatal("actual normal guard status one/output suppression not established", status, initial)
			}
		default:
			t.Fatal("actual status-one guard wait/proof did not finish before caller return", initial)
		}
	} else if mode == "pending-proof" {
		ready, gate := make(chan error, 1), make(chan struct{})
		var once sync.Once
		release = func() { once.Do(func() { close(gate) }) }
		defer release()
		type callResult struct {
			err       error
			published int
		}
		done := make(chan callResult, 1)
		go func() {
			out, err := executePrepared(ctx, "/usr/bin/true", nil, 128, func(cmd *exec.Cmd, control *processControl) func(error) error {
				classify := prepareNativeCommand(ctx, "/usr/bin/true", nil)(cmd, control)
				actual := control.proof
				control.proof = func(waitErr error, end time.Time) error {
					proof := actual(waitErr, end) // Real guard Wait, readers and group proof first.
					ready <- proof
					<-gate // Delay only acknowledgement; never synthesize actual proof.
					return proof
				}
				return classify
			})
			done <- callResult{err: err, published: len(out.data)}
		}()
		select {
		case proof := <-ready:
			if proof != nil {
				t.Fatal("actual native retirement proof failed", proof)
			}
		case <-ctx.Done():
			t.Fatal("actual native proof did not reach the fixture gate")
		}
		select {
		case result := <-done:
			if result.published != 0 {
				t.Fatal("expired native call published output")
			}
			initial = result.err
		case <-ctx.Done():
			t.Fatal("caller did not respect its finite original retirement reserve")
		}
	} else {
		t.Fatal("unknown private native ownership case")
	}
	if !errors.Is(initial, ErrCleanupDebt) {
		t.Fatal("uncertain guard/proof did not fence admission", initial)
	}
	probeCommands.mu.Lock()
	original := probeCommands.owner
	probeCommands.mu.Unlock()
	if original == nil {
		t.Fatal("original native owner was discarded")
	}
	// If admission regresses, this fixed harmless command leaves an owned witness.
	witness := filepath.Join(dir, "forbidden-next-launch")
	var nextCmd *exec.Cmd
	_, next := executePrepared(ctx, "/usr/bin/touch", []string{witness}, 128, func(cmd *exec.Cmd, control *processControl) func(error) error {
		nextCmd = cmd // Preparation is not a birth witness; Process below is.
		return prepareNativeCommand(ctx, "/usr/bin/touch", []string{witness})(cmd, control)
	})
	if !errors.Is(next, ErrCleanupDebt) {
		t.Fatal("next native Start was admitted", next)
	}
	if nextCmd == nil || nextCmd.Process != nil {
		t.Fatal("denied dispatch acquired an actual process")
	}
	if _, err := os.Lstat(witness); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("next native command executed or its absence is unknown", err)
	}
	probeCommands.mu.Lock()
	same := probeCommands.owner == original
	probeCommands.mu.Unlock()
	if !same {
		t.Fatal("pending native owner was replaced")
	}
	drained := false
	if err := probeCommands.drainIdle(func() error { drained = true; return nil }); !errors.Is(err, ErrCleanupDebt) || drained {
		t.Fatal("outer native reaper was admitted while an original owner remained")
	}
	if release != nil {
		release()
		awaitSlot(t, &probeCommands, true)
		select {
		case terminal := <-original.done:
			if !terminal.clean || !errors.Is(terminal.err, context.DeadlineExceeded) {
				t.Fatal("late actual proof lost expired-call failure", terminal)
			}
		case <-ctx.Done():
			t.Fatal("original late native retirement did not finish")
		}
		if _, err := executeNative(ctx, "/usr/bin/true", nil, 128); err != nil {
			t.Fatal("late actual proof did not release later admission", err)
		}
	}
}
