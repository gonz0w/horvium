// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gonz0w/horvium/internal/inventory"
)

const labelOutput = "------------------------------------\nLABEL 0 \n------------------------------------\n    version: 5000\n    name: 'fixture'\n    state: 1\n    txg: 8\n    pool_guid: 100\n    top_guid: 101\n    guid: 101\n    vdev_children: 1\n    vdev_tree:\n        type: 'disk'\n        id: 0\n        guid: 101\n        path: '/dev/ada1'\n        whole_disk: 0\n    features_for_read:\n    labels = 0 1 2 3 \n"

func labelFixture() PoolLabelCapture {
	return PoolLabelCapture{Device: "/dev/ada1", Output: []byte(labelOutput)}
}

func TestPoolLabelsRetainPositiveIdentityAndNativeState(t *testing.T) {
	for _, state := range []string{"0", "1", "2"} {
		c := labelFixture()
		c.Output = []byte(strings.Replace(labelOutput, "state: 1", "state: "+state, 1))
		got, err := ParsePoolLabels([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), c, inventory.DefaultLimits())
		if err != nil || got.Disk.Provider.Name != "ada1" || len(got.Configurations) != 1 {
			t.Fatal("positive label observation unavailable", err)
		}
		p := got.Configurations[0]
		if p.Name != "fixture" || p.PoolGUID != "100" || p.VdevGUID != "101" || p.TopGUID != "101" || p.State != state || p.TXG != "8" || !reflect.DeepEqual(p.Labels, []int{0, 1, 2, 3}) {
			t.Fatal("pool label identity/state lost")
		}
	}
}

func TestPoolLabelsRejectFailedPartialContradictoryAndUnsupportedCaptures(t *testing.T) {
	for _, output := range []string{"", "failed to unpack label 0\n", strings.TrimSuffix(labelOutput, "\n"),
		strings.Replace(labelOutput, "LABEL 0 ", "LABEL 0 (Bad label cksum)", 1),
		strings.Replace(labelOutput, "labels = 0 1 2 3 ", "labels = 0 1 2 ", 1),
		strings.Replace(labelOutput, "labels = 0 1 2 3 ", "labels = 0 1 1 3 ", 1),
		strings.Replace(labelOutput, "pool_guid: 100", "pool_guid: 0", 1),
		strings.Replace(labelOutput, "state: 1", "state: 4", 1),
		strings.Replace(labelOutput, "name: 'fixture'", "name: 'bad/name'", 1),
		strings.Replace(labelOutput, "txg: 8", "txg: 8\n    txg: 9", 1),
		strings.Replace(labelOutput, "pool_guid: 100\n", "", 1),
		strings.Replace(labelOutput, "version: 5000", "version: 99999", 1),
		strings.Replace(labelOutput, "vdev_tree:\n", "vdev_tree: 7\n", 1),
		strings.Replace(labelOutput, "        type:", "            type:", 1),
		labelOutput + "unexpected\n", labelOutput + labelOutput,
		strings.Replace(labelOutput, "guid: 101", "guid: 101\x00", 1),
	} {
		c := labelFixture()
		c.Output = []byte(output)
		got, err := ParsePoolLabels([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), c, inventory.DefaultLimits())
		if err == nil || !reflect.DeepEqual(got, PoolLabels{}) {
			t.Fatal("invalid label capture escaped", err)
		}
	}
	c := labelFixture()
	c.Device = "/dev/ada2"
	if _, err := ParsePoolLabels([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), c, inventory.DefaultLimits()); !errors.Is(err, ErrDrift) {
		t.Fatal("wrong probe address accepted", err)
	}
}

