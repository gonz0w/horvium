// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func FuzzPersistentBackingProjection(f *testing.F) {
	f.Add([]byte("/dev/ufsid/1234567823456789\n"))
	f.Add([]byte("tank/private\tguid\t456\ntank/private\ttype\tfilesystem\ntank/private\tmounted\tyes\ntank/private\tmountpoint\t/root/private\n"))
	f.Add([]byte("/dev/ufsid/12345678ffffffff\n"))
	f.Add([]byte(""))
	f.Fuzz(func(t *testing.T, data []byte) {
		ufs, e := parsePersistentUFS(data)
		if e == nil && (ufs[0] == 0 || ufs[1] == 0 || string(data) != fmt.Sprintf("/dev/ufsid/%08x%08x\n", ufs[0], ufs[1])) {
			t.Fatal("invalid UFS identity")
		}
		if e != nil && ufs != ([2]uint32{}) {
			t.Fatal("partial UFS identity")
		}
		zfs, e := parsePersistentZFS(data, "tank/private", "/root/private")
		if e == nil && zfs == 0 || e != nil && zfs != 0 {
			t.Fatal("invalid/partial ZFS identity")
		}
	})
}

func persistentFixture(kind string) (*persistentBackingCollector, *persistentBackingMount) {
	m := &persistentBackingMount{kind: kind, source: "/dev/md7", target: "/root/private", fsid: [2]int32{0x12345678, 0x23456789}}
	if kind == "zfs" {
		m.source = "tank/private"
	}
	s := &persistentBackingCollector{bytes: 64 << 10, commandTimeout: time.Second, totalTimeout: time.Second, observe: func() (persistentBackingMount, error) { return *m, nil }}
	s.read = func(ctx context.Context, probe, name string, limit int) (readOutput, error) {
		data := ""
		switch probe {
		case "versions":
			data = "15.1-RELEASE-p3\n15.1-RELEASE-p3\n15.1-RELEASE-p3\n"
		case "ufs-id":
			data = "/dev/ufsid/1234567823456789\n"
		case "zfs-id":
			data = "tank/private\tguid\t456\ntank/private\ttype\tfilesystem\ntank/private\tmounted\tyes\ntank/private\tmountpoint\t/root/private\n"
		case "pool-id":
			data = "123\n"
		}
		return readOutput{[]byte(data), len(data)}, nil
	}
	return s, m
}

func TestPersistentBackingCollection(t *testing.T) {
	for _, kind := range []string{"ufs", "zfs"} {
		t.Run(kind, func(t *testing.T) {
			s, _ := persistentFixture(kind)
			var calls []string
			read := s.read
			s.read = func(ctx context.Context, p, n string, limit int) (readOutput, error) {
				calls = append(calls, p+":"+n)
				return read(ctx, p, n, limit)
			}
			id, e := s.collect(context.Background())
			want := persistentBackingIdentity{kind: "ufs", ufs: [2]uint32{0x12345678, 0x23456789}}
			if kind == "zfs" {
				want = persistentBackingIdentity{kind: "zfs", pool: 123, dataset: 456}
			}
			if e != nil || id != want {
				t.Fatal("native fact collection", e)
			}
			count := 4
			if kind == "zfs" {
				count = 6
			}
			if len(calls) != count {
				t.Fatal("independent complete passes", calls)
			}
			if kind == "zfs" && (calls[1] != "zfs-id:tank/private" || calls[2] != "pool-id:tank") {
				t.Fatal("source selected projections", calls)
			}
		})
	}
}

