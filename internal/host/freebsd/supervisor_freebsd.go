// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

// Native layouts from qualified sys/procctl.h; amd64 only until native evidence
// supports another ABI. We never target another process or a saved PID.
type nativeReaperStatus struct {
	Flags, Children, Descendants uint32
	Reaper, PID                  int32
	Pad                          [15]uint32
}
type nativeReaperKill struct {
	Signal    int32
	Flags     uint32
	Subtree   int32
	Killed    uint32
	FailedPID int32
	Pad       [15]uint32
}

func ownProcctl(command uintptr, data unsafe.Pointer) error {
	if runtime.GOARCH != "amd64" {
		return ErrUnsupported
	}
	_, _, e := unix.Syscall6(unix.SYS_PROCCTL, 0, 0, command, uintptr(data), 0, 0)
	if e != 0 {
		return ErrUnavailable
	}
	return nil
}
func acquireNativeReaper() error {
	if os.Getuid() != 0 || os.Geteuid() != 0 {
		return ErrPermission
	}
	if err := ownProcctl(2, nil); err != nil {
		return err
	}
	_, err := nativeReaperObservation()
	return err
}
func nativeReaperObservation() (nativeReaperStatus, error) {
	var s nativeReaperStatus
	if err := ownProcctl(4, unsafe.Pointer(&s)); err != nil {
		return s, err
	}
	if s.Flags != 1 || s.Reaper != int32(os.Getpid()) {
		return s, ErrUnavailable
	}
	return s, nil
}
func drainNativeReaper() error {
	deadline := time.Now().Add(2 * time.Second)
	residual := false
	for time.Now().Before(deadline) {
		var status unix.WaitStatus
		// The guard is already waited by exec.Cmd. Reap only this isolated supervisor's
		// adopted children, including unexpected escaped/nested-reaper survivors.
		for {
			pid, e := unix.Wait4(-1, &status, unix.WNOHANG, nil)
			if e == unix.EINTR {
				continue
			}
			if e != nil && e != unix.ECHILD {
				return ErrUnavailable
			}
			if pid <= 0 {
				break
			}
		}
		s, err := nativeReaperObservation()
		if err != nil {
			return err
		}
		if s.Children == 0 && s.Descendants == 0 {
			if residual {
				return ErrUnavailable
			}
			return nil
		}
		residual = true
		k := nativeReaperKill{Signal: int32(syscall.SIGKILL)}
		if err = ownProcctl(6, unsafe.Pointer(&k)); err != nil {
			return err
		}
		time.Sleep(5 * time.Millisecond)
	}
	return ErrUnavailable
}
func supervisorPipe(fd uintptr) *os.File {
	unix.CloseOnExec(int(fd))
	// exec.Cmd exposes inherited File.Fd values in blocking mode. Restore
	// nonblocking before NewFile so the original channel supports bounded deadlines.
	if unix.SetNonblock(int(fd), true) != nil {
		return nil
	}
	f := os.NewFile(fd, "private-property-supervisor")
	if f == nil {
		return nil
	}
	var st unix.Stat_t
	flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if unix.Fstat(int(fd), &st) != nil || err != nil || !validSupervisorPipe(uint32(st.Mode), st.Uid, uint64(st.Nlink), flags) {
		f.Close()
		return nil
	}
	return f
}

// PropertySupervisorProcess recognizes only the private reexecution entrypoint.
// Root-only inherited pipes and validated intent; no public executable selector.
// Normal commands never enter this path. Admission remains the parent backend's
// obligation; this component cannot supply missing journal/global clearance.
func PropertySupervisorProcess(args []string) (bool, int) {
	if len(args) == 0 || args[0] != supervisorArgument {
		return false, 0
	}
	if len(args) != 1 || os.Getuid() != 0 || os.Geteuid() != 0 {
		return true, 1
	}
	request, response := supervisorPipe(3), supervisorPipe(4)
	if request == nil || response == nil {
		return true, 1
	}
	defer request.Close()
	defer response.Close()
	return true, runPropertySupervisor(request, response, func(ctx context.Context, i storage.PropertyIntent) error {
		args, err := propertySetArguments(i)
		if err != nil {
			return err
		}
		out, err := executeNative(ctx, "/sbin/zfs", args, 4096)
		if err != nil {
			return err
		}
		if len(out.data) != 0 {
			return ErrUnavailable
		}
		return nil
	})
}
func runPropertySupervisor(request, response *os.File, run func(context.Context, storage.PropertyIntent) error) int {
	deadline := time.Now().Add(10 * time.Second)
	if request.SetReadDeadline(deadline) != nil || response.SetWriteDeadline(deadline.Add(2*time.Second)) != nil {
		return 1
	}
	token, boot, intent, err := readSupervisorRequest(request)
	if err != nil {
		return 1
	}
	owner, err := bindBootOwner(readNativeBoot)
	if err != nil || owner.identity != boot || acquireNativeReaper() != nil {
		return 1
	}
	ready := supervisorFrame(1, token, boot)
	if _, err = response.Write(ready[:]); err != nil {
		return 1
	}
	if readSupervisorFrame(request, 2, token, boot) != nil {
		return 1
	}
	var tail [1]byte
	if n, e := request.Read(tail[:]); n != 0 || e != io.EOF {
		return 1
	}
	if owner.check() != nil || time.Now().Before(intent.Review().CreatedAt) || !time.Now().Before(intent.Review().ExpiresAt) || run == nil {
		return 1
	}
	ctx, stop := context.WithDeadline(context.Background(), deadline)
	defer stop()
	err = run(ctx, intent)
	cleanup := probeCommands.drainIdle(drainNativeReaper)
	if err != nil || cleanup != nil || owner.check() != nil {
		return 1
	}
	complete := supervisorFrame(3, token, boot)
	if _, err = response.Write(complete[:]); err != nil {
		return 1
	}
	return 0
}

