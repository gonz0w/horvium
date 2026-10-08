//go:build !freebsd

// SPDX-License-Identifier: BSD-3-Clause
package helper

import "net"

const nativeSupported = false

func peerUID(*net.UnixConn) (uint32, error) { return 0, ErrUnsupported }
func actionPeerQuiet(*net.UnixConn) error   { return ErrUnsupported }
func safeDirectory(Config) error            { return ErrUnsupported }
func safeSocket(Config) error               { return ErrUnsupported }
