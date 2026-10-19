// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFixtureReservationFiniteSignalsAndPermanentSeal(t *testing.T) {
	calls := 0
	r := fixtureReservation{end: time.Now().Add(time.Second), positive: func(int) error { calls++; return nil }}
	for _, s := range []int{0, -1, 1, 15, 999} {
		if r.signal(s) == nil {
			t.Fatal("foreign signal admitted")
		}
	}
	for _, s := range []int{17, 19, 9} {
		if r.signal(s) != nil {
			t.Fatal("held signal denied")
		}
	}
	r.seal()
	r.seal()
	if r.signal(9) == nil || calls != 3 {
		t.Fatal("sealed authority revived", calls)
	}
}
func TestFixtureReservationExpiredNeverCallsPositive(t *testing.T) {
	r := fixtureReservation{end: time.Now().Add(-time.Nanosecond), positive: func(int) error { t.Fatal("expired positive call"); return nil }}
	if r.signal(9) == nil {
		t.Fatal("expiry admitted")
	}
	r.seal()
}
func TestFixtureReservationSealJoinsActualPositive(t *testing.T) {
	entered, release, sealed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	r := fixtureReservation{end: time.Now().Add(time.Second), positive: func(int) error { calls.Add(1); close(entered); <-release; return nil }}
	done := make(chan error, 1)
	go func() { done <- r.signal(9) }()
	<-entered
	go func() { r.seal(); close(sealed) }()
	select {
	case <-sealed:
		t.Fatal("seal crossed pending positive")
	case <-time.After(10 * time.Millisecond):
	}
	close(release)
	if <-done != nil {
		t.Fatal("actual signal result")
	}
	<-sealed
	if r.signal(9) == nil || calls.Load() != 1 {
		t.Fatal("post-seal signal")
	}
}

type fixtureFailedWrite struct{ short bool }

func (w fixtureFailedWrite) Write(b []byte) (int, error) {
	if w.short {
		return len(b) - 1, nil
	}
	return 0, io.ErrClosedPipe
}

type fixtureExpiredRead struct{}

func (fixtureExpiredRead) Read([]byte) (int, error) { return 0, os.ErrDeadlineExceeded }
func TestFixtureReservationBrokenHandshakeNeverReleasesStop(t *testing.T) {
	for _, tt := range []struct {
		name     string
		request  io.Reader
		response io.Writer
		pid      int
	}{
		{"EOF", strings.NewReader(""), io.Discard, 23},
		{"wrong-seal", strings.NewReader("H"), io.Discard, 23},
		{"expired", fixtureExpiredRead{}, io.Discard, 23},
		{"write-failed", strings.NewReader("S"), fixtureFailedWrite{}, 23},
		{"short-write", strings.NewReader("S"), fixtureFailedWrite{true}, 23},
		{"invalid-birth", strings.NewReader("S"), io.Discard, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			abandoned, stops := 0, 0
			e := fixturePublishBirth(tt.response, tt.pid, func() error { return nil }, func() { abandoned++ })
			if e == nil {
				e = fixtureStopAfterSeal(tt.request, func() error { stops++; return nil }, func() { abandoned++ })
			}
			if !errors.Is(e, ErrCleanupDebt) || abandoned != 1 || stops != 0 {
				t.Fatal("broken channel released original stop/Wait", e, abandoned, stops)
			}
		})
	}
}
func TestFixtureReservationBirthFrameAndSeal(t *testing.T) {
	var frame bytes.Buffer
	stops := 0
	sentinel := errors.New("actual stop result")
	e := fixturePublishBirth(&frame, 234, func() error { return nil }, func() { t.Fatal("valid original birth abandoned") })
	if e != nil {
		t.Fatal(e)
	}
	e = fixtureStopAfterSeal(strings.NewReader("S"), func() error { stops++; return sentinel }, func() { t.Fatal("valid original gate abandoned") })
	if !errors.Is(e, sentinel) || stops != 1 || frame.Len() != 8 {
		t.Fatal("actual stop join", e, stops)
	}
	var b [8]byte
	copy(b[:], frame.Bytes())
	if fixtureBirthPID(b) != 234 {
		t.Fatal("original birth frame")
	}
}
