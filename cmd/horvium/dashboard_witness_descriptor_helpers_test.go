//go:build freebsd || darwin || linux || openbsd || netbsd || dragonfly

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"golang.org/x/sys/unix"
	"net"
	"os"
)

func witnessPassedDescriptor(c *net.UnixConn) (*net.UnixConn, error) {
	var tag [1]byte
	oob := make([]byte, unix.CmsgSpace(4)*2)
	n, on, flags, _, readErr := c.ReadMsgUnix(tag[:], oob)
	messages, e := unix.ParseSocketControlMessage(oob[:on])
	if e != nil {
		return nil, errDashboardWitness
	}
	var descriptors []int
	for _, message := range messages {
		if message.Header.Level != unix.SOL_SOCKET || message.Header.Type != unix.SCM_RIGHTS {
			e = errDashboardWitness
			continue
		}
		fds, fe := unix.ParseUnixRights(&message)
		if fe != nil {
			e = fe
		}
		descriptors = append(descriptors, fds...)
	}
	defer func() {
		for _, fd := range descriptors {
			_ = unix.Close(fd)
		}
	}()
	if e != nil || readErr != nil || n != 1 || tag[0] != 'F' || flags&unix.MSG_CTRUNC != 0 || len(messages) != 1 || len(descriptors) != 1 {
		return nil, errDashboardWitness
	}
	fd := descriptors[0]
	unix.CloseOnExec(fd)
	if flags, e := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0); e != nil || flags&unix.FD_CLOEXEC == 0 {
		return nil, errDashboardWitness
	}
	f := os.NewFile(uintptr(fd), "owned-helper-handoff")
	// Transfer ownership to os.File, and close that wrapper explicitly. Closing
	// fd with unix.Close alone leaves its finalizer able to close a reused number.
	descriptors = nil
	conn, e := net.FileConn(f)
	ce := f.Close()
	if e != nil || ce != nil {
		if conn != nil {
			conn.Close()
		}
		return nil, errDashboardWitness
	}
	u, ok := conn.(*net.UnixConn)
	if !ok {
		conn.Close()
		return nil, errDashboardWitness
	}
	return u, nil
}
