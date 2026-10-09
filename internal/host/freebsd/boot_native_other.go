// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package freebsd

func readNativeBoot() (bootIdentity, error) { return bootIdentity{}, ErrUnsupported }
