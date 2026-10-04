// SPDX-License-Identifier: BSD-3-Clause
// Package client is the authenticated loopback HTTP adapter used by the CLI.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gonz0w/horvium/internal/admin"
	"github.com/gonz0w/horvium/internal/inventory"
	"github.com/gonz0w/horvium/internal/observation"
)

var ErrResponse = errors.New("management response unavailable")

// Client has one volatile operator session. It is not safe for concurrent use.
type Client struct {
	base string
	http *http.Client
	csrf string
}

func New(endpoint string) (*Client, error) {
	u, e := url.Parse(endpoint)
	if e != nil || u.Scheme != "http" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.ForceQuery {
		return nil, ErrResponse
	}
	ip, e := netip.ParseAddr(u.Hostname())
	port, pe := strconv.Atoi(u.Port())
	if e != nil || pe != nil || port < 1 || port > 65535 || !ip.IsLoopback() || ip.Zone() != "" || ip.Is4In6() || net.JoinHostPort(ip.String(), strconv.Itoa(port)) != u.Host {
		return nil, ErrResponse
	}
	jar, _ := cookiejar.New(nil)
	// No environment proxy or redirects: never forward an operator credential.
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, DisableCompression: true, MaxResponseHeaderBytes: 16 << 10, MaxConnsPerHost: 2, ResponseHeaderTimeout: 5 * time.Second}
	return &Client{base: endpoint, http: &http.Client{Transport: transport, Jar: jar, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) exchange(ctx context.Context, method, path string, form url.Values, out any) error {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	r, e := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if e != nil {
		return ErrResponse
	}
	if form != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	res, e := c.http.Do(r)
	if e != nil {
		return ErrResponse
	}
	defer res.Body.Close()
	if res.StatusCode == 401 || res.StatusCode == 403 {
		return admin.ErrUnauthorized
	}
	if res.StatusCode == 429 {
		return admin.ErrThrottled
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, (64<<20)+1))
	if e != nil || len(b) > 64<<20 {
		return ErrResponse
	}
	if res.StatusCode != 200 && res.StatusCode != 202 && res.StatusCode != 204 {
		var problem struct{ Code, Message string }
		if json.Unmarshal(b, &problem) != nil {
			return ErrResponse
		}
		switch {
		case res.StatusCode == 400 && problem.Code == "invalid_query":
			return observation.ErrQuery
		case res.StatusCode == 409 && problem.Code == "generation_expired":
			return observation.ErrExpired
		case res.StatusCode == 409 && problem.Code == "busy":
			return observation.ErrBusy
		case res.StatusCode == 503 && problem.Code == "unavailable":
			return observation.ErrUnavailable
		}
		return ErrResponse
	}
	if out == nil {
		if res.StatusCode != 204 {
			return ErrResponse
		}
		return nil
	}
	if res.Header.Get("Content-Type") != "application/json" {
		return ErrResponse
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return ErrResponse
	}
	if d.Decode(new(any)) != io.EOF {
		return ErrResponse
	}
	return nil
}
func (c *Client) Login(ctx context.Context, token string) error {
	if c.csrf != "" || len(token) != 64 {
		return admin.ErrUnauthorized
	}
	for _, b := range token {
		if !(b >= '0' && b <= '9' || b >= 'a' && b <= 'f') {
			return admin.ErrUnauthorized
		}
	}
	var challenge admin.Challenge
	if e := c.exchange(ctx, "GET", "/auth/challenge", nil, &challenge); e != nil {
		return e
	}
	var session admin.Session
	if e := c.exchange(ctx, "POST", "/auth/login", url.Values{"token": {token}, "csrf": {challenge.CSRF}}, &session); e != nil {
		return e
	}
	if len(session.CSRF) != 64 {
		return ErrResponse
	}
	c.csrf = session.CSRF
	return nil
}
func (c *Client) Query(ctx context.Context, kind string, q observation.Query) (any, error) {
	values := url.Values{}
	if q.Generation != "" {
		values.Set("generation", q.Generation)
	}
	if q.Filter != "" {
		values.Set("filter", q.Filter)
	}
	if q.Sort != "" {
		values.Set("sort", q.Sort)
	}
	if q.Offset != 0 {
		values.Set("offset", strconv.Itoa(q.Offset))
	}
	if q.Limit != 0 {
		values.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Descending {
		values.Set("descending", "true")
	}
	path := "/api/" + kind
	if len(values) > 0 {
		path += "?" + values.Encode()
	}
	switch kind {
	case "overview":
		var out observation.Overview
		e := c.exchange(ctx, "GET", path, nil, &out)
		return out, e
	case "pools":
		var out observation.Page[inventory.Pool]
		e := c.exchange(ctx, "GET", path, nil, &out)
		return out, e
	case "datasets":
		var out observation.Page[inventory.Dataset]
		e := c.exchange(ctx, "GET", path, nil, &out)
		return out, e
	case "providers":
		var out observation.Page[inventory.Provider]
		e := c.exchange(ctx, "GET", path, nil, &out)
		return out, e
	default:
		return nil, observation.ErrQuery
	}
}
func (c *Client) Refresh(ctx context.Context) error {
	var out struct{ Code string }
	return c.exchange(ctx, "POST", "/api/refresh", url.Values{"csrf": {c.csrf}}, &out)
}

// Logout drops local cookies even when the server is unreachable.
func (c *Client) Logout(ctx context.Context) error {
	e := c.exchange(ctx, "POST", "/auth/logout", url.Values{"csrf": {c.csrf}}, nil)
	c.csrf = ""
	c.http.Jar, _ = cookiejar.New(nil)
	c.http.CloseIdleConnections()
	return e
}
