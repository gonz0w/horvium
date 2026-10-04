// SPDX-License-Identifier: BSD-3-Clause
// Package helper implements the private, inventory-only management transport.
package helper

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gonz0w/horvium/internal/inventory"
)

var (
	ErrProtocol    = errors.New("invalid helper protocol")
	ErrLimit       = errors.New("helper safety budget exceeded")
	ErrPermission  = errors.New("helper peer denied")
	ErrConfig      = errors.New("unsafe helper configuration")
	ErrUnsupported = errors.New("helper transport requires FreeBSD")
)

const maxFrame = 64 << 10

// Limits are supplied by the trusted caller, never by a request. MaxBytes counts
// JSON payload bytes; MaxRecords includes begin/end and every graph/ZFS item.
type Limits struct{ MaxBytes, MaxRecords, MaxFieldBytes, MaxDepth int }

func DefaultLimits() Limits { return Limits{64 << 20, 100000, 4096, 32} }
func (l Limits) validate() error {
	if l.MaxBytes < 1 || l.MaxRecords < 2 || l.MaxFieldBytes < 1 || l.MaxDepth < 1 {
		return ErrConfig
	}
	return nil
}

type request struct {
	Version   int    `json:"version"`
	ID        string `json:"request_id"`
	Operation string `json:"operation"`
}

type record struct {
	Version int             `json:"version"`
	ID      string          `json:"request_id"`
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
}

type sectionMeta struct {
	Status     inventory.Status    `json:"status"`
	StartedAt  time.Time           `json:"started_at"`
	FinishedAt time.Time           `json:"finished_at"`
	Problems   []inventory.Problem `json:"problems,omitempty"`
}

func metadata[T any](s inventory.Section[T]) sectionMeta {
	return sectionMeta{s.Status, s.StartedAt, s.FinishedAt, s.Problems}
}
func section[T any](m sectionMeta) inventory.Section[T] {
	return inventory.Section[T]{Status: m.Status, StartedAt: m.StartedAt, FinishedAt: m.FinishedAt, Problems: m.Problems}
}

type begin struct {
	StartedAt  time.Time                         `json:"started_at"`
	FinishedAt time.Time                         `json:"finished_at"`
	Host       inventory.Section[inventory.Host] `json:"host"`
	GEOM       sectionMeta                       `json:"geom"`
	Pools      sectionMeta                       `json:"pools"`
	Datasets   sectionMeta                       `json:"datasets"`
}

// Consumers are independent records so a large geom cannot evade frame bounds.
type geomItem struct {
	ID      string `json:"local_id"`
	ClassID string `json:"class_local_id"`
	Name    string `json:"name"`
}
type consumerItem struct {
	GeomID   string             `json:"geom_local_id"`
	Consumer inventory.Consumer `json:"consumer"`
}
type end struct {
	Records  int  `json:"records"`
	Complete bool `json:"complete"`
}

type budget struct {
	limits         Limits
	bytes, records int
}

