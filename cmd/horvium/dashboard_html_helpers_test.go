// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"cmp"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

// This is a bounded controller parser for the current rendered template grammar,
// not a browser HTML implementation. RawToken permits HTML boolean attributes;
// explicit stack checks reject repairs, mismatched closes and truncated markup.
var dashboardGenerationPattern = regexp.MustCompile(`^[0-9a-f]{32}-[1-9][0-9]{0,19}$`)

type dashboardHTMLContent struct {
	text string
	node *dashboardHTMLNode
}
type dashboardHTMLNode struct {
	name     string
	attrs    map[string]string
	children []*dashboardHTMLNode
	content  []dashboardHTMLContent
}

func (n *dashboardHTMLNode) rawText() string {
	var b strings.Builder
	var write func(*dashboardHTMLNode)
	write = func(v *dashboardHTMLNode) {
		for _, p := range v.content {
			if p.node != nil {
				write(p.node)
			} else {
				b.WriteString(p.text)
			}
		}
	}
	write(n)
	return b.String()
}
func (n *dashboardHTMLNode) text() string { return strings.Join(strings.Fields(n.rawText()), " ") }
func (n *dashboardHTMLNode) find(name, key, value string) []*dashboardHTMLNode {
	var out []*dashboardHTMLNode
	var visit func(*dashboardHTMLNode)
	visit = func(v *dashboardHTMLNode) {
		if v.name == name && (key == "" || v.attrs[key] == value) {
			out = append(out, v)
		}
		for _, c := range v.children {
			visit(c)
		}
	}
	visit(n)
	return out
}
func (n *dashboardHTMLNode) direct(name string) []*dashboardHTMLNode {
	var out []*dashboardHTMLNode
	for _, c := range n.children {
		if c.name == name {
			out = append(out, c)
		}
	}
	return out
}
func (n *dashboardHTMLNode) one(name, key, value string) (*dashboardHTMLNode, error) {
	nodes := n.find(name, key, value)
	if len(nodes) != 1 {
		return nil, errors.New("required unique response element missing")
	}
	return nodes[0], nil
}
func dashboardParseHTML(body []byte) (*dashboardHTMLNode, error) {
	invalid := func() (*dashboardHTMLNode, error) {
		return nil, errors.New("malformed or unbounded dashboard document")
	}
	if len(body) > dashboardBodyLimit || !bytes.HasPrefix(bytes.TrimSpace(body), []byte("<!doctype html>")) || !bytes.HasSuffix(bytes.TrimSpace(body), []byte("</html>")) {
		return invalid()
	}
	d := xml.NewDecoder(bytes.NewReader(body))
	d.Strict = false
	d.Entity = xml.HTMLEntity
	void := map[string]bool{"meta": true, "link": true, "input": true, "br": true, "hr": true, "img": true, "area": true, "base": true, "col": true, "embed": true, "param": true, "source": true, "track": true, "wbr": true}
	var root *dashboardHTMLNode
	var stack []*dashboardHTMLNode
	ids := map[string]bool{}
	nodes := 0
	doctype := 0
	for {
		token, err := d.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return invalid()
		}
		switch token := token.(type) {
		case xml.Directive:
			if string(token) != "doctype html" || root != nil || doctype != 0 {
				return invalid()
			}
			doctype++
		case xml.ProcInst:
			return invalid()
		case xml.StartElement:
			if token.Name.Space != "" {
				return invalid()
			}
			nodes++
			if nodes > 20000 || len(stack) >= 64 {
				return invalid()
			}
			n := &dashboardHTMLNode{name: token.Name.Local, attrs: make(map[string]string)}
			for _, a := range token.Attr {
				if _, exists := n.attrs[a.Name.Local]; exists || len(a.Value) > 8192 || a.Name.Space != "" {
					return invalid()
				}
				n.attrs[a.Name.Local] = a.Value
			}
			if id := n.attrs["id"]; id != "" {
				if ids[id] {
					return invalid()
				}
				ids[id] = true
			}
			if len(stack) == 0 {
				if root != nil || n.name != "html" || n.attrs["lang"] != "en" {
					return invalid()
				}
				root = n
			} else {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, n)
				parent.content = append(parent.content, dashboardHTMLContent{node: n})
			}
			if !void[n.name] {
				stack = append(stack, n)
			}
		case xml.EndElement:
			if token.Name.Space != "" {
				return invalid()
			}
			if void[token.Name.Local] {
				continue
			}
			if len(stack) == 0 || stack[len(stack)-1].name != token.Name.Local {
				return invalid()
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 {
				if strings.TrimSpace(string(token)) != "" {
					return invalid()
				}
			} else {
				n := stack[len(stack)-1]
				n.content = append(n.content, dashboardHTMLContent{text: string(token)})
			}
		}
	}
	if root == nil || len(stack) != 0 || doctype != 1 {
		return invalid()
	}
	return root, nil
}

