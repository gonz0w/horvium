// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func workerAccounting(limit int) *propertyServer {
	s := &propertyServer{config: Config{MaxClients: limit}, active: make(map[*propertyConnection]struct{})}
	s.identity = s
	return s
}
func TestPropertyWorkerRetirementIndependentOfSocket(t *testing.T) {
	p := connectionAdmission(t, context.Background(), 1, time.Second)
	s := workerAccounting(1)
	_, first := connected(t)
	c, err := p.admit(first)
	if err != nil {
		t.Fatal(err)
	}
	if s.reserve(c) != nil {
		t.Fatal("original worker not reserved")
	}
	if c.Close() != nil {
		t.Fatal("socket retirement")
	}
	_, second := connected(t)
	next, err := p.admit(second)
	if err != nil {
		t.Fatal("socket slot not reused", err)
	}
	if !errors.Is(s.reserve(next), ErrLimit) {
		t.Fatal("retired socket released a live worker slot")
	}
	copy := *s
	copy.finish(c)
	if !errors.Is(s.reserve(next), ErrLimit) {
		t.Fatal("copied owner released original worker")
	}
	s.finish(c)
	s.finish(c)
	if s.reserve(next) != nil {
		t.Fatal("finished worker slot not reused")
	}
	s.finish(next)
	if len(s.active) != 0 {
		t.Fatal("worker reservation retained")
	}
}
func TestPropertyWorkerOriginalLease(t *testing.T) {
	p := connectionAdmission(t, context.Background(), 2, time.Second)
	s := workerAccounting(2)
	_, raw := connected(t)
	c, err := p.admit(raw)
	if err != nil {
		t.Fatal(err)
	}
	copy := *c
	for _, lease := range []*propertyConnection{nil, {}, &copy} {
		if s.reserve(lease) == nil {
			t.Fatal("unissued/copied lease reserved worker")
		}
	}
	if s.reserve(c) != nil {
		t.Fatal("original lease denied")
	}
	if !errors.Is(s.reserve(c), ErrProtocol) {
		t.Fatal("same worker reserved twice")
	}
	s.finish(&copy)
	if len(s.active) != 1 {
		t.Fatal("copied lease finished original worker")
	}
	s.finish(c)
}
func TestPropertyServerUnissued(t *testing.T) {
	var decoded propertyServer
	if json.Unmarshal([]byte(`{}`), &decoded) != nil {
		t.Fatal("fixture")
	}
	for _, s := range []*propertyServer{nil, {}, &decoded} {
		if s.serve(context.Background()) == nil {
			t.Fatal("unissued listener")
		}
		if s.reserve(nil) == nil {
			t.Fatal("unissued worker")
		}
		s.finish(nil)
	}
}

func TestPropertyWorkerConcurrentLastSlot(t *testing.T) {
	p := connectionAdmission(t, context.Background(), 32, 5*time.Second)
	s := workerAccounting(1)
	leases := make([]*propertyConnection, 32)
	for i := range leases {
		_, raw := connected(t)
		c, err := p.admit(raw)
		if err != nil {
			t.Fatal(err)
		}
		leases[i] = c
	}
	start := make(chan struct{})
	results := make(chan error, 32)
	for _, c := range leases {
		go func(c *propertyConnection) { <-start; results <- s.reserve(c) }(c)
	}
	close(start)
	success, limited := 0, 0
	for range leases {
		err := <-results
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrLimit):
			limited++
		default:
			t.Fatal("unexpected worker denial", err)
		}
	}
	if success != 1 || limited != 31 || len(s.active) != 1 {
		t.Fatal("last worker slot not exclusive", success, limited)
	}
	for _, c := range leases {
		s.finish(c)
	}
	if len(s.active) != 0 {
		t.Fatal("worker reservations retained")
	}
}
func TestPropertyServerFactoryValidation(t *testing.T) {
	valid := Config{Socket: "/var/run/horvium/helper.sock", ManagementUID: 42, SocketGID: 42, Limits: DefaultLimits(), Timeout: time.Second, MaxClients: 1}
	for _, name := range []string{"relative", "uid", "capacity", "timeout"} {
		t.Run(name, func(t *testing.T) {
			c := valid
			switch name {
			case "relative":
				c.Socket = "relative"
			case "uid":
				c.ManagementUID = 0
			case "capacity":
				c.MaxClients = 65
			case "timeout":
				c.Timeout = 0
			}
			if s, err := newPropertyServer(c); s != nil || !errors.Is(err, ErrConfig) {
				t.Fatal("invalid root server configuration accepted", err)
			}
		})
	}
	if !nativeSupported {
		if s, err := newPropertyServer(valid); s != nil || !errors.Is(err, ErrUnsupported) {
			t.Fatal("unsupported root listener authorized", err)
		}
	}
}

func TestPropertyWorkerCloseFailureReporting(t *testing.T) {
	closed := &net.OpError{Op: "close", Net: "unix", Err: net.ErrClosed}
	read := &net.OpError{Op: "read", Net: "unix", Err: io.ErrUnexpectedEOF}
	sentinel := errors.New("synthetic close failure for error classification only")
	closeError := &net.OpError{Op: "close", Net: "unix", Err: sentinel}
	for _, err := range []error{nil, ErrPermission, context.DeadlineExceeded, read, closed, errors.Join(read, closed)} {
		if propertyWorkerCloseFailure(err) != nil {
			t.Fatal("ordinary denial treated as cleanup failure")
		}
	}
	for _, err := range []error{closeError, errors.Join(read, closeError), fmt.Errorf("wrapped: %w", errors.Join(ErrProtocol, closeError))} {
		if !errors.Is(propertyWorkerCloseFailure(err), sentinel) {
			t.Fatal("first real close error discarded")
		}
	}
}
