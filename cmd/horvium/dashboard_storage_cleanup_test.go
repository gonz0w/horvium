// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestDashboardStorageCleanupObservedSequence(t *testing.T) {
	for _, created := range []bool{true, false} {
		t.Run(map[bool]string{true: "created", false: "creation-aborted"}[created], func(t *testing.T) {
			expected := dashboardCleanupIdentity{"horvium_ui_0123456789abcdef", "md7", "/tmp/fixture/disk.img", "42", created}
			if !created {
				expected.GUID = ""
			}
			commands := [][]string{{"/sbin/mdconfig", "-l", "-n", "-f", expected.File}}
			outputs := []string{"7 \n"}
			if created {
				commands = append(commands, []string{"/sbin/zpool", "get", "-H", "-p", "-o", "value", "guid", expected.Pool}, []string{"/sbin/zpool", "destroy", expected.Pool}, []string{"/sbin/zpool", "list", "-H", "-o", "name"})
				outputs = append(outputs, "42\n", "", "boot\n")
			}
			commands = append(commands, []string{"/sbin/mdconfig", "-l", "-n", "-f", expected.File}, []string{"/sbin/mdconfig", "-d", "-u", "7"}, []string{"/sbin/mdconfig", "-l"})
			outputs = append(outputs, "7 \n", "", "md0 md1 \n")
			n := 0
			receipt, err := dashboardCleanupStorage(expected, func(path string, args ...string) ([]byte, error) {
				if n >= len(commands) || !reflect.DeepEqual(append([]string{path}, args...), commands[n]) {
					t.Fatalf("unexpected cleanup command %d", n)
				}
				data := outputs[n]
				n++
				return []byte(data), nil
			})
			if err != nil || n != len(commands) || !receipt.MDRemoved || receipt.PoolCreated != created || receipt.PoolRemoved != created {
				t.Fatalf("incomplete cleanup observation: %+v %v", receipt, err)
			}
		})
	}
}

func TestDashboardStorageCleanupRejectsEveryFailedCommand(t *testing.T) {
	for failure := 0; failure < 7; failure++ {
		t.Run(string(rune('A'+failure)), func(t *testing.T) {
			outputs := []string{"7\n", "42\n", "", "boot\n", "7\n", "", "md0\n"}
			n := 0
			receipt, err := dashboardCleanupStorage(dashboardCleanupIdentity{"horvium_ui_0123456789abcdef", "md7", "/tmp/fixture/disk.img", "42", true}, func(string, ...string) ([]byte, error) {
				i := n
				n++
				if i == failure {
					return []byte(outputs[i]), errors.New("inspection unavailable")
				}
				return []byte(outputs[i]), nil
			})
			if err == nil || receipt != (dashboardCleanupReceipt{}) || n != failure+1 {
				t.Fatalf("failed command accepted or continued: %+v %v calls=%d", receipt, err, n)
			}
		})
	}
}

func TestDashboardStorageCleanupRejectsUncertainObservations(t *testing.T) {
	cases := []struct {
		at   int
		data string
	}{
		{0, "8\n"}, {0, "7 8\n"}, {0, "07\n"}, {0, "7\x00\n"}, {0, "7"},
		{1, "43\n"}, {1, "042\n"}, {1, "42\n42\n"}, {1, "42"},
		{2, "unexpected\n"}, {3, "boot\nhorvium_ui_0123456789abcdef\n"}, {3, "boot\nboot\n"}, {3, "boot\x00\n"}, {3, "boot"},
		{4, "8\n"}, {5, "unexpected\n"}, {6, "md0 md7\n"}, {6, "md0 md0\n"}, {6, "md00\n"}, {6, "md0 garbage\n"}, {6, "md0"},
		{3, strings.Repeat("x", 65537)}, {6, strings.Repeat("x", 65537)},
	}
	for i, c := range cases {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			outputs := []string{"7\n", "42\n", "", "boot\n", "7\n", "", "md0\n"}
			outputs[c.at] = c.data
			n := 0
			receipt, err := dashboardCleanupStorage(dashboardCleanupIdentity{"horvium_ui_0123456789abcdef", "md7", "/tmp/fixture/disk.img", "42", true}, func(string, ...string) ([]byte, error) { data := outputs[n]; n++; return []byte(data), nil })
			if err == nil || receipt != (dashboardCleanupReceipt{}) || n != c.at+1 {
				t.Fatalf("uncertain observation accepted or continued: %+v %v calls=%d", receipt, err, n)
			}
		})
	}
}

func TestDashboardStorageCleanupRejectsInvalidIdentity(t *testing.T) {
	valid := dashboardCleanupIdentity{"horvium_ui_0123456789abcdef", "md7", "/tmp/fixture/disk.img", "42", true}
	for _, change := range []func(*dashboardCleanupIdentity){
		func(x *dashboardCleanupIdentity) { x.Pool = "boot" }, func(x *dashboardCleanupIdentity) { x.Pool = "horvium_ui_0123456789abcde;" },
		func(x *dashboardCleanupIdentity) { x.Unit = "md07" }, func(x *dashboardCleanupIdentity) { x.Unit = "-7" },
		func(x *dashboardCleanupIdentity) { x.File = "relative/disk.img" }, func(x *dashboardCleanupIdentity) { x.File = "/tmp/a/../disk.img" }, func(x *dashboardCleanupIdentity) { x.File = "/tmp/a/other.img" },
		func(x *dashboardCleanupIdentity) { x.GUID = "0" }, func(x *dashboardCleanupIdentity) { x.GUID = "042" }, func(x *dashboardCleanupIdentity) { x.GUID = "18446744073709551616" }, func(x *dashboardCleanupIdentity) { x.GUID = "" },
	} {
		x := valid
		change(&x)
		called := false
		receipt, err := dashboardCleanupStorage(x, func(string, ...string) ([]byte, error) { called = true; return nil, nil })
		if err == nil || called || receipt != (dashboardCleanupReceipt{}) {
			t.Fatal("invalid identity reached native command")
		}
	}
}

func TestDashboardStorageCleanupOutputBound(t *testing.T) {
	var output dashboardCleanupOutput
	data := []byte(strings.Repeat("x", 65536))
	if n, err := output.Write(data); err != nil || n != 65536 || output.overflow {
		t.Fatal("bounded output rejected")
	}
	if n, err := output.Write([]byte("x")); err == nil || n != 0 || !output.overflow || len(output.data) != 65536 {
		t.Fatal("overflow retained or accepted")
	}
}
