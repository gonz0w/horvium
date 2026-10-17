//go:build freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/admin"
)

type witnessRawSourceReceipt struct {
	DataClass                                    string
	Actor                                        witnessActor
	Socket                                       witnessFileObject
	PeerUID                                      uint32
	PeerPID                                      int
	Request                                      []byte
	RequestID                                    string
	StartedNS, DeadlineNS, FinishedNS, SentBytes int64
	SentSHA256                                   string
	WriteError, CloseWriteError                  witnessForwardError
}

// A separate generated source fixture, not the trusted normal producer. The
// product helper cannot normally exceed its derived wire budget. Never count
// these zero bytes as producer accounting, a valid reply or a workload sample.
func TestNativeDashboardWitnessRawProducer(t *testing.T) {
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_PERF_VM") != "1" {
		t.Skip("disposable VM opt-in")
	}
	if os.Getuid() != 0 || len(flag.Args()) != 1 {
		t.Fatal("owned root source config required")
	}
	f, e := os.Open(flag.Args()[0])
	if e != nil {
		t.Fatal(e)
	}
	cfg, e := admin.ParseConfig(f)
	f.Close()
	if e != nil || cfg.TimeoutSeconds != 45 {
		t.Fatal("existing source config / deadline", e)
	}
	f, e = os.Open("../../.tools/witness-probe-original.json")
	if e != nil {
		t.Fatal(e)
	}
	var probe witnessProbeReceipt
	e = witnessControlRead(f, &probe)
	end := witnessEOF(f)
	info, se := f.Stat()
	f.Close()
	if e != nil || end != nil || se != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0400 || !probe.ABI.valid() {
		t.Fatal("original native ABI receipt")
	}
	obj, e := witnessObject(info)
	if e != nil || obj.UID != 0 || obj.Links != 1 {
		t.Fatal("native probe custody")
	}
	if _, e = os.Lstat(cfg.HelperSocket); !os.IsNotExist(e) {
		t.Fatal("source socket already exists")
	}
	listener, e := net.ListenUnix("unix", &net.UnixAddr{Name: cfg.HelperSocket, Net: "unix"})
	if e != nil {
		t.Fatal(e)
	}
	listener.SetUnlinkOnClose(false)
	if os.Chown(cfg.HelperSocket, 0, int(cfg.SocketGID)) != nil || os.Chmod(cfg.HelperSocket, 0660) != nil {
		listener.Close()
		t.Fatal("source socket ownership")
	}
	socket, e := witnessRootSocket(cfg.HelperSocket, cfg.SocketGID)
	if e != nil {
		listener.Close()
		t.Fatal(e)
	}
	defer func() {
		listener.Close()
		current, e := os.Lstat(cfg.HelperSocket)
		if e != nil || !os.SameFile(socket, current) || os.Remove(cfg.HelperSocket) != nil {
			t.Error("original source socket cleanup")
		}
	}()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			listener.Close()
		case <-done:
		}
	}()
	conn, e := listener.AcceptUnix()
	if e != nil {
		t.Fatal("one original connector", e)
	}
	defer conn.Close()
	r := witnessRawSourceReceipt{DataClass: "synthetic-oversized-producer-control"}
	r.StartedNS = time.Now().UnixNano()
	r.DeadlineNS = time.Unix(0, r.StartedNS).Add(time.Duration(cfg.TimeoutSeconds) * time.Second).UnixNano()
	if conn.SetDeadline(time.Unix(0, r.DeadlineNS)) != nil {
		t.Fatal("source original deadline")
	}
	r.Socket, e = witnessObject(socket)
	if e != nil {
		t.Fatal(e)
	}
	r.Actor, e = witnessObserveActor(os.Getpid(), probe.ABI)
	if e != nil {
		t.Fatal(e)
	}
	r.PeerUID, r.PeerPID, e = witnessNativePeer(conn, probe.ABI)
	if e != nil || r.PeerUID != cfg.ManagementUID || r.PeerPID <= 0 {
		t.Fatal("actual connector credentials", e)
	}
	// The connecting process is the dropped-privilege connector, already reaped
	// before the observer sends the request. Its PID is compared to the actual
	// observer wait/identity receipt, not guessed to be the manager PID.
	r.Request, e = io.ReadAll(io.LimitReader(conn, witnessFrameCap+5))
	if e != nil {
		t.Fatal("original source request", e)
	}
	r.RequestID, e = dashboardWitnessRequest(bytes.NewReader(r.Request))
	if e != nil {
		t.Fatal(e)
	}
	sink := newWitnessForwardWriter(conn)
	n, we := io.CopyBuffer(sink, io.LimitReader(witnessCapacityZero{}, int64(witnessRawCap)+1), make([]byte, witnessFrameCap))
	r.FinishedNS = time.Now().UnixNano()
	r.SentBytes = n
	r.SentSHA256 = fmt.Sprintf("%x", sink.hash.Sum(nil))
	r.WriteError = witnessClassifyForwardError(we)
	r.CloseWriteError = witnessClassifyForwardError(conn.CloseWrite())
	conn.Close()
	if witnessWritePrivateIndex("../../.tools", "witness-raw-source.json", r) != nil {
		t.Fatal("exclusive source receipt")
	}
	// Keep the original helper actor/socket alive through independent readback.
	// No second request or producer retry is served.
	<-ctx.Done()
}

func startWitnessRawProducer(t *testing.T, cfg admin.Config, configPath, fixture string) *nativeProcess {
	t.Helper()
	cmd := exec.Command(fixture, "-test.run=^TestNativeDashboardWitnessRawProducer$", "-test.count=1", "--", configPath)
	cmd.Env = []string{"HORVIUM_NATIVE_VM=1", "HORVIUM_PERF_VM=1", "LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	p := &nativeProcess{cmd: cmd, done: make(chan error, 1)}
	if cmd.Start() != nil {
		t.Fatal("owned oversized source start")
	}
	go func() { p.done <- cmd.Wait() }()
	t.Cleanup(func() {
		if !p.waited {
			p.stop(t, syscall.SIGTERM, true)
		}
	})
	return p
}
