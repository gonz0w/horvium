// SPDX-License-Identifier: BSD-3-Clause
// Package console adapts cached observation contracts to HTML and internal M1 JSON.
// There is no native execution, privileged setup or storage mutation here.
package console

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

//go:embed assets/*
var assets embed.FS

type app struct {
	manager   *observation.Manager
	gate      *admin.Gate
	templates *template.Template
}
type fact struct{ Label, Value string }
type row struct {
	Key, Name, URL string
	Cells          []string
	Facts          []fact
	Selected       bool
}
type option struct{ Value, Label string }
type view struct {
	Title, Path, Generation, CSRF, Selected, Error, ErrorCode, Filter, Sort, Desc, Next, Previous string
	Offset, Limit, Total                                                                          int
	Columns                                                                                       []string
	Rows                                                                                          []row
	Facts                                                                                         []fact
	Options                                                                                       []option
	Detail                                                                                        *row
	Overview                                                                                      *observation.Overview
	State                                                                                         observation.State
	Freshness                                                                                     observation.Freshness
}

func New(c admin.Config, a *admin.Auth, m *observation.Manager) (http.Handler, error) {
	if m == nil {
		return nil, admin.ErrConfig
	}
	g, e := admin.NewGate(c, a)
	if e != nil {
		return nil, e
	}
	t, e := template.New("console").Funcs(template.FuncMap{"columns": func(columns []string) int { return len(columns) + 1 }, "time": timestamp, "status": freshness, "bytes": bytesValue, "known": known, "count": count, "same": func(a, b string) bool { return a == b }}).ParseFS(assets, "assets/*.html")
	if e != nil {
		return nil, e
	}
	app := &app{m, g, t}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/challenge", g.Challenge)
	mux.HandleFunc("POST /auth/login", g.Login)
	mux.HandleFunc("POST /auth/logout", g.Logout)
	mux.HandleFunc("GET /login", app.login)
	mux.HandleFunc("POST /login", app.login)
	mux.HandleFunc("POST /logout", func(w http.ResponseWriter, r *http.Request) {
		if g.EndSession(w, r) {
			http.Redirect(w, r, "/login", 303)
		}
	})
	for _, path := range []string{"/", "/storage/pools", "/storage/datasets", "/storage/providers"} {
		pattern := path
		if path == "/" {
			pattern = "/{$}"
		}
		mux.Handle("GET "+pattern, g.ProtectPage(http.HandlerFunc(app.page)))
	}
	for _, kind := range []string{"overview", "pools", "datasets", "providers"} {
		mux.Handle("GET /api/"+kind, g.Protect(http.HandlerFunc(app.api)))
	}
	mux.Handle("GET /refresh", g.ProtectPage(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "Method not allowed", 405) })))
	mux.Handle("POST /refresh", g.ProtectPage(http.HandlerFunc(app.refresh)))
	mux.Handle("POST /api/refresh", g.Protect(http.HandlerFunc(app.refresh)))
	mux.Handle("GET /status", g.ProtectPage(http.HandlerFunc(app.status)))
	mux.Handle("GET /assets/", g.Public(http.HandlerFunc(app.asset)))
	// Missing assets and unknown paths must not mint a login challenge.
	mux.Handle("GET /", g.Public(http.NotFoundHandler()))
	return mux, nil
}
func known(s string) string {
	if s == "" {
		return "Unavailable"
	}
	return s
}
func timestamp(t time.Time) string {
	if t.IsZero() {
		return "Not collected"
	}
	return t.UTC().Format("2006-01-02 15:04:05 UTC")
}
func bytesValue(v uint64) string { return strconv.FormatUint(v, 10) + " B" }
func count(n int, f observation.Freshness) string {
	if f.Status != inventory.OK {
		return "Unavailable"
	}
	return strconv.Itoa(n)
}
func freshness(f observation.Freshness) string {
	if f.Status == "" {
		return "Not collected"
	}
	if f.Status != inventory.OK {
		return string(f.Status)
	}
	if f.Stale {
		return "Stale"
	}
	return "Observed"
}
func (a *app) asset(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/assets/")
	typ := ""
	switch name {
	case "horvium.svg":
		typ = "image/svg+xml"
	case "console.css":
		typ = "text/css; charset=utf-8"
	case "console.js", "htmx.min.js":
		typ = "text/javascript; charset=utf-8"
	default:
		http.NotFound(w, r)
		return
	}
	b, e := assets.ReadFile("assets/" + name)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", typ)
	_, _ = w.Write(b)
}
func (a *app) login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if _, ok := a.gate.ExchangeLogin(w, r); ok {
			http.Redirect(w, r, "/", 303)
		}
		return
	}
	c, ok := a.gate.BeginLogin(w, r)
	if !ok {
		return
	}
	a.render(w, "login", view{CSRF: c.CSRF, Title: "Sign in"}, 200)
}
func parseQuery(r *http.Request, html bool) (observation.Query, string, error) {
	values, e := url.ParseQuery(r.URL.RawQuery)
	if e != nil {
		return observation.Query{}, "", observation.ErrQuery
	}
	allowed := map[string]bool{"generation": true, "filter": true, "sort": true, "descending": true, "offset": true, "limit": true}
	if html {
		allowed["selected"] = true
	}
	for k, v := range values {
		if !allowed[k] || len(v) != 1 || len(v[0]) > 4096 {
			return observation.Query{}, "", observation.ErrQuery
		}
	}
	q := observation.Query{Generation: values.Get("generation"), Filter: values.Get("filter"), Sort: values.Get("sort")}
	for key, target := range map[string]*int{"offset": &q.Offset, "limit": &q.Limit} {
		if s, ok := values[key]; ok {
			n, e := strconv.Atoi(s[0])
			if e != nil || n < 0 {
				return q, "", observation.ErrQuery
			}
			*target = n
		}
	}
	if d, ok := values["descending"]; ok {
		if d[0] != "true" && d[0] != "false" {
			return q, "", observation.ErrQuery
		}
		q.Descending = d[0] == "true"
	}
	return q, values.Get("selected"), nil
}
func queryURL(path string, q observation.Query, selected string) string {
	v := url.Values{"generation": {q.Generation}, "offset": {strconv.Itoa(q.Offset)}, "limit": {strconv.Itoa(q.Limit)}, "sort": {q.Sort}, "filter": {q.Filter}, "descending": {strconv.FormatBool(q.Descending)}}
	if selected != "" {
		v.Set("selected", selected)
	}
	return path + "?" + v.Encode()
}
func resultError(e error) (int, string, string) {
	switch {
	case errors.Is(e, observation.ErrQuery):
		return 400, "invalid_query", "Invalid view options"
	case errors.Is(e, observation.ErrExpired):
		return 409, "generation_expired", "Observation expired. Reload this view to use a current generation."
	case errors.Is(e, observation.ErrBusy):
		return 409, "busy", "Inventory work already active"
	case errors.Is(e, observation.ErrUnavailable), errors.Is(e, observation.ErrNotRunning):
		return 503, "unavailable", "Inventory unavailable"
	default:
		return 503, "unavailable", "Inventory unavailable"
	}
}
func (a *app) fail(w http.ResponseWriter, r *http.Request, e error, jsonMode bool, v view) {
	status, code, message := resultError(e)
	if jsonMode {
		writeJSON(w, map[string]string{"code": code, "message": message}, status)
		return
	}
	v.Error = message
	v.ErrorCode = code
	if r.Header.Get("HX-Request") == "true" {
		a.render(w, "workspace", v, status)
	} else {
		a.render(w, "page", v, status)
	}
}
func (a *app) render(w http.ResponseWriter, name string, v view, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Vary", "HX-Request")
	var output bytes.Buffer
	if a.templates.ExecuteTemplate(&output, name, v) != nil {
		http.Error(w, "Rendering unavailable", 503)
		return
	}
	w.WriteHeader(status)
	_, _ = w.Write(output.Bytes())
}
func writeJSON(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func (a *app) api(w http.ResponseWriter, r *http.Request) {
	q, _, e := parseQuery(r, false)
	if e != nil {
		a.fail(w, r, e, true, view{})
		return
	}
	var result any
	switch r.URL.Path {
	case "/api/overview":
		if q.Filter != "" || q.Sort != "" || q.Offset != 0 || q.Limit != 0 || q.Descending {
			e = observation.ErrQuery
		} else {
			result, e = a.manager.Overview(r.Context(), q.Generation)
		}
	case "/api/pools":
		result, e = a.manager.Pools(r.Context(), q)
	case "/api/datasets":
		result, e = a.manager.Datasets(r.Context(), q)
	case "/api/providers":
		result, e = a.manager.Providers(r.Context(), q)
	}
	if e != nil {
		a.fail(w, r, e, true, view{})
		return
	}
	writeJSON(w, result, 200)
}
func (a *app) page(w http.ResponseWriter, r *http.Request) {
	titles := map[string]string{"/": "Overview", "/storage/pools": "Pools", "/storage/datasets": "Datasets", "/storage/providers": "Providers"}
	title, ok := titles[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	v := view{Title: title, Path: r.URL.Path, CSRF: admin.RequestCSRF(r), State: a.manager.State()}
	q, selected, e := parseQuery(r, true)
	if e != nil {
		a.fail(w, r, e, false, v)
		return
	}
	if r.URL.Path == "/" && (q.Filter != "" || q.Sort != "" || q.Offset != 0 || q.Limit != 0 || q.Descending || selected != "") {
		a.fail(w, r, observation.ErrQuery, false, v)
		return
	}
	if q.Limit == 0 {
		q.Limit = 100
	}
	if q.Sort == "" {
		q.Sort = "name"
	}
	v.Filter = q.Filter
	v.Sort = q.Sort
	v.Limit = q.Limit
	v.Offset = q.Offset
	v.Desc = strconv.FormatBool(q.Descending)
	if r.URL.Path == "/" {
		if q.Filter != "" || q.Offset != 0 || selected != "" {
			a.fail(w, r, observation.ErrQuery, false, v)
			return
		}
		o, e := a.manager.Overview(r.Context(), q.Generation)
		if e != nil {
			a.fail(w, r, e, false, v)
			return
		}
		v.Overview = &o
		v.Generation = o.Generation
		v.Facts = []fact{{"Installed kernel", known(o.Host.InstalledKernel)}, {"Running kernel", known(o.Host.RunningKernel)}, {"Userland", known(o.Host.Userland)}, {"Architecture", known(o.Host.Architecture)}, {"ZFS userland", known(o.Host.ZFSUserland)}, {"ZFS module", known(o.Host.ZFSKernel)}}
		if o.Host.PhysicalMemory != nil {
			v.Facts = append(v.Facts, fact{"Physical memory", bytesValue(*o.Host.PhysicalMemory)})
		} else {
			v.Facts = append(v.Facts, fact{"Physical memory", "Unavailable"})
		}
	} else {
		var more bool
		switch r.URL.Path {
		case "/storage/pools":
			p, err := a.manager.Pools(r.Context(), q)
			e = err
			v.Generation = p.Generation
			v.Total = p.Total
			v.Freshness = p.Freshness
			more = p.More
			v.Columns = []string{"GUID", "Size", "Allocated", "Free", "Pool state"}
			v.Options = sortOptions("name", "guid", "size", "allocated", "free", "health")
			for _, p := range p.Rows {
				v.Rows = append(v.Rows, row{Key: p.GUID, Name: p.Name, Cells: []string{p.GUID, bytesValue(p.Size), bytesValue(p.Allocated), bytesValue(p.Free), p.Health}, Facts: []fact{{"Pool GUID", p.GUID}, {"Reported pool state", p.Health}, {"Size", bytesValue(p.Size)}, {"Allocated", bytesValue(p.Allocated)}, {"Free", bytesValue(p.Free)}}})
			}
		case "/storage/datasets":
			p, err := a.manager.Datasets(r.Context(), q)
			e = err
			v.Generation = p.Generation
			v.Total = p.Total
			v.Freshness = p.Freshness
			more = p.More
			v.Columns = []string{"GUID", "Used", "Available", "Referenced"}
			v.Options = sortOptions("name", "guid", "used", "available", "referenced")
			for _, d := range p.Rows {
				v.Rows = append(v.Rows, row{Key: d.GUID, Name: d.Name, Cells: []string{d.GUID, bytesValue(d.Used), bytesValue(d.Available), bytesValue(d.Referenced)}, Facts: []fact{{"Dataset GUID", d.GUID}, {"Used", bytesValue(d.Used)}, {"Available", bytesValue(d.Available)}, {"Referenced", bytesValue(d.Referenced)}}})
			}
		case "/storage/providers":
			p, err := a.manager.Providers(r.Context(), q)
			e = err
			v.Generation = p.Generation
			v.Total = p.Total
			v.Freshness = p.Freshness
			more = p.More
			v.Columns = []string{"Class", "Kind", "Media size", "Identity"}
			v.Options = sortOptions("name", "class", "kind", "size", "identity")
			for _, p := range p.Rows {
				facts := []fact{{"Collection-local ID", p.ID}, {"Collection-local GEOM ID", p.GeomID}, {"Class", p.Class}, {"Kind", string(p.Kind)}, {"Media size", bytesValue(p.MediaSize)}, {"Sector size", bytesValue(p.SectorSize)}, {"Identity confidence", string(p.Identity)}}
				for _, id := range p.Identifiers {
					facts = append(facts, fact{id.Kind + " · " + id.Source, id.Value})
				}
				v.Rows = append(v.Rows, row{Key: p.ID, Name: p.Name, Cells: []string{p.Class, string(p.Kind), bytesValue(p.MediaSize), string(p.Identity)}, Facts: facts})
			}
		}
		if e != nil {
			a.fail(w, r, e, false, v)
			return
		}
		q.Generation = v.Generation
		for i := range v.Rows {
			v.Rows[i].URL = queryURL(v.Path, q, v.Rows[i].Key)
			if selected == v.Rows[i].Key {
				v.Rows[i].Selected = true
				v.Detail = &v.Rows[i]
			}
		}
		if selected != "" && v.Detail == nil {
			a.fail(w, r, observation.ErrQuery, false, v)
			return
		}
		if more {
			next := q
			next.Offset += len(v.Rows)
			v.Next = queryURL(v.Path, next, "")
		}
		if q.Offset > 0 {
			prev := q
			prev.Offset = max(0, q.Offset-q.Limit)
			v.Previous = queryURL(v.Path, prev, "")
		}
	}
	if r.Header.Get("HX-Request") == "true" {
		a.render(w, "workspace", v, 200)
	} else {
		a.render(w, "page", v, 200)
	}
}
func sortOptions(values ...string) []option {
	out := make([]option, len(values))
	for i, v := range values {
		out[i] = option{v, strings.ToUpper(v[:1]) + v[1:]}
	}
	return out
}
func (a *app) status(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		http.Error(w, "Invalid view options", 400)
		return
	}
	a.render(w, "collector", view{State: a.manager.State()}, 200)
}
func (a *app) refresh(w http.ResponseWriter, r *http.Request) {
	jsonMode := r.URL.Path == "/api/refresh"
	allowed := map[string]bool{"csrf": true}
	if !jsonMode {
		allowed["return"] = true
	}
	if r.URL.RawQuery != "" {
		a.fail(w, r, observation.ErrQuery, jsonMode, view{})
		return
	}
	for k, vs := range r.PostForm {
		if !allowed[k] || len(vs) != 1 {
			a.fail(w, r, observation.ErrQuery, jsonMode, view{})
			return
		}
	}
	path := r.PostForm.Get("return")
	if path == "" {
		path = "/"
	}
	switch path {
	case "/", "/storage/pools", "/storage/datasets", "/storage/providers":
	default:
		a.fail(w, r, observation.ErrQuery, jsonMode, view{})
		return
	}
	e := a.manager.Refresh()
	if jsonMode {
		if e != nil {
			a.fail(w, r, e, true, view{})
			return
		}
		writeJSON(w, map[string]string{"code": "refresh_requested"}, 202)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		v := view{Path: path}
		if e != nil {
			_, v.ErrorCode, v.Error = resultError(e)
		}
		a.render(w, "refresh", v, 200)
		return
	}
	if e != nil {
		a.fail(w, r, e, false, view{Title: "Inventory", Path: path, CSRF: admin.RequestCSRF(r), State: a.manager.State()})
		return
	}
	http.Redirect(w, r, path, 303)
}
