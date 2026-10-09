// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if handled, code := nativeSupervisorFixtureProcess(os.Args[1:]); handled {
		os.Exit(code)
	}
	if handled, code := PropertySupervisorProcess(os.Args[1:]); handled {
		os.Exit(code)
	}
	os.Exit(m.Run())
}