func TestPersistentBackingDenials(t *testing.T) {
	for _, name := range []string{"missing-reader", "missing-observer", "invalid-budget", "nil-context", "cancelled", "unsafe-provider", "unsafe-dataset", "unsupported-fs", "invalid-target", "reader-error", "consumed-over", "consumed-under", "timeout", "over-limit", "missing-newline", "duplicate-ufs", "zero-ufs", "wrong-ufs", "uppercase-ufs", "unknown-version", "version-drift", "ufs-drift", "mount-drift", "observer-error", "zfs-duplicate", "zfs-foreign", "zfs-volume", "zfs-unmounted", "zfs-mountpoint", "zfs-noncanonical", "pool-zero", "pool-overflow", "pool-drift"} {
		t.Run(name, func(t *testing.T) {
			kind := "ufs"
			if strings.HasPrefix(name, "zfs-") || strings.HasPrefix(name, "pool-") {
				kind = "zfs"
			}
			s, m := persistentFixture(kind)
			read := s.read
			count := 0
			s.read = func(ctx context.Context, p, n string, l int) (readOutput, error) {
				count++
				out, e := read(ctx, p, n, l)
				data := string(out.data)
				switch name {
				case "reader-error":
					return readOutput{}, ErrUnavailable
				case "consumed-over":
					out.consumed = l + 1
					return out, nil
				case "consumed-under":
					out.consumed = 0
					return out, nil
				case "timeout":
					<-ctx.Done()
					return readOutput{}, ctx.Err()
				case "over-limit":
					data = strings.Repeat("x", l+1)
				case "missing-newline":
					if p == "ufs-id" {
						data = strings.TrimSuffix(data, "\n")
					}
				case "duplicate-ufs":
					if p == "ufs-id" {
						data += data
					}
				case "zero-ufs":
					if p == "ufs-id" {
						data = "/dev/ufsid/0000000023456789\n"
					}
				case "wrong-ufs":
					if p == "ufs-id" {
						data = "/dev/ufsid/1234567823456788\n"
					}
				case "uppercase-ufs":
					if p == "ufs-id" {
						data = "/dev/ufsid/123456782345678A\n"
					}
				case "unknown-version":
					if p == "versions" {
						data = strings.ReplaceAll(data, "15.1", "16.0")
					}
				case "version-drift":
					if p == "versions" && count > 2 {
						data = strings.ReplaceAll(data, "p3", "p4")
					}
				case "ufs-drift":
					if p == "ufs-id" && count > 2 {
						data = "/dev/ufsid/1234567823456788\n"
					}
				case "mount-drift":
					m.flags++
				case "zfs-duplicate":
					if p == "zfs-id" {
						data += data
					}
				case "zfs-foreign":
					if p == "zfs-id" {
						data = strings.ReplaceAll(data, "tank/private", "tank/other")
					}
				case "zfs-volume":
					if p == "zfs-id" {
						data = strings.ReplaceAll(data, "filesystem", "volume")
					}
				case "zfs-unmounted":
					if p == "zfs-id" {
						data = strings.ReplaceAll(data, "\tyes\n", "\tno\n")
					}
				case "zfs-mountpoint":
					if p == "zfs-id" {
						data = strings.ReplaceAll(data, "/root/private", "/root/other")
					}
				case "zfs-noncanonical":
					if p == "zfs-id" {
						data = strings.ReplaceAll(data, "\t456\n", "\t0456\n")
					}
				case "pool-zero":
					if p == "pool-id" {
						data = "0\n"
					}
				case "pool-overflow":
					if p == "pool-id" {
						data = "18446744073709551616\n"
					}
				case "pool-drift":
					if p == "pool-id" && count > 3 {
						data = "124\n"
					}
				}
				return readOutput{[]byte(data), len(data)}, e
			}
			ctx := context.Background()
			switch name {
			case "missing-reader":
				s.read = nil
			case "missing-observer":
				s.observe = nil
			case "invalid-budget":
				s.bytes = 0
			case "nil-context":
				ctx = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "unsafe-provider":
				m.source = "/dev/../etc/passwd"
			case "unsafe-dataset":
				m.source = "-tank/private"
			case "unsupported-fs":
				m.kind = "nfs"
			case "invalid-target":
				m.target = "relative"
			case "observer-error":
				s.observe = func() (persistentBackingMount, error) { return persistentBackingMount{}, ErrUnavailable }
			case "timeout":
				s.commandTimeout = time.Millisecond
			}
			id, e := s.collect(ctx)
			if e == nil || id != (persistentBackingIdentity{}) {
				t.Fatal("unsafe/partial facts escaped", e)
			}
		})
	}
	var s *persistentBackingCollector
	if _, e := s.collect(context.Background()); !errors.Is(e, ErrUnavailable) {
		t.Fatal(e)
	}
}
