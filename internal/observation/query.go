// SPDX-License-Identifier: BSD-3-Clause
package observation

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/gonz0w/horvium/internal/inventory"
)

const defaultPageSize = 100
const maxPageSize = 500

func (m *Manager) query(q Query) (Query, error) {
	if q.Limit == 0 {
		q.Limit = defaultPageSize
	}
	if q.Sort == "" {
		q.Sort = "name"
	}
	if q.Limit < 1 || q.Limit > maxPageSize || q.Offset < 0 || len(q.Filter) > m.config.Limits.MaxFieldBytes || len(q.Generation) > 64 || !utf8.ValidString(q.Filter) {
		return q, ErrQuery
	}
	for _, c := range q.Filter {
		if c < 32 || c == 127 {
			return q, ErrQuery
		}
	}
	return q, nil
}
func (m *Manager) Datasets(ctx context.Context, q Query) (Page[inventory.Dataset], error) {
	q, err := m.query(q)
	if err != nil {
		return Page[inventory.Dataset]{}, err
	}
	compare := func(a, b inventory.Dataset) int {
		var n int
		switch q.Sort {
		case "name":
			n = cmp.Compare(a.Name, b.Name)
		case "guid":
			n = guidCompare(a.GUID, b.GUID)
		case "used":
			n = cmp.Compare(a.Used, b.Used)
		case "available":
			n = cmp.Compare(a.Available, b.Available)
		case "referenced":
			n = cmp.Compare(a.Referenced, b.Referenced)
		}
		if n == 0 {
			n = cmp.Compare(a.Name, b.Name)
		}
		if n == 0 {
			n = guidCompare(a.GUID, b.GUID)
		}
		return n
	}
	switch q.Sort {
	case "name", "guid", "used", "available", "referenced":
	default:
		return Page[inventory.Dataset]{}, ErrQuery
	}
	return page(ctx, m, q, 3, func(s inventory.Snapshot) []inventory.Dataset { return s.Datasets.Data }, func(v inventory.Dataset) string { return v.Name }, compare, func(v inventory.Dataset) inventory.Dataset { return v })
}
func (m *Manager) Pools(ctx context.Context, q Query) (Page[inventory.Pool], error) {
	q, err := m.query(q)
	if err != nil {
		return Page[inventory.Pool]{}, err
	}
	compare := func(a, b inventory.Pool) int {
		var n int
		switch q.Sort {
		case "name":
			n = cmp.Compare(a.Name, b.Name)
		case "guid":
			n = guidCompare(a.GUID, b.GUID)
		case "size":
			n = cmp.Compare(a.Size, b.Size)
		case "allocated":
			n = cmp.Compare(a.Allocated, b.Allocated)
		case "free":
			n = cmp.Compare(a.Free, b.Free)
		case "health":
			n = cmp.Compare(a.Health, b.Health)
		}
		if n == 0 {
			n = cmp.Compare(a.Name, b.Name)
		}
		if n == 0 {
			n = guidCompare(a.GUID, b.GUID)
		}
		return n
	}
	switch q.Sort {
	case "name", "guid", "size", "allocated", "free", "health":
	default:
		return Page[inventory.Pool]{}, ErrQuery
	}
	return page(ctx, m, q, 2, func(s inventory.Snapshot) []inventory.Pool { return s.Pools.Data }, func(v inventory.Pool) string { return v.Name }, compare, func(v inventory.Pool) inventory.Pool { return v })
}
func (m *Manager) Providers(ctx context.Context, q Query) (Page[inventory.Provider], error) {
	q, err := m.query(q)
	if err != nil {
		return Page[inventory.Provider]{}, err
	}
	compare := func(a, b inventory.Provider) int {
		var n int
		switch q.Sort {
		case "name":
			n = cmp.Compare(a.Name, b.Name)
		case "class":
			n = cmp.Compare(a.Class, b.Class)
		case "kind":
			n = cmp.Compare(a.Kind, b.Kind)
		case "size":
			n = cmp.Compare(a.MediaSize, b.MediaSize)
		case "identity":
			n = cmp.Compare(a.Identity, b.Identity)
		}
		if n == 0 {
			n = cmp.Compare(a.Name, b.Name)
		}
		if n == 0 {
			n = cmp.Compare(a.ID, b.ID)
		}
		return n
	}
	switch q.Sort {
	case "name", "class", "kind", "size", "identity":
	default:
		return Page[inventory.Provider]{}, ErrQuery
	}
	return page(ctx, m, q, 1, func(s inventory.Snapshot) []inventory.Provider { return s.GEOM.Data.Providers }, func(v inventory.Provider) string { return v.Name }, compare, func(v inventory.Provider) inventory.Provider {
		v.Identifiers = append([]inventory.Identifier(nil), v.Identifiers...)
		return v
	})
}
func guidCompare(a, b string) int {
	if n := cmp.Compare(len(a), len(b)); n != 0 {
		return n
	}
	return cmp.Compare(a, b)
}
func page[T any](ctx context.Context, m *Manager, q Query, section int, items func(inventory.Snapshot) []T, name func(T) string, compare func(T, T) int, detach func(T) T) (Page[T], error) {
	if err := ctx.Err(); err != nil {
		return Page[T]{}, err
	}
	select {
	case m.readers <- struct{}{}:
		defer func() { <-m.readers }()
	default:
		return Page[T]{}, ErrBusy
	}
	// Hold the read lock through bounded materialization. Evicted generations
	// cannot accumulate in slow readers; only detached <=500-row pages escape.
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, err := m.resolve(q.Generation)
	if err != nil {
		return Page[T]{}, err
	}
	rows := items(g.snapshot)
	indices := make([]int, 0, len(rows))
	for i, v := range rows {
		if i%256 == 0 {
			if err := ctx.Err(); err != nil {
				return Page[T]{}, err
			}
		}
		if strings.Contains(name(v), q.Filter) {
			indices = append(indices, i)
		}
	}
	slices.SortFunc(indices, func(a, b int) int {
		n := compare(rows[a], rows[b])
		if q.Descending {
			return -n
		}
		return n
	})
	if err := ctx.Err(); err != nil {
		return Page[T]{}, err
	}
	out := Page[T]{Generation: g.id, Total: len(indices), Offset: q.Offset, Freshness: m.freshness(g, section), Rows: make([]T, 0)}
	if q.Offset >= len(indices) {
		return out, nil
	}
	n := min(q.Limit, len(indices)-q.Offset)
	out.Rows = make([]T, n)
	for i := range n {
		out.Rows[i] = detach(rows[indices[q.Offset+i]])
	}
	out.More = n < len(indices)-q.Offset
	return out, nil
}