type dashboardHTMLRow struct {
	name, guid string
	cells      []string
	measures   []dashboardHTMLMeasure
	zero       bool
}
type dashboardHTMLMeasure struct{ class, width, percent string }
type dashboardHTMLPage struct {
	title, path, caption string
	status               inventory.Status
	offset, limit, total int
	rows                 []dashboardHTMLRow
	maximum              uint64
	overview             bool
}

func dashboardExpectedRatio(value, size uint64, class string) dashboardHTMLMeasure {
	// Independent arbitrary-precision reference; never use product ratio helpers.
	if size == 0 || value > size {
		return dashboardHTMLMeasure{class, "0", "Unavailable"}
	}
	scaled := new(big.Int).Mul(new(big.Int).SetUint64(value), big.NewInt(1000))
	denominator := new(big.Int).SetUint64(size)
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(scaled, denominator, remainder)
	if remainder.Lsh(remainder, 1).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	width := quotient.Uint64()
	return dashboardHTMLMeasure{class, strconv.FormatUint(width, 10), fmt.Sprintf("%d.%d%%", width/10, width%10)}
}
func dashboardBytes(v uint64) string { return strconv.FormatUint(v, 10) + " B" }
func dashboardGUIDCompare(a, b string) int {
	aN, _ := strconv.ParseUint(a, 10, 64)
	bN, _ := strconv.ParseUint(b, 10, 64)
	return cmp.Compare(aN, bN)
}