func propertySetSupervised(ctx context.Context, intent storage.PropertyIntent) error {
	if ctx == nil {
		return storage.ErrInvalid
	}
	if os.Getuid() != 0 || os.Geteuid() != 0 {
		return ErrPermission
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	encoded, err := storage.EncodePropertyIntent(intent)
	if err != nil {
		return err
	}
	if _, err = propertySetArguments(intent); err != nil {
		return err
	}
	owner, err := bindBootOwner(readNativeBoot)
	if err != nil {
		return err
	}
	return exchangePropertySupervisor(ctx, intent, encoded, owner, nil)
}

// Native tests may select their own fixed child entrypoint; production always
// reexecutes the current executable's narrow property-only implementation.
func exchangePropertySupervisor(ctx context.Context, intent storage.PropertyIntent, encoded []byte, owner *bootOwner, prepare func(*exec.Cmd)) error {
	if owner.check() != nil {
		return ErrUnavailable
	}
	ctx, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	deadline, _ := ctx.Deadline()
	reader, writer, err := os.Pipe()
	if err != nil {
		return ErrUnavailable
	}
	defer reader.Close()
	defer writer.Close()
	statusReader, statusWriter, err := os.Pipe()
	if err != nil {
		return ErrUnavailable
	}
	defer statusReader.Close()
	defer statusWriter.Close()
	if writer.SetWriteDeadline(deadline) != nil || statusReader.SetReadDeadline(deadline) != nil {
		return ErrUnavailable
	}
	executable, err := os.Executable()
	if err != nil {
		return ErrUnavailable
	}
	cmd := exec.Command(executable, supervisorArgument)
	cmd.Env = []string{"LC_ALL=C", "PATH=/sbin:/bin:/usr/sbin:/usr/bin"}
	cmd.ExtraFiles = []*os.File{reader, statusWriter}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if prepare != nil {
		prepare(cmd)
	}
	if err = cmd.Start(); err != nil {
		return ErrUnavailable
	}
	reader.Close()
	statusWriter.Close()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	waited := false
	defer func() {
		writer.Close()
		statusReader.Close()
		if !waited {
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				_ = cmd.Process.Kill()
				select {
				case <-done:
				case <-time.After(3 * time.Second):
				}
			}
		}
	}()
	var token [16]byte
	if _, err = rand.Read(token[:]); err != nil {
		return ErrUnavailable
	}
	if writeSupervisorRequest(writer, token, owner.identity, encoded) != nil || readSupervisorFrame(statusReader, 1, token, owner.identity) != nil {
		return ErrUnavailable
	}
	if ctx.Err() != nil || owner.check() != nil || time.Now().Before(intent.Review().CreatedAt) || !time.Now().Before(intent.Review().ExpiresAt) {
		return ErrUnavailable
	}
	goFrame := supervisorFrame(2, token, owner.identity)
	if _, err = writer.Write(goFrame[:]); err != nil {
		return ErrUnavailable
	}
	writer.Close()
	if readSupervisorFrame(statusReader, 3, token, owner.identity) != nil {
		return ErrUnavailable
	}
	select {
	case err = <-done:
		waited = true
	case <-ctx.Done():
		return ErrUnavailable
	}
	if err != nil || owner.check() != nil {
		return ErrUnavailable
	}
	var tail [1]byte
	if n, e := statusReader.Read(tail[:]); n != 0 || !errors.Is(e, io.EOF) {
		return ErrUnavailable
	}
	return nil
}