func newBudget(l Limits) (*budget, error) {
	if err := l.validate(); err != nil {
		return nil, err
	}
	return &budget{limits: l}, nil
}
func (b *budget) debit(n int) error {
	if n < 1 {
		return ErrProtocol
	}
	if n > maxFrame || n > b.limits.MaxBytes-b.bytes || b.records >= b.limits.MaxRecords {
		return ErrLimit
	}
	b.bytes += n
	b.records++
	return nil
}
func (b *budget) read(r io.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, ErrProtocol
	}
	n := uint64(binary.BigEndian.Uint32(header[:]))
	// The uint64 check precedes int conversion (including on 32-bit targets).
	if n > maxFrame {
		return nil, ErrLimit
	}
	if err := b.debit(int(n)); err != nil {
		return nil, err
	}
	data := make([]byte, int(n))
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, ErrProtocol
	}
	return data, nil
}
func (b *budget) write(w io.Writer, v any) error {
	// Bound the object before json.Marshal can allocate an oversized record.
	if _, err := valueBound(reflect.ValueOf(v), b.limits, 0, maxFrame); err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return ErrProtocol
	}
	if err := checkJSON(data, nil, b.limits); err != nil {
		return err
	}
	if err := b.debit(len(data)); err != nil {
		return err
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(data)))
	if err := writeAll(w, header[:]); err != nil {
		return err
	}
	return writeAll(w, data)
}
func writeAll(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		if n <= 0 || n > len(b) {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}
func validID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
func writeRequest(w io.Writer, id string, l Limits) error {
	if !validID(id) {
		return ErrProtocol
	}
	b, err := newBudget(l)
	if err != nil {
		return err
	}
	return b.write(w, request{1, id, "inventory"})
}
func readRequest(r io.Reader, l Limits) (string, error) {
	b, err := newBudget(l)
	if err != nil {
		return "", err
	}
	data, err := b.read(r)
	if err != nil {
		return "", err
	}
	var req request
	if err := decode(data, &req, l); err != nil {
		return "", err
	}
	if req.Version != 1 || req.Operation != "inventory" || !validID(req.ID) {
		return "", ErrProtocol
	}
	return req.ID, nil
}
func writeSnapshot(w io.Writer, id string, s inventory.Snapshot, l Limits) error {
	if !validID(id) {
		return ErrProtocol
	}
	if err := l.validate(); err != nil {
		return err
	}
	// Bound validation maps before populating them, as well as frames before
	// emission. The record cap includes begin/end and every consumer.
	remaining := l.MaxRecords - 2
	for _, n := range []int{len(s.GEOM.Data.Classes), len(s.GEOM.Data.Geoms), len(s.GEOM.Data.Providers), len(s.Pools.Data), len(s.Datasets.Data)} {
		if n > remaining {
			return ErrLimit
		}
		remaining -= n
	}
	for _, g := range s.GEOM.Data.Geoms {
		if len(g.Consumers) > remaining {
			return ErrLimit
		}
		remaining -= len(g.Consumers)
	}
	if err := validateSnapshot(s); err != nil {
		return err
	}
	b, err := newBudget(l)
	if err != nil {
		return err
	}
	emit := func(kind string, v any) error {
		if _, err := valueBound(reflect.ValueOf(v), l, 0, maxFrame); err != nil {
			return err
		}
		data, err := json.Marshal(v)
		if err != nil {
			return ErrProtocol
		}
		return b.write(w, record{1, id, kind, data})
	}
	if err := emit("begin", begin{s.StartedAt, s.FinishedAt, s.Host, metadata(s.GEOM), metadata(s.Pools), metadata(s.Datasets)}); err != nil {
		return err
	}
	for _, v := range s.GEOM.Data.Classes {
		if err := emit("class", v); err != nil {
			return err
		}
	}
	for _, v := range s.GEOM.Data.Geoms {
		if err := emit("geom", geomItem{v.ID, v.ClassID, v.Name}); err != nil {
			return err
		}
		for _, c := range v.Consumers {
			if err := emit("consumer", consumerItem{v.ID, c}); err != nil {
				return err
			}
		}
	}
	for _, v := range s.GEOM.Data.Providers {
		if err := emit("provider", v); err != nil {
			return err
		}
	}
	for _, v := range s.Pools.Data {
		if err := emit("pool", v); err != nil {
			return err
		}
	}
	for _, v := range s.Datasets.Data {
		if err := emit("dataset", v); err != nil {
			return err
		}
	}
	return emit("end", end{b.records + 1, s.Complete()})
}
func readSnapshot(r io.Reader, id string, l Limits) (inventory.Snapshot, error) {
	s, err := receiveSnapshot(r, id, l)
	if err != nil {
		return inventory.Snapshot{}, err
	}
	return s, nil
}
func receiveSnapshot(r io.Reader, id string, l Limits) (inventory.Snapshot, error) {
	if !validID(id) {
		return inventory.Snapshot{}, ErrProtocol
	}
	b, err := newBudget(l)
	if err != nil {
		return inventory.Snapshot{}, err
	}
	var s inventory.Snapshot
	geoms := map[string]int{}
	for {
		data, err := b.read(r)
		if err != nil {
			return s, err
		}
		var rec record
		if err := decode(data, &rec, l); err != nil {
			return s, err
		}
		if rec.Version != 1 || rec.ID != id || (b.records == 1) != (rec.Type == "begin") {
			return s, ErrProtocol
		}
		switch rec.Type {
		case "begin":
			var v begin
			if err := decode(rec.Data, &v, l); err != nil {
				return s, err
			}
			s = inventory.Snapshot{StartedAt: v.StartedAt, FinishedAt: v.FinishedAt, Host: v.Host, GEOM: section[inventory.GEOM](v.GEOM), Pools: section[[]inventory.Pool](v.Pools), Datasets: section[[]inventory.Dataset](v.Datasets)}
		case "class":
			var v inventory.Class
			if err := decode(rec.Data, &v, l); err != nil {
				return s, err
			}
			s.GEOM.Data.Classes = append(s.GEOM.Data.Classes, v)
		case "geom":
			var v geomItem
			if err := decode(rec.Data, &v, l); err != nil {
				return s, err
			}
			if _, ok := geoms[v.ID]; ok {
				return s, ErrProtocol
			}
			geoms[v.ID] = len(s.GEOM.Data.Geoms)
			s.GEOM.Data.Geoms = append(s.GEOM.Data.Geoms, inventory.Geom{ID: v.ID, ClassID: v.ClassID, Name: v.Name})
		case "consumer":
			var v consumerItem
			if err := decode(rec.Data, &v, l); err != nil {
				return s, err
			}
			i, ok := geoms[v.GeomID]
			if !ok {
				return s, ErrProtocol
			}
			s.GEOM.Data.Geoms[i].Consumers = append(s.GEOM.Data.Geoms[i].Consumers, v.Consumer)
		case "provider":
			var v inventory.Provider
			if err := decode(rec.Data, &v, l); err != nil {
				return s, err
			}
			s.GEOM.Data.Providers = append(s.GEOM.Data.Providers, v)
		case "pool":
			var v inventory.Pool
			if err := decode(rec.Data, &v, l); err != nil {
				return s, err
			}
			s.Pools.Data = append(s.Pools.Data, v)
		case "dataset":
			var v inventory.Dataset
			if err := decode(rec.Data, &v, l); err != nil {
				return s, err
			}
			s.Datasets.Data = append(s.Datasets.Data, v)
		case "end":
			var v end
			if err := decode(rec.Data, &v, l); err != nil {
				return s, err
			}
			if v.Records != b.records || v.Complete != s.Complete() {
				return s, ErrProtocol
			}
			if err := validateSnapshot(s); err != nil {
				return s, err
			}
			var extra [1]byte
			if n, err := r.Read(extra[:]); n != 0 || err != io.EOF {
				return s, ErrProtocol
			}
			return s, nil
		default:
			return s, ErrProtocol
		}
	}
}

var timeType = reflect.TypeFor[time.Time]()
var rawType = reflect.TypeFor[json.RawMessage]()

func decode(data []byte, dst any, l Limits) error {
	if err := checkJSON(data, reflect.TypeOf(dst).Elem(), l); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return ErrProtocol
	}
	return nil
}

