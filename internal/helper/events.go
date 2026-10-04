// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/gonz0w/horvium/internal/eventlog"
)

func (s *Server) failure(section, outcome, class string, started time.Time) {
	if s.events == nil {
		return
	}
	s.events.Record(eventlog.Event{Operation: "helper", ID: s.events.NextID(), Section: section, Outcome: outcome, ErrorClass: class, Duration: time.Since(started)})
}
func boundaryClass(ctx context.Context, err error, deadline time.Time) string {
	if errors.Is(ctx.Err(), context.Canceled) {
		return "cancelled"
	}
	// Framing deliberately hides underlying read errors. Socket expiry can precede
	// the context timer, so also compare the owned deadline before classification.
	var network net.Error
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || !time.Now().Before(deadline) || errors.As(err, &network) && network.Timeout() {
		return "timeout"
	}
	if errors.Is(err, ErrLimit) {
		return "limit"
	}
	return "protocol"
}