func TestPoolLabelsPreserveDistinctConfigurationsAndRequireCompleteCoverage(t *testing.T) {
	first := strings.Replace(labelOutput, "labels = 0 1 2 3 ", "labels = 0 1 ", 1)
	second := strings.Replace(strings.Replace(strings.Replace(labelOutput, "LABEL 0 ", "LABEL 2 ", 1), "labels = 0 1 2 3 ", "labels = 2 3 ", 1), "txg: 8", "txg: 9", 1)
	c := labelFixture()
	c.Output = []byte(first + second)
	got, err := ParsePoolLabels([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), c, inventory.DefaultLimits())
	if err != nil || len(got.Configurations) != 2 || got.Configurations[1].TXG != "9" {
		t.Fatal("distinct label configurations lost", err)
	}
	c.Output = []byte(first + strings.Replace(second, "pool_guid: 100", "pool_guid: 200", 1))
	if got, err := ParsePoolLabels([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), c, inventory.DefaultLimits()); err == nil || !reflect.DeepEqual(got, PoolLabels{}) {
		t.Fatal("conflicting pool identity accepted")
	}
}

func TestPoolLabelsBoundAggregateBytesFieldsAndRecords(t *testing.T) {
	c := labelFixture()
	l := inventory.DefaultLimits()
	l.MaxBytes = len(signatureGEOM) + len(signatureInfo) + len(signatureSerial) + len(c.Device) + len(c.Output) - 1
	if _, err := ParsePoolLabels([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), c, l); !errors.Is(err, ErrLimit) {
		t.Fatal("aggregate bytes ignored", err)
	}
	l = inventory.DefaultLimits()
	l.MaxRecords = 10
	if _, err := ParsePoolLabels([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), c, l); !errors.Is(err, ErrLimit) {
		t.Fatal("record cap ignored", err)
	}
	l = inventory.DefaultLimits()
	l.MaxFieldBytes = 10
	if _, err := ParsePoolLabels([]byte(signatureGEOM), []byte(signatureInfo), []byte(signatureSerial), c, l); !errors.Is(err, ErrLimit) {
		t.Fatal("field cap ignored", err)
	}
}

func TestDeviceUseRetainsLabelIdentityAndForcesReviewOnExport(t *testing.T) {
	c := deviceUseFixture()
	c.PoolLabels = &PoolLabelCapture{Device: "/dev/ada1", Output: []byte(strings.Replace(labelOutput, "state: 1", "state: 0", 1))}
	r, err := ReviewDeviceUse(c, inventory.DefaultLimits())
	if err != nil || !r.Observation().PoolLabelsAvailable || len(r.Observation().PoolLabels) != 1 {
		t.Fatal("label observation did not join", err)
	}
	copy := r.Observation()
	copy.PoolLabels[0].Labels[0] = 3
	c.PoolLabels.Output[0] = 'x'
	if r.Observation().PoolLabels[0].Labels[0] != 0 {
		t.Fatal("caller modified baseline")
	}
	fresh := deviceUseFixture()
	fresh.PoolLabels = new(PoolLabelCapture)
	*fresh.PoolLabels = labelFixture()
	if err := r.Validate(fresh, inventory.DefaultLimits()); !errors.Is(err, ErrDrift) {
		t.Fatal("exported state did not invalidate active labels", err)
	}
	fresh.PoolLabels = nil
	if err := r.Validate(fresh, inventory.DefaultLimits()); !errors.Is(err, ErrDrift) {
		t.Fatal("unavailable label evidence did not invalidate baseline", err)
	}
}

func FuzzPoolLabels(f *testing.F) {
	f.Add(signatureGEOM, signatureInfo, signatureSerial, "/dev/ada1", labelOutput)
	f.Fuzz(func(t *testing.T, geom, info, serial, device, output string) {
		l := inventory.DefaultLimits()
		l.MaxBytes = 64 << 10
		l.MaxRecords = 1024
		l.MaxFieldBytes = 1024
		got, err := ParsePoolLabels([]byte(geom), []byte(info), []byte(serial), PoolLabelCapture{Device: device, Output: []byte(output)}, l)
		if err != nil {
			if !reflect.DeepEqual(got, PoolLabels{}) {
				t.Fatal("partial label result")
			}
			return
		}
		if got.Disk.Provider.Identity != inventory.IdentityUnique || len(got.Configurations) == 0 {
			t.Fatal("unbound labels")
		}
	})
}