func newDashboardHTMLPredicate(original inventory.Snapshot, plan []dashboardRequest) (dashboardPredicate, error) {
	s, err := inventory.CloneBounded(original, inventory.DefaultLimits())
	if err != nil {
		return nil, errors.New("invalid declared dashboard inventory")
	}
	prepared, err := dashboardPlan(max(1, len(s.Datasets.Data)))
	if err != nil || !slices.Equal(prepared, plan) {
		return nil, errors.New("invalid declared dashboard plan")
	}
	slices.SortFunc(s.Pools.Data, func(a, b inventory.Pool) int {
		if n := cmp.Compare(a.Name, b.Name); n != 0 {
			return n
		}
		return dashboardGUIDCompare(a.GUID, b.GUID)
	})
	slices.SortFunc(s.Datasets.Data, func(a, b inventory.Dataset) int {
		if n := cmp.Compare(a.Name, b.Name); n != 0 {
			return n
		}
		return dashboardGUIDCompare(a.GUID, b.GUID)
	})
	pages := make(map[string]dashboardHTMLPage)
	for _, request := range prepared {
		if _, ok := pages[request.Path]; ok {
			continue
		}
		u, _ := url.Parse(request.Path)
		q := u.Query()
		offset, _ := strconv.Atoi(q.Get("offset"))
		limit := 100
		if q.Get("limit") == "500" {
			limit = 500
		}
		p := dashboardHTMLPage{title: "Pools", path: "/storage/pools", caption: "Pools in this observation", offset: offset, limit: limit, total: len(s.Pools.Data), status: s.Pools.Status, overview: request.Case == "overview"}
		if p.overview {
			p.title = "Overview"
			p.caption = fmt.Sprintf("Pool capacity in this observation · first %d pools by name; exact byte values", min(20, len(s.Pools.Data)))
			limit = 20
		}
		if strings.HasPrefix(request.Case, "datasets-") {
			p.title = "Datasets"
			p.path = "/storage/datasets"
			p.caption = "Datasets in this observation"
			p.total = len(s.Datasets.Data)
			p.status = s.Datasets.Status
			for _, d := range s.Datasets.Data[offset:min(offset+limit, len(s.Datasets.Data))] {
				p.rows = append(p.rows, dashboardHTMLRow{name: d.Name, guid: d.GUID, cells: []string{d.GUID, dashboardBytes(d.Used), dashboardBytes(d.Available), dashboardBytes(d.Referenced)}})
				p.maximum = max(p.maximum, d.Used)
			}
			for i := range min(20, len(p.rows)) {
				d := s.Datasets.Data[offset+i]
				p.rows[i].measures = []dashboardHTMLMeasure{dashboardExpectedRatio(d.Used, p.maximum, "capacity-used")}
			}
		} else {
			for _, pool := range s.Pools.Data[:min(limit, len(s.Pools.Data))] {
				p.rows = append(p.rows, dashboardHTMLRow{name: pool.Name, guid: pool.GUID, cells: []string{pool.GUID, dashboardBytes(pool.Size), dashboardBytes(pool.Allocated), dashboardBytes(pool.Free), pool.Health}, zero: pool.Size == 0, measures: []dashboardHTMLMeasure{dashboardExpectedRatio(pool.Allocated, pool.Size, "capacity-allocated"), dashboardExpectedRatio(pool.Free, pool.Size, "capacity-free")}})
			}
		}
		pages[request.Path] = p
	}
	return func(request dashboardRequest, status int, body []byte) error {
		if request.Index < 0 || request.Index >= len(prepared) || request != prepared[request.Index] {
			return errors.New("response has wrong prepared request")
		}
		root, err := dashboardParseHTML(body)
		if err != nil {
			return err
		}
		workspace, err := root.one("main", "id", "workspace")
		if err != nil {
			return err
		}
		heading, err := workspace.one("h1", "", "")
		if err != nil || heading.text() != pages[request.Path].title {
			return errors.New("wrong dashboard view")
		}
		if len(workspace.find("script", "", "")) != 0 || len(workspace.find("aside", "class", "detail")) != 0 {
			return errors.New("unexpected response script")
		}
		if status == 409 && request.Case == "overview" {
			return dashboardExpiryHTML(workspace)
		}
		if status != 200 || len(workspace.find("section", "role", "alert")) != 0 {
			return errors.New("dashboard response denied")
		}
		g, err := workspace.one("p", "class", "generation")
		if err != nil {
			return err
		}
		code, err := g.one("code", "", "")
		if err != nil {
			return err
		}
		generation := code.rawText()
		if !dashboardGenerationPattern.MatchString(generation) {
			return errors.New("invalid response generation")
		}
		p := pages[request.Path]
		tableCount := 1
		if p.overview && p.status == inventory.OK {
			tableCount = 2
		}
		if len(workspace.find("table", "", "")) != tableCount {
			return errors.New("unexpected dashboard table count")
		}
		if p.overview {
			if err := dashboardOverviewHTML(workspace, s, generation); err != nil {
				return err
			}
		} else {
			if err := dashboardStorageMetadata(workspace, p, generation); err != nil {
				return err
			}
		}
		if p.overview && p.status != inventory.OK {
			return dashboardNoComparisons(workspace)
		}
		table, err := dashboardTable(workspace, p.caption)
		if err != nil {
			return err
		}
		if err := dashboardTableHTML(table, p, generation); err != nil {
			return err
		}
		if p.status != inventory.OK {
			return dashboardNoComparisons(workspace)
		}
		return dashboardCapacityHTML(workspace, p, generation)
	}, nil
}
func dashboardNoComparisons(workspace *dashboardHTMLNode) error {
	if len(workspace.find("section", "class", "capacity-section")) != 0 || len(workspace.find("svg", "", "")) != 0 || len(workspace.find("ol", "", "")) != 0 {
		return errors.New("unknown data has comparisons")
	}
	return nil
}
func dashboardExpiryHTML(w *dashboardHTMLNode) error {
	alert, err := w.one("section", "role", "alert")
	if err != nil {
		return err
	}
	h, err := alert.one("h2", "", "")
	if err != nil || h.text() != "Observation expired. Reload this view to use a current generation." {
		return errors.New("unsafe expiry explanation")
	}
	code, err := alert.one("p", "class", "mono")
	if err != nil || code.text() != "generation_expired" {
		return errors.New("wrong expiry outcome")
	}
	a, err := alert.one("a", "", "")
	if err != nil || a.attrs["href"] != "/" || a.text() != "Reload current observation" {
		return errors.New("unsafe expiry reload")
	}
	if len(w.find("section", "", "")) != 1 || len(w.find("p", "class", "generation")) != 0 {
		return errors.New("expiry retained a successful projection")
	}
	for _, tag := range []string{"table", "dl", "svg", "ol"} {
		if len(w.find(tag, "", "")) != 0 {
			return errors.New("expiry retained successful data")
		}
	}
	return nil
}
func dashboardStateLabel(label string, status inventory.Status) bool {
	if status == inventory.OK {
		return label == "Observed" || label == "Stale"
	}
	return label == string(status)
}
func dashboardValidTimestamp(value string, optional bool) bool {
	if optional && value == "Not collected" {
		return true
	}
	_, err := time.Parse("2006-01-02 15:04:05 UTC", value)
	return err == nil
}
func dashboardLink(path, generation string, offset, limit int, guid string) string {
	q := url.Values{"generation": {generation}, "sort": {"name"}, "filter": {""}, "descending": {"false"}, "offset": {strconv.Itoa(offset)}, "limit": {strconv.Itoa(limit)}}
	if guid != "" {
		q.Set("selected", guid)
	}
	return path + "?" + q.Encode()
}
func dashboardCheckLink(a *dashboardHTMLNode, expected string, htmx bool) bool {
	return a.attrs["href"] == expected && (!htmx || a.attrs["hx-get"] == expected && a.attrs["hx-target"] == "#workspace" && a.attrs["hx-swap"] == "outerHTML" && a.attrs["hx-push-url"] == "true")
}
func dashboardTable(w *dashboardHTMLNode, caption string) (*dashboardHTMLNode, error) {
	var found *dashboardHTMLNode
	for _, table := range w.find("table", "", "") {
		cs := table.direct("caption")
		if len(cs) == 1 && cs[0].text() == caption {
			if found != nil {
				return nil, errors.New("duplicate dashboard table")
			}
			found = table
		}
	}
	if found == nil {
		return nil, errors.New("complete dashboard table missing")
	}
	return found, nil
}

