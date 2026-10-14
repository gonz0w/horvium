//go:build freebsd || darwin || linux || openbsd || netbsd || dragonfly

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"io"
	"net"
	"os"
	"runtime"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func witnessTestPair(t *testing.T) (*net.UnixConn, *net.UnixConn) {
	t.Helper()
	fds, e := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if e != nil {
		t.Fatal(e)
	}
	convert := func(fd int) *net.UnixConn {
		unix.CloseOnExec(fd)
		f := os.NewFile(uintptr(fd), "private-descriptor-control")
		c, e := net.FileConn(f)
		f.Close()
		if e != nil {
			t.Fatal(e)
		}
		u := c.(*net.UnixConn)
		t.Cleanup(func() { u.Close() })
		_ = u.SetDeadline(time.Now().Add(5 * time.Second))
		return u
	}
	return convert(fds[0]), convert(fds[1])
}
func TestDashboardWitnessDescriptorOwnershipSurvivesGC(t *testing.T) {
	sender, receiver := witnessTestPair(t)
	original, other := witnessTestPair(t)
	f, e := original.File()
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = sender.WriteMsgUnix([]byte{'F'}, unix.UnixRights(int(f.Fd())), nil); e != nil {
		t.Fatal(e)
	}
	f.Close()
	passed, e := witnessPassedDescriptor(receiver)
	if e != nil {
		t.Fatal(e)
	}
	defer passed.Close()
	// Force reuse of the received descriptor's now-closed original number. A
	// wrapper finalizer must not close this unrelated object after FileConn dup.
	sentinel, e := os.CreateTemp(witnessPrivateDirectory(t), "sentinel-")
	if e != nil {
		t.Fatal(e)
	}
	defer sentinel.Close()
	for i := 0; i < 3; i++ {
		runtime.GC()
	}
	if _, e := sentinel.Stat(); e != nil {
		t.Fatal("descriptor wrapper closed unrelated reused fd", e)
	}
	if _, e := passed.Write([]byte("original-connection")); e != nil {
		t.Fatal("transferred connection lost", e)
	}
	b := make([]byte, len("original-connection"))
	if _, e := io.ReadFull(other, b); e != nil || string(b) != "original-connection" {
		t.Fatal("unchanged connection")
	}
	if passed.CloseWrite() != nil {
		t.Fatal("transferred half-close")
	}
	var extra [1]byte
	if n, e := other.Read(extra[:]); n != 0 || e != io.EOF {
		t.Fatal("original endpoint EOF")
	}
	raw, e := passed.SyscallConn()
	if e != nil {
		t.Fatal(e)
	}
	flags := 0
	var fe error
	if raw.Control(func(fd uintptr) { flags, fe = unix.FcntlInt(fd, unix.F_GETFD, 0) }) != nil || fe != nil || flags&unix.FD_CLOEXEC == 0 {
		t.Fatal("duplicate lacks CLOEXEC")
	}
}
func TestDashboardWitnessDescriptorRejectsExtraAndWrongTag(t *testing.T) {
	for _, name := range []string{"extra", "tag", "truncated", "non-socket", "closed"} {
		t.Run(name, func(t *testing.T) {
			a, b := witnessTestPair(t)
			x, _ := witnessTestPair(t)
			f, e := x.File()
			if e != nil {
				t.Fatal(e)
			}
			defer f.Close()
			fds := []int{int(f.Fd())}
			tag := byte('F')
			if name == "closed" {
				a.Close()
				if c, e := witnessPassedDescriptor(b); e == nil || c != nil {
					t.Fatal("closed sender accepted")
				}
				return
			}
			if name == "non-socket" {
				temporary, e := os.CreateTemp(witnessPrivateDirectory(t), "not-socket-")
				if e != nil {
					t.Fatal(e)
				}
				defer temporary.Close()
				fds = []int{int(temporary.Fd())}
			}
			if name == "truncated" {
				for i := 0; i < 64; i++ {
					fds = append(fds, int(f.Fd()))
				}
			}
			if name == "extra" {
				fds = append(fds, int(f.Fd()))
			} else if name == "tag" {
				tag = 'X'
			}
			if _, _, e := a.WriteMsgUnix([]byte{tag}, unix.UnixRights(fds...), nil); e != nil {
				t.Fatal(e)
			}
			if c, e := witnessPassedDescriptor(b); e == nil || c != nil {
				if c != nil {
					c.Close()
				}
				t.Fatal("ambiguous descriptor accepted")
			}
		})
	}
}

func TestDashboardWitnessTransferredEndpointSurvivesSenderClosure(t *testing.T) {
	sender, receiver := witnessTestPair(t)
	original, peer := witnessTestPair(t)
	file, e := original.File()
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e := sender.WriteMsgUnix([]byte{'F'}, unix.UnixRights(int(file.Fd())), nil); e != nil {
		t.Fatal(e)
	}
	file.Close()
	original.Close()
	sender.Close()
	passed, e := witnessPassedDescriptor(receiver)
	if e != nil {
		t.Fatal("sender exit discarded established endpoint", e)
	}
	defer passed.Close()
	if _, e := passed.Write([]byte("original")); e != nil {
		t.Fatal(e)
	}
	var body [8]byte
	if _, e := io.ReadFull(peer, body[:]); e != nil || string(body[:]) != "original" {
		t.Fatal("transferred bytes changed", e)
	}
	receiver.Close()
	if passed.CloseWrite() != nil {
		t.Fatal("EOF after sender exit")
	}
	var extra [1]byte
	if n, e := peer.Read(extra[:]); n != 0 || e != io.EOF {
		t.Fatal("original EOF lost")
	}
}