// Token validation rejects duplicate/case-variant/unknown keys and missing fields
// at every typed level. It also bounds generic RawMessage before typed dispatch.
func checkJSON(data []byte, t reflect.Type, l Limits) error {
	if !utf8.Valid(data) {
		return ErrProtocol
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := jsonValue(d, t, l, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrProtocol
	}
	return nil
}
func jsonValue(d *json.Decoder, t reflect.Type, l Limits, depth int) error {
	if depth > l.MaxDepth {
		return ErrLimit
	}
	if t == rawType {
		t = nil
	}
	pointer := t != nil && t.Kind() == reflect.Pointer
	if pointer {
		t = t.Elem()
	}
	v, err := d.Token()
	if err != nil {
		return ErrProtocol
	}
	if v == nil {
		if t == nil || pointer || t.Kind() == reflect.Slice {
			return nil
		}
		return ErrProtocol
	}
	if str, ok := v.(string); ok {
		if len(str) > l.MaxFieldBytes {
			return ErrLimit
		}
		for _, c := range str {
			if c < 32 || c == 127 {
				return ErrProtocol
			}
		}
		if t != nil && t != timeType && t.Kind() != reflect.String {
			return ErrProtocol
		}
		return nil
	}
	if delim, ok := v.(json.Delim); ok {
		switch delim {
		case '{':
			if t != nil && (t.Kind() != reflect.Struct || t == timeType) {
				return ErrProtocol
			}
			fields := map[string]reflect.StructField{}
			if t != nil {
				for i := 0; i < t.NumField(); i++ {
					f := t.Field(i)
					name := strings.Split(f.Tag.Get("json"), ",")[0]
					if name != "-" {
						fields[name] = f
					}
				}
			}
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return ErrProtocol
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return ErrProtocol
				}
				if len(name) > l.MaxFieldBytes {
					return ErrLimit
				}
				seen[name] = true
				var child reflect.Type
				if t != nil {
					f, ok := fields[name]
					if !ok {
						return ErrProtocol
					}
					child = f.Type
				}
				if err := jsonValue(d, child, l, depth+1); err != nil {
					return err
				}
			}
			if t != nil {
				for name, f := range fields {
					if !seen[name] && !strings.Contains(f.Tag.Get("json"), ",omitempty") {
						return ErrProtocol
					}
				}
			}
			close, err := d.Token()
			if err != nil || close != json.Delim('}') {
				return ErrProtocol
			}
			return nil
		case '[':
			if t != nil && t.Kind() != reflect.Slice {
				return ErrProtocol
			}
			var child reflect.Type
			if t != nil {
				child = t.Elem()
			}
			for d.More() {
				if err := jsonValue(d, child, l, depth+1); err != nil {
					return err
				}
			}
			close, err := d.Token()
			if err != nil || close != json.Delim(']') {
				return ErrProtocol
			}
			return nil
		default:
			return ErrProtocol
		}
	}
	if t != nil {
		switch t.Kind() {
		case reflect.Bool:
			if _, ok := v.(bool); !ok {
				return ErrProtocol
			}
		case reflect.Int, reflect.Uint64:
			if _, ok := v.(json.Number); !ok {
				return ErrProtocol
			}
		default:
			return ErrProtocol
		}
	}
	return nil
}

