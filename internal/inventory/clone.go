// SPDX-License-Identifier: BSD-3-Clause
package inventory

import (
	"reflect"
	"strings"
	"time"
)

// CloneBounded gives the cache exclusive ownership of arrays, pointers and string
// backing storage. MaxBytes accounts for model storage and string bytes, not an
// exact allocator/RSS bound. Check limits before validation maps and clone arrays.
func CloneBounded(s Snapshot, l Limits) (Snapshot, error) {
	if err := l.Validate(); err != nil {
		return Snapshot{}, err
	}
	remaining := l.MaxRecords
	for _, n := range []int{len(s.GEOM.Data.Classes), len(s.GEOM.Data.Geoms), len(s.GEOM.Data.Providers), len(s.Pools.Data), len(s.Datasets.Data)} {
		if n > remaining {
			return Snapshot{}, ErrLimit
		}
		remaining -= n
	}
	for _, g := range s.GEOM.Data.Geoms {
		if len(g.Consumers) > remaining {
			return Snapshot{}, ErrLimit
		}
		remaining -= len(g.Consumers)
	}
	bytes := l.MaxBytes
	v := reflect.ValueOf(s)
	if err := charge(&bytes, v.Type().Size()); err != nil {
		return Snapshot{}, err
	}
	if err := boundStorage(v, l, &bytes); err != nil {
		return Snapshot{}, err
	}
	if err := ValidateSnapshot(s); err != nil {
		return Snapshot{}, err
	}
	out := s
	out.StartedAt = out.StartedAt.UTC()
	out.FinishedAt = out.FinishedAt.UTC()
	out.Host = cloneSection(s.Host)
	out.GEOM = cloneSection(s.GEOM)
	out.Pools = cloneSection(s.Pools)
	out.Datasets = cloneSection(s.Datasets)
	h := s.Host.Data
	out.Host.Data = Host{InstalledKernel: strings.Clone(h.InstalledKernel), RunningKernel: strings.Clone(h.RunningKernel), Userland: strings.Clone(h.Userland), Architecture: strings.Clone(h.Architecture), ZFSUserland: strings.Clone(h.ZFSUserland), ZFSKernel: strings.Clone(h.ZFSKernel)}
	if h.PhysicalMemory != nil {
		n := *h.PhysicalMemory
		out.Host.Data.PhysicalMemory = &n
	}
	out.GEOM.Data.Classes = cloneSlice(s.GEOM.Data.Classes, func(v Class) Class { v.ID = strings.Clone(v.ID); v.Name = strings.Clone(v.Name); return v })
	out.GEOM.Data.Geoms = cloneSlice(s.GEOM.Data.Geoms, func(v Geom) Geom {
		v.ID = strings.Clone(v.ID)
		v.ClassID = strings.Clone(v.ClassID)
		v.Name = strings.Clone(v.Name)
		v.Consumers = cloneSlice(v.Consumers, func(c Consumer) Consumer {
			c.ID = strings.Clone(c.ID)
			c.ProviderID = strings.Clone(c.ProviderID)
			return c
		})
		return v
	})
	out.GEOM.Data.Providers = cloneSlice(s.GEOM.Data.Providers, cloneProvider)
	out.Pools.Data = cloneSlice(s.Pools.Data, func(v Pool) Pool {
		v.Name = strings.Clone(v.Name)
		v.GUID = strings.Clone(v.GUID)
		v.Health = strings.Clone(v.Health)
		return v
	})
	out.Datasets.Data = cloneSlice(s.Datasets.Data, func(v Dataset) Dataset { v.Name = strings.Clone(v.Name); v.GUID = strings.Clone(v.GUID); return v })
	return out, nil
}
func cloneProvider(v Provider) Provider {
	v.ID = strings.Clone(v.ID)
	v.GeomID = strings.Clone(v.GeomID)
	v.Name = strings.Clone(v.Name)
	v.Class = strings.Clone(v.Class)
	v.Kind = ProviderKind(strings.Clone(string(v.Kind)))
	v.Identity = IdentityState(strings.Clone(string(v.Identity)))
	v.Identifiers = cloneSlice(v.Identifiers, func(i Identifier) Identifier {
		i.Kind = strings.Clone(i.Kind)
		i.Value = strings.Clone(i.Value)
		i.Source = strings.Clone(i.Source)
		return i
	})
	return v
}
func cloneSection[T any](s Section[T]) Section[T] {
	s.StartedAt = s.StartedAt.UTC()
	s.FinishedAt = s.FinishedAt.UTC()
	s.Status = Status(strings.Clone(string(s.Status)))
	s.Problems = cloneSlice(s.Problems, func(p Problem) Problem { return Problem{strings.Clone(p.Source), strings.Clone(p.Code)} })
	return s
}
func cloneSlice[T any](items []T, clone func(T) T) []T {
	if items == nil {
		return nil
	}
	out := make([]T, len(items))
	for i, v := range items {
		out[i] = clone(v)
	}
	return out
}
func charge(remaining *int, n uintptr) error {
	if n > uintptr(*remaining) {
		return ErrLimit
	}
	*remaining -= int(n)
	return nil
}

var snapshotTimeType = reflect.TypeFor[time.Time]()

func boundStorage(v reflect.Value, l Limits, remaining *int) error {
	if v.Type() == snapshotTimeType {
		return nil
	}
	switch v.Kind() {
	case reflect.String:
		if err := safeTextAllowEmpty(v.String(), l); err != nil {
			return err
		}
		return charge(remaining, uintptr(v.Len()))
	case reflect.Pointer:
		if !v.IsNil() {
			if err := charge(remaining, v.Type().Elem().Size()); err != nil {
				return err
			}
			return boundStorage(v.Elem(), l, remaining)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if err := boundStorage(v.Field(i), l, remaining); err != nil {
				return err
			}
		}
	case reflect.Slice:
		size := v.Type().Elem().Size()
		if size > 0 && uintptr(v.Len()) > uintptr(*remaining)/size {
			return ErrLimit
		}
		if err := charge(remaining, uintptr(v.Len())*size); err != nil {
			return err
		}
		for i := 0; i < v.Len(); i++ {
			if err := boundStorage(v.Index(i), l, remaining); err != nil {
				return err
			}
		}
	}
	return nil
}
func safeTextAllowEmpty(s string, l Limits) error {
	if s == "" {
		return nil
	}
	return safeText(s, l)
}
