// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

func collectionPolicy() storage.Policy {
	return storage.Policy{Revision: 1, PoolGUID: "100", PoolName: "tank", Scopes: []storage.Scope{{GUID: "101", Name: "tank", Properties: []storage.Property{storage.Quota}}}}
}

func TestPropertyPolicyCollectionEquivalentOrderingAndEmptyScope(t *testing.T) {
	calls := 0
	s := policySource(func(string) (storage.Policy, error) {
		calls++
		p := collectionPolicy()
		p.Scopes[0].Properties = []storage.Property{storage.Quota, storage.RefQuota}
		if calls%2 == 0 {
			p.Scopes[0].Properties = []storage.Property{storage.RefQuota, storage.Quota}
		}
		return p, nil
	})
	if _, _, err := s.Collect(context.Background(), "100"); err != nil {
		t.Fatal("canonical ordering drift", err)
	}
	s.ownership = func(string) (storage.Policy, error) { p := collectionPolicy(); p.Scopes = nil; return p, nil }
	if _, _, err := s.Collect(context.Background(), "100"); !errors.Is(err, storage.ErrDenied) {
		t.Fatal("empty enrollment", err)
	}
}

func FuzzPropertyPolicyCollection(f *testing.F) {
	f.Add([]byte("tank"), uint8(0))
	f.Add([]byte("foreign"), uint8(1))
	f.Fuzz(func(t *testing.T, data []byte, mode uint8) {
		if len(data) > 4096 {
			t.Skip()
		}
		calls := 0
		s := policySource(func(string) (storage.Policy, error) {
			calls++
			p := collectionPolicy()
			switch mode % 4 {
			case 0:
				p.Scopes[0].Name = string(data)
			case 1:
				if calls == 2 {
					p.Scopes[0].Name = string(data)
				}
			case 2:
				p.Scopes[0].Properties = []storage.Property{storage.Property(string(data))}
			case 3:
				p.PoolGUID = strings.TrimSpace(string(data))
			}
			return p, nil
		})
		p, o, err := s.Collect(context.Background(), "100")
		if err != nil {
			if !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) {
				t.Fatal("partial output")
			}
		} else if o.Eligible || storage.ValidatePropertyPolicyBinding(p, o) != nil {
			t.Fatal("invalid successful binding")
		}
	})
}

func policySource(read func(string) (storage.Policy, error)) *PropertyPolicySource {
	rows := propertyCaptures()
	return &PropertyPolicySource{ownership: read, pool: propertySource(func(_ context.Context, _p poolProbe, name string, _ int) (readOutput, error) {
		if name != "tank" {
			panic("policy address changed")
		}
		return testOutput(rows[_p]), nil
	})}
}

func TestPropertyPolicyCollectionBindsWithoutAuthority(t *testing.T) {
	calls := 0
	s := policySource(func(guid string) (storage.Policy, error) {
		if guid != "100" {
			t.Fatal("foreign selection")
		}
		calls++
		return collectionPolicy(), nil
	})
	p, o, err := s.Collect(context.Background(), "100")
	if err != nil || calls != 2 || p.Revision != 1 || !o.Complete || o.Eligible {
		t.Fatal("unbound or authoritative output", err)
	}
	if _, err := storage.Build(time.Now(), "session", p, o, storage.Intent{FilesystemGUID: "101", Property: storage.Quota, Value: "1"}); !errors.Is(err, storage.ErrUnavailable) {
		t.Fatal("granted eligibility", err)
	}
	p.Scopes[0].Name = "changed"
	o.Filesystems[0].Name = "changed"
	p, o, err = s.Collect(context.Background(), "100")
	if err != nil || p.Scopes[0].Name != "tank" || o.Filesystems[0].Name != "tank" {
		t.Fatal("aliased result", err)
	}
}

func TestPropertyPolicyCollectionDenialsExposeNothing(t *testing.T) {
	for _, failure := range []string{"first-read", "second-read", "revision", "scope", "missing", "parent", "foreign-pool", "invalid-policy", "native", "cancel"} {
		t.Run(failure, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			s := policySource(func(string) (storage.Policy, error) {
				calls++
				p := collectionPolicy()
				switch failure {
				case "first-read":
					return storage.Policy{}, storage.ErrStateDirectory
				case "second-read":
					if calls == 2 {
						return storage.Policy{}, storage.ErrStateRecord
					}
				case "revision":
					if calls == 2 {
						p.Revision++
					}
				case "scope":
					if calls == 2 {
						p.Scopes[0].Properties = []storage.Property{storage.RefQuota}
					}
				case "missing":
					p.Scopes[0].GUID = "999"
				case "parent":
					p.Scopes[0].Name = "tank/child"
					p.Scopes[0].ParentGUID = "101"
				case "foreign-pool":
					p.PoolGUID = "200"
				case "invalid-policy":
					p.Revision = 0
				case "cancel":
					if calls == 2 {
						cancel()
					}
				}
				return p, nil
			})
			if failure == "native" {
				s.pool.read = func(context.Context, poolProbe, string, int) (readOutput, error) { return readOutput{}, ErrExit }
			}
			p, o, err := s.Collect(ctx, "100")
			if err == nil || !reflect.DeepEqual(p, storage.Policy{}) || !reflect.DeepEqual(o, storage.Observation{}) {
				t.Fatal("partial denied output", err)
			}
		})
	}
}

func TestPropertyPolicyCollectionInvalidSelectionBeforeReads(t *testing.T) {
	s := policySource(func(string) (storage.Policy, error) {
		t.Fatal("invalid input read policy")
		return storage.Policy{}, nil
	})
	for _, guid := range []string{"", "0", "01", "-1", "18446744073709551616", "100/path"} {
		if _, _, err := s.Collect(context.Background(), guid); err == nil {
			t.Fatal("invalid GUID", guid)
		}
	}
	if _, _, err := s.Collect(nil, "100"); err == nil {
		t.Fatal("nil context")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := s.Collect(ctx, "100"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var absent *PropertyPolicySource
	if _, _, err := absent.Collect(context.Background(), "100"); err == nil {
		t.Fatal("nil source")
	}
}