// Each string is marshalled separately (bounded by MaxFieldBytes), so the size
// preflight never creates an unbounded whole-record buffer. RawMessage is already
// bounded by emit and is accounted as bytes rather than a []uint8 JSON array.
func valueBound(v reflect.Value, l Limits, depth, remaining int) (int, error) {
	if depth > l.MaxDepth {
		return 0, ErrLimit
	}
	if !v.IsValid() {
		return 4, nil
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 4, nil
		}
		return valueBound(v.Elem(), l, depth+1, remaining)
	}
	if v.Type() == rawType {
		n := v.Len()
		if n > remaining {
			return 0, ErrLimit
		}
		return n, nil
	}
	if v.Type() == timeType {
		if remaining < 64 {
			return 0, ErrLimit
		}
		return 64, nil
	}
	size := 0
	add := func(n int) error {
		if n > remaining-size {
			return ErrLimit
		}
		size += n
		return nil
	}
	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if len(s) > l.MaxFieldBytes {
			return 0, ErrLimit
		}
		if !utf8.ValidString(s) {
			return 0, ErrProtocol
		}
		encoded, _ := json.Marshal(s)
		if err := add(len(encoded)); err != nil {
			return 0, err
		}
	case reflect.Struct:
		if err := add(2); err != nil {
			return 0, err
		}
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")
			if tag[0] == "-" {
				continue
			}
			value := v.Field(i)
			if len(tag) > 1 && tag[1] == "omitempty" && (value.IsZero() || (value.Kind() == reflect.Slice && value.Len() == 0)) {
				continue
			}
			if err := add(len(tag[0]) + 4); err != nil {
				return 0, err
			}
			n, err := valueBound(value, l, depth+1, remaining-size)
			if err != nil {
				return 0, err
			}
			if err := add(n); err != nil {
				return 0, err
			}
		}
	case reflect.Slice:
		if v.IsNil() {
			return 4, nil
		}
		if err := add(2); err != nil {
			return 0, err
		}
		for i := 0; i < v.Len(); i++ {
			if err := add(1); err != nil {
				return 0, err
			}
			n, err := valueBound(v.Index(i), l, depth+1, remaining-size)
			if err != nil {
				return 0, err
			}
			if err := add(n); err != nil {
				return 0, err
			}
		}
	case reflect.Int, reflect.Uint64:
		if err := add(20); err != nil {
			return 0, err
		}
	case reflect.Bool:
		if err := add(5); err != nil {
			return 0, err
		}
	default:
		return 0, ErrProtocol
	}
	return size, nil
}
