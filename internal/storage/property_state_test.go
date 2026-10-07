// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"github.com/gonz0w/horvium/internal/inventory"
	"reflect"
	"strings"
	"testing"
)

func TestPropertyStoreBackingRequiresCompleteIndependentSystemMount(t *testing.T) {
	u := propertySystemFixture(t, "/dev/gpt/rootfs / ufs rw 1 1\ntank/data /srv/data zfs rw 0 0\n", emptySwap)
	base := propertyStateBacking{kind: "ufs", source: "/dev/gpt/rootfs", target: "/"}
	owners := map[string]bool{"100": true}
	if err := checkPropertyStateBacking(base, u, owners); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"readonly", "softupdates", "unsupported", "missing", "source", "target", "data", "owned", "overlap", "zero"} {
		t.Run(mode, func(t *testing.T) {
			b, v := base, u
			own := map[string]bool{"100": true}
			switch mode {
			case "readonly":
				b.readonly = true
			case "softupdates":
				b.softupdates = true
			case "unsupported":
				b.kind = "tmpfs"
			case "missing":
				v.system.Mounts = nil
			case "source":
				b.source = "/dev/other"
			case "target":
				b.target = "/var"
			case "data":
				b = propertyStateBacking{kind: "zfs", source: "tank/data", target: "/srv/data"}
			case "owned":
				v.system.Mounts = append([]MountUse(nil), u.system.Mounts...)
				v.system.Mounts[0].PoolGUID = "999"
				own["999"] = true
			case "overlap":
				v.system.Mounts = append([]MountUse(nil), u.system.Mounts...)
				v.system.Mounts[0].Disks = append([]inventory.Provider(nil), u.system.Mounts[0].Disks...)
				v.system.Mounts[0].Disks[0].ID = u.pools[0].Disks[0].ID
			case "zero":
				v = PoolSystemUse{}
			}
			if err := checkPropertyStateBacking(b, v, own); err == nil {
				t.Fatal("unsafe backing accepted")
			}
		})
	}
}
func TestPropertyStoreRetainsAllSelectedPoolHoldsAndGlobalPendingWork(t *testing.T) {
	empty, err := BuildJobReservations(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkPropertyStateJobs(empty, "100"); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"queued", "applying", "observing", "succeeded", "failed", "cancelled", "reconcile-required", "other-pending", "other-terminal"} {
		t.Run(mode, func(t *testing.T) {
			j := jobFixture()
			var err error
			switch mode {
			case "applying":
				j = restartJobFixture(t, j, JobApplying, "", "")
			case "observing":
				j = restartJobFixture(t, j, JobObserving, "", "")
			case "succeeded":
				j = restartJobFixture(t, j, JobSucceeded, "", "verified")
			case "failed", "other-terminal":
				j = restartJobFixture(t, j, JobFailed, "", "verified-failure")
			case "cancelled":
				j = restartJobFixture(t, j, JobCancelled, "", "")
			case "reconcile-required":
				j = restartJobFixture(t, j, JobReconcileRequired, JobQueued, "review-expired")
			}
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(mode, "other-") {
				j.Intent.PoolGUID = "999"
			}
			data, err := EncodeJobRecord(j)
			if err != nil {
				t.Fatal(err)
			}
			v, err := BuildJobReservations([][]byte{data})
			if err != nil {
				t.Fatal(err)
			}
			before := v.Pools()
			err = checkPropertyStateJobs(v, "100")
			if mode == "other-terminal" {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, ErrDenied) {
				t.Fatal("retained hold ignored", err)
			}
			if !reflect.DeepEqual(before, v.Pools()) {
				t.Fatal("hold released")
			}
		})
	}
	if err := checkPropertyStateJobs(JobReservations{}, "100"); err == nil {
		t.Fatal("uninitialized view")
	}
}
