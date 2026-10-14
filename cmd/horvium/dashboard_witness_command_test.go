// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestDashboardWitnessCommandProbe(t *testing.T) {
	mode := os.Getenv("HORVIUM_WITNESS_COMMAND_PROBE")
	if mode == "" {
		return
	}
	switch mode {
	case "kernel-idle":
		os.Stdout.WriteString("ready\n")
		time.Sleep(30 * time.Second)
		os.Exit(0)
	case "exit":
		os.Stderr.WriteString("original command error")
		os.Exit(7)
	case "overflow":
		os.Stdout.WriteString(strings.Repeat("x", witnessFrameCap+1))
		os.Exit(0)
	case "stderr-overflow":
		os.Stderr.WriteString(strings.Repeat("x", 4097))
		os.Exit(0)
	case "deadline":
		time.Sleep(5 * time.Second)
		os.Exit(0)
	case "success":
		os.Stdout.WriteString("original sample")
		os.Exit(0)
	default:
		os.Exit(9)
	}
}

func TestDashboardWitnessCommandRetainsBoundedFailure(t *testing.T) {
	for _, mode := range []string{"success", "exit", "overflow", "deadline", "start", "stderr-overflow", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			if mode == "deadline" {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
			}
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			executable, e := os.Executable()
			if e != nil {
				t.Fatal(e)
			}
			if mode == "start" {
				executable = t.TempDir() + "/absent"
			}
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestDashboardWitnessCommandProbe$")
			cmd.Env = append(os.Environ(), "HORVIUM_WITNESS_COMMAND_PROBE="+mode)
			b, diagnostic, e := witnessRunSampleCommand(ctx, cmd)
			if len(b) > witnessFrameCap || len(diagnostic.Stderr) > 4096 {
				t.Fatal("unbounded command original")
			}
			if mode == "success" {
				if e != nil || string(b) != "original sample" || diagnostic.Reason != "" || diagnostic.ExitCode != 0 || !diagnostic.StatePresent {
					t.Fatal("successful original rejected", diagnostic, e)
				}
				return
			}
			want := mode
			if mode == "stderr-overflow" {
				want = "overflow"
			}
			if e == nil || diagnostic.Reason != want {
				t.Fatal("failure reason lost", diagnostic, e)
			}
			if mode == "exit" && (diagnostic.ExitCode != 7 || diagnostic.Stderr != "original command error") {
				t.Fatal("original exit/stderr lost", diagnostic)
			}
			if mode == "overflow" && (!diagnostic.StdoutOverflow || len(b) != witnessFrameCap) {
				t.Fatal("oversized original accepted or bounded prefix lost", diagnostic)
			}
			if mode == "stderr-overflow" && (!diagnostic.StderrOverflow || len(diagnostic.Stderr) != 4096) {
				t.Fatal("bounded stderr cause lost")
			}
			if mode == "cancelled" && !diagnostic.Cancelled {
				t.Fatal("cancellation cause lost")
			}
			if mode == "deadline" && !diagnostic.DeadlineExceeded {
				t.Fatal("deadline cause lost", diagnostic)
			}
			if mode == "start" && diagnostic.StatePresent {
				t.Fatal("unstarted command has fabricated wait state")
			}
		})
	}
}
