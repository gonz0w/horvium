// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package freebsd

func PropertySupervisorProcess(args []string) (bool, int) {
	if len(args) > 0 && args[0] == supervisorArgument {
		return true, 1
	}
	return false, 0
}
