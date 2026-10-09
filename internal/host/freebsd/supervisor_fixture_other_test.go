// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package freebsd

func nativeSupervisorFixtureProcess([]string) (bool, int) { return false, 0 }
