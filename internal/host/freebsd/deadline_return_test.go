// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"testing"
	"time"
)

type deadlineReaderError struct{ cause error }

func (*deadlineReaderError) Error() string   { return "private synthetic reader detail" }
func (e *deadlineReaderError) Unwrap() error { return e.cause }

type deadlineCollectorCase struct {
	name, command string
	collect       func(context.Context) error
}

// Existing deterministic producer seams only; no native commands or state.
func deadlineCollectorCases(t *testing.T, read func(context.Context) error) []deadlineCollectorCase {
	t.Helper()
	backing, _ := persistentFixture("ufs")
	backing.commandTimeout, backing.totalTimeout = 10*time.Second, 45*time.Second
	backing.read = func(c context.Context, _ string, _ string, _ int) (readOutput, error) { return readOutput{}, read(c) }
	pool := propertySource(func(c context.Context, _ poolProbe, _ string, _ int) (readOutput, error) {
		return readOutput{}, read(c)
	})
	pool.commandTimeout, pool.totalTimeout = 10*time.Second, 45*time.Second
	system := useSource(func(c context.Context, _ systemProbe, _ string, _ int) (readOutput, error) {
		return readOutput{}, read(c)
	})
	system.commandTimeout, system.totalTimeout = 10*time.Second, 45*time.Second
	providers := providersForTest(func(c context.Context, _ systemProbe, _ string, _ int) (readOutput, error) {
		return readOutput{}, read(c)
	})
	providers.source.commandTimeout, providers.source.totalTimeout = 10*time.Second, 45*time.Second
	o := providerObservation(t)
	return []deadlineCollectorCase{
		{"backing-collector", "backing-command", func(c context.Context) error { _, e := backing.collect(c); return e }},
		{"pool-collector", "pool-command", func(c context.Context) error { _, e := pool.Collect(c, "100", "tank"); return e }},
		{"system-collector", "system-command", func(c context.Context) error { _, e := system.Collect(c); return e }},
		{"provider-collector", "system-command", func(c context.Context) error { _, e := providers.Collect(c, o); return e }},
	}
}

func TestDeadlineReturnDisabledCollectorIdentity(t *testing.T) {
	for _, original := range []error{&deadlineReaderError{context.DeadlineExceeded}, context.Canceled, errors.New("ordinary failure")} {
		calls := 0
		for _, tc := range deadlineCollectorCases(t, func(context.Context) error { calls++; return original }) {
			t.Run(tc.name+"/"+original.Error(), func(t *testing.T) {
				before := calls
				if got := tc.collect(context.Background()); got != original || calls != before+1 {
					t.Fatal("disabled adapter altered error identity or original call count")
				}
			})
		}
	}
}