// Error text and serialized observations never carry observed values or identities.
type dashboardTableMismatch string

func (dashboardTableMismatch) Error() string { return "wrong original dashboard table" }

func dashboardTableHTML(table *dashboardHTMLNode, p dashboardHTMLPage, generation string) error {
	invalid := func(check string) error { return dashboardTableMismatch(check) }
	heads := table.direct("thead")
	bodies := table.direct("tbody")
	if len(heads) != 1 || len(bodies) != 1 {
		return invalid("structure")
	}
	expected := []string{"Name", "GUID", "Size", "Allocated", "Free", "Pool state"}
	if p.title == "Datasets" {
		expected = []string{"Name", "GUID", "Used", "Available", "Referenced"}
	}
	cols := heads[0].find("th", "", "")
	if len(cols) != len(expected) {
		return invalid("header-count")
	}
	for i, c := range cols {
		if c.text() != expected[i] || c.attrs["scope"] != "col" {
			return invalid("header")
		}
	}
	rows := bodies[0].direct("tr")
	if len(p.rows) == 0 {
		if len(rows) != 1 || len(rows[0].direct("th")) != 0 {
			return invalid("empty-shape")
		}
		cells := rows[0].direct("td")
		if len(cells) != 1 || cells[0].attrs["colspan"] != strconv.Itoa(len(expected)) {
			return invalid("empty-shape")
		}
		text := "No matching resources in this observation."
		if p.overview {
			text = "No observed pools in this generation."
		} else if p.status != inventory.OK {
			text = "This section is unavailable. No resource count is implied."
		}
		if cells[0].text() != text {
			return invalid("empty-text")
		}
		return nil
	}
	if len(rows) != len(p.rows) {
		return invalid("row-count")
	}
	for i, row := range rows {
		want := p.rows[i]
		th := row.direct("th")
		cells := row.direct("td")
		if len(th) != 1 || th[0].attrs["scope"] != "row" || len(cells) != len(want.cells) || row.attrs["class"] != "" {
			return invalid("row-shape")
		}
		a, err := th[0].one("a", "", "")
		if err != nil {
			return invalid("row-shape")
		}
		if a.rawText() != want.name {
			return invalid("row-name")
		}
		if a.attrs["aria-current"] != "" || !p.overview && a.attrs["id"] != "row-"+want.guid {
			return invalid("row-selection")
		}
		if !dashboardCheckLink(a, dashboardLink(p.path, generation, p.offset, p.limit, want.guid), !p.overview) {
			return invalid("row-link")
		}
		for j, c := range cells {
			if c.rawText() != want.cells[j] {
				fields := []string{"guid", "size", "allocated", "free", "pool-state"}
				if p.title == "Datasets" {
					fields = []string{"guid", "used", "available", "referenced"}
				}
				return invalid("cell-" + fields[j])
			}
			if c.attrs["class"] != "mono" {
				return invalid("cell-style")
			}
		}
	}
	return nil
}
func dashboardStorageMetadata(w *dashboardHTMLNode, p dashboardHTMLPage, generation string) error {
	invalid := func() error { return errors.New("wrong storage query or freshness metadata") }
	var header *dashboardHTMLNode
	for _, c := range w.direct("div") {
		if c.attrs["class"] == "section-heading" {
			if header != nil {
				return invalid()
			}
			header = c
		}
	}
	if header == nil {
		return invalid()
	}
	ps := header.direct("p")
	if len(ps) != 2 {
		return invalid()
	}
	prefix := fmt.Sprintf("%d matching items · ", p.total)
	if p.status != inventory.OK {
		prefix = "Resource count unavailable · "
	}
	if !strings.HasPrefix(ps[0].text(), prefix) || !dashboardStateLabel(strings.TrimPrefix(ps[0].text(), prefix), p.status) || !strings.HasPrefix(ps[1].text(), "Section read ") || !dashboardValidTimestamp(strings.TrimPrefix(ps[1].text(), "Section read "), false) {
		return invalid()
	}
	form, err := w.one("form", "class", "filters")
	if err != nil || form.attrs["action"] != p.path {
		return invalid()
	}
	for _, want := range [][2]string{{"generation", generation}, {"limit", strconv.Itoa(p.limit)}, {"filter", ""}} {
		input, e := form.one("input", "name", want[0])
		if e != nil || input.attrs["value"] != want[1] {
			return invalid()
		}
	}
	for _, want := range [][2]string{{"sort", "name"}, {"descending", "false"}} {
		sel, e := form.one("select", "name", want[0])
		if e != nil {
			return invalid()
		}
		selected := 0
		for _, option := range sel.direct("option") {
			if _, ok := option.attrs["selected"]; ok {
				selected++
				if option.attrs["value"] != want[1] {
					return invalid()
				}
			}
		}
		if selected != 1 {
			return invalid()
		}
	}
	nav, err := w.one("nav", "class", "pagination")
	if err != nil {
		return invalid()
	}
	span, err := nav.one("span", "", "")
	if err != nil || span.text() != fmt.Sprintf("Offset %d · Up to %d items", p.offset, p.limit) {
		return invalid()
	}
	links := map[string]string{}
	if p.offset > 0 {
		links["Previous page"] = dashboardLink(p.path, generation, max(0, p.offset-p.limit), p.limit, "")
	}
	if p.offset+len(p.rows) < p.total {
		links["Next page"] = dashboardLink(p.path, generation, p.offset+len(p.rows), p.limit, "")
	}
	anchors := nav.direct("a")
	if len(anchors) != len(links) {
		return invalid()
	}
	for _, a := range anchors {
		want, ok := links[a.text()]
		if !ok || !dashboardCheckLink(a, want, true) {
			return invalid()
		}
		delete(links, a.text())
	}
	return nil
}
func dashboardOverviewHTML(w *dashboardHTMLNode, s inventory.Snapshot, generation string) error {
	invalid := func() error { return errors.New("wrong original Overview metadata") }
	var host *dashboardHTMLNode
	for _, section := range w.find("section", "class", "section") {
		headings := section.direct("h2")
		for _, h := range headings {
			if h.text() == "Host facts" {
				if host != nil {
					return invalid()
				}
				host = section
			}
		}
		for _, d := range section.direct("div") {
			for _, h := range d.direct("h2") {
				if h.text() == "Host facts" {
					if host != nil {
						return invalid()
					}
					host = section
				}
			}
		}
	}
	if host == nil {
		return invalid()
	}
	badge, err := host.one("span", "class", "badge")
	if err != nil || !dashboardStateLabel(badge.text(), s.Host.Status) {
		return invalid()
	}
	facts, err := host.one("dl", "class", "facts")
	if err != nil {
		return invalid()
	}
	known := func(v string) string {
		if v == "" {
			return "Unavailable"
		}
		return v
	}
	memory := "Unavailable"
	if s.Host.Data.PhysicalMemory != nil {
		memory = dashboardBytes(*s.Host.Data.PhysicalMemory)
	}
	expected := [][2]string{{"Installed kernel", known(s.Host.Data.InstalledKernel)}, {"Running kernel", known(s.Host.Data.RunningKernel)}, {"Userland", known(s.Host.Data.Userland)}, {"Architecture", known(s.Host.Data.Architecture)}, {"ZFS userland", known(s.Host.Data.ZFSUserland)}, {"ZFS module", known(s.Host.Data.ZFSKernel)}, {"Physical memory", memory}}
	items := facts.direct("div")
	if len(items) != len(expected) {
		return invalid()
	}
	for i, item := range items {
		dt, e1 := item.one("dt", "", "")
		dd, e2 := item.one("dd", "", "")
		if e1 != nil || e2 != nil || dt.text() != expected[i][0] || dd.rawText() != expected[i][1] {
			return invalid()
		}
	}
	paragraphs := host.direct("p")
	if len(paragraphs) != 1 || !strings.HasPrefix(paragraphs[0].text(), "Host observation: ") || !dashboardValidTimestamp(strings.TrimPrefix(paragraphs[0].text(), "Host observation: "), false) {
		return invalid()
	}
	table, err := dashboardTable(w, "Sections in this generation")
	if err != nil {
		return err
	}
	headers := table.find("th", "scope", "col")
	labels := []string{"Section", "Items", "Observation", "Last successful read"}
	if len(headers) != 4 {
		return invalid()
	}
	for i, header := range headers {
		if header.text() != labels[i] {
			return invalid()
		}
	}
	tbody, err := table.one("tbody", "", "")
	if err != nil {
		return err
	}
	rows := tbody.direct("tr")
	if len(rows) != 3 {
		return invalid()
	}
	specs := []struct {
		name, path string
		count      int
		status     inventory.Status
	}{{"Pools", "/storage/pools", len(s.Pools.Data), s.Pools.Status}, {"Datasets", "/storage/datasets", len(s.Datasets.Data), s.Datasets.Status}, {"GEOM providers", "/storage/providers", len(s.GEOM.Data.Providers), s.GEOM.Status}}
	for i, row := range rows {
		a, e := row.one("a", "", "")
		want := specs[i]
		if e != nil || a.rawText() != want.name || a.attrs["href"] != want.path+"?"+(url.Values{"generation": {generation}}).Encode() {
			return invalid()
		}
		cells := row.direct("td")
		count := strconv.Itoa(want.count)
		if want.status != inventory.OK {
			count = "Unavailable"
		}
		if len(cells) != 3 || cells[0].rawText() != count || !dashboardStateLabel(cells[1].text(), want.status) || !dashboardValidTimestamp(cells[2].text(), want.status != inventory.OK) {
			return invalid()
		}
	}
	return nil
}
func dashboardCapacityHTML(w *dashboardHTMLNode, p dashboardHTMLPage, generation string) error {
	invalid := func() error { return errors.New("wrong original dashboard comparisons") }
	section, err := w.one("section", "class", "capacity-section")
	if err != nil {
		return err
	}
	title := "Pool capacity comparisons"
	if p.overview {
		title = "Observed pool capacity"
	} else if p.title == "Datasets" {
		title = "Dataset Used comparisons"
	}
	h, err := section.one("h2", "id", "capacity-title")
	if err != nil || h.text() != title {
		return invalid()
	}
	badge, err := section.one("span", "class", "badge")
	if err != nil || !dashboardStateLabel(badge.text(), p.status) {
		return invalid()
	}
	coverage, err := section.one("p", "class", "capacity-coverage")
	if err != nil {
		return err
	}
	n := min(20, len(p.rows))
	prefix := ""
	if p.overview {
		prefix = fmt.Sprintf("First %d pools by name in observation %s; %d observed pools", n, generation, p.total)
		if p.total > 20 {
			prefix += "; more available"
		}
		prefix += ". At most 20 comparisons from one 100-row page."
		if n == 0 {
			prefix += " No pool comparisons in this observation."
		}
	} else {
		kind := "pools"
		if p.title == "Datasets" {
			kind = "datasets"
		}
		if n > 0 {
			prefix = fmt.Sprintf("Showing %d–%d of %d matching %s (up to 20 comparisons per page).", p.offset+1, p.offset+n, p.total, kind)
		} else {
			prefix = fmt.Sprintf("No comparisons on this page · %d matching %s.", p.total, kind)
		}
	}
	text := coverage.text()
	prefix += " Observation " + generation + " · Filter: None · Sort: name · Ascending. Section read "
	if !strings.HasPrefix(text, prefix) {
		return invalid()
	}
	times := strings.Split(strings.TrimPrefix(text, prefix), "; last successful read ")
	if len(times) != 2 || !strings.HasSuffix(times[1], ".") {
		return invalid()
	}
	read, success := times[0], strings.TrimSuffix(times[1], ".")
	if !dashboardValidTimestamp(read, false) || read != success {
		return invalid()
	}
	if !p.overview {
		var readHeader string
		for _, header := range w.direct("div") {
			if header.attrs["class"] == "section-heading" {
				for _, paragraph := range header.direct("p") {
					if strings.HasPrefix(paragraph.text(), "Section read ") {
						readHeader = strings.TrimPrefix(paragraph.text(), "Section read ")
					}
				}
			}
		}
		if readHeader != read {
			return invalid()
		}
	} else {
		table, e := dashboardTable(w, "Sections in this generation")
		if e != nil {
			return e
		}
		tbody, e := table.one("tbody", "", "")
		if e != nil {
			return e
		}
		if tbody.direct("tr")[0].direct("td")[2].text() != success {
			return invalid()
		}
	}
	if len(w.find("svg", "", "")) != len(section.find("svg", "", "")) || len(w.find("ol", "", "")) != 1 {
		return invalid()
	}
	list, err := section.one("ol", "class", "capacity-list")
	if err != nil {
		return err
	}
	items := list.direct("li")
	if len(items) != n {
		return invalid()
	}
	if p.title == "Datasets" && n > 0 {
		scale := fmt.Sprintf("Scale: largest Used on this page · %s. The scale includes table rows beyond the comparisons and may change with filtering, ordering or pagination.", dashboardBytes(p.maximum))
		found := 0
		for _, paragraph := range section.direct("p") {
			if paragraph.text() == scale {
				found++
			}
		}
		if found != 1 {
			return invalid()
		}
		if p.maximum == 0 && !strings.Contains(section.text(), "Scale unavailable: every Used value on this page is zero.") {
			return invalid()
		}
	}
	for i, item := range items {
		row := p.rows[i]
		a, e := item.one("a", "", "")
		if e != nil || a.attrs["id"] != "capacity-"+row.guid || a.rawText() != row.name || !dashboardCheckLink(a, dashboardLink(p.path, generation, p.offset, p.limit, row.guid), true) {
			return invalid()
		}
		svgs := item.find("svg", "", "")
		if row.zero || p.title == "Datasets" && p.maximum == 0 {
			message := "Ratio unavailable: reported size is zero."
			if p.title == "Datasets" {
				message = "Used · 0 B; no nonzero page scale."
			}
			if len(svgs) != 0 || !strings.Contains(item.text(), message) {
				return invalid()
			}
			continue
		}
		viewBox, plotClass := "0 0 1000 36", "capacity-plot"
		if p.title == "Datasets" {
			viewBox = "0 0 1000 16"
			plotClass = "capacity-plot capacity-single"
		}
		if len(svgs) != 1 || svgs[0].attrs["aria-hidden"] != "true" || svgs[0].attrs["focusable"] != "false" || svgs[0].attrs["viewBox"] != viewBox || svgs[0].attrs["class"] != plotClass || svgs[0].attrs["preserveAspectRatio"] != "none" || svgs[0].attrs["xmlns"] != "http://www.w3.org/2000/svg" {
			return invalid()
		}
		measurements := map[string]*dashboardHTMLNode{}
		rects := svgs[0].direct("rect")
		if len(rects) != 2*len(row.measures) || len(svgs[0].children) != len(rects) {
			return invalid()
		}
		for k, rect := range rects {
			y := "0"
			if k >= 2 {
				y = "20"
			}
			if rect.attrs["x"] != "0" || rect.attrs["y"] != y || rect.attrs["height"] != "16" {
				return invalid()
			}
			if k%2 == 0 && (rect.attrs["class"] != "capacity-track" || rect.attrs["width"] != "1000") || k%2 == 1 && rect.attrs["class"] != row.measures[k/2].class {
				return invalid()
			}
		}
		for _, rect := range rects {
			if rect.attrs["class"] == "capacity-track" {
				continue
			}
			class := rect.attrs["class"]
			if measurements[class] != nil {
				return invalid()
			}
			measurements[class] = rect
		}
		if len(measurements) != len(row.measures) {
			return invalid()
		}
		values, e := item.one("p", "class", "capacity-values")
		if e != nil {
			return e
		}
		spans := values.direct("span")
		if len(spans) != len(row.measures) {
			return invalid()
		}
		for j, m := range row.measures {
			rect := measurements[m.class]
			if rect == nil || rect.attrs["x"] != "0" || rect.attrs["width"] != m.width {
				return invalid()
			}
			label := "Allocated · approximately " + m.percent
			if m.class == "capacity-free" {
				label = "Free · approximately " + m.percent
			} else if m.class == "capacity-used" {
				label = "Used · approximately " + m.percent + " of page scale"
			}
			if spans[j].text() != label {
				return invalid()
			}
		}
	}
	return nil
}
