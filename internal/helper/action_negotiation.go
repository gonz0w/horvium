// SPDX-License-Identifier: BSD-3-Clause
package helper

import "io"

// The version-2 hello is distinct from the existing version-1 inventory request.
// These codecs are not wired into Server or Client. Context is an untrusted
// service assertion at the helper boundary, never a credential or kernel UID.
type actionHello struct {
	Version   int    `json:"version"`
	ID        string `json:"request_id"`
	Operation string `json:"operation"`
	Context   string `json:"session_context"`
}
type actionReady struct {
	Version int    `json:"version"`
	ID      string `json:"request_id"`
	Type    string `json:"type"`
	Context string `json:"session_context"`
}

func negotiationLimits() Limits { return Limits{512, 2, 64, 2} }

func validApplicationContext(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func validActionHello(h actionHello) bool {
	return h.Version == 2 && h.Operation == "negotiate" && validID(h.ID) && validApplicationContext(h.Context)
}

// WriteActionNegotiation writes correlation data only. The caller must supply
// the original live application context and independently authenticate the root
// peer before sending. This function cannot attest either identity or permission.
func WriteActionNegotiation(w io.Writer, id, applicationContext string) error {
	h := actionHello{2, id, "negotiate", applicationContext}
	if w == nil || !validActionHello(h) {
		return ErrProtocol
	}
	b, _ := newBudget(negotiationLimits())
	return b.write(w, h)
}

// Returns only parsed assertions. Native peer authorization must precede this
// read when integrated; parsing a forged but canonical context proves nothing.
func readActionNegotiation(r io.Reader) (actionHello, error) {
	if r == nil {
		return actionHello{}, ErrProtocol
	}
	l := negotiationLimits()
	b, _ := newBudget(l)
	data, err := b.read(r)
	if err != nil {
		return actionHello{}, err
	}
	var h actionHello
	if err := decode(data, &h, l); err != nil {
		return actionHello{}, err
	}
	if !validActionHello(h) {
		return actionHello{}, ErrProtocol
	}
	return h, nil
}

// A ready frame acknowledges only version/context correlation. It advertises
// no mutation operation and supplies no policy, recovery or launch capability.
func writeActionReady(w io.Writer, h actionHello) error {
	if w == nil || !validActionHello(h) {
		return ErrProtocol
	}
	b, _ := newBudget(negotiationLimits())
	return b.write(w, actionReady{2, h.ID, "negotiated", h.Context})
}

// ReadActionNegotiation consumes one bounded ready frame and requires exact
// request/context correlation, without fallback. It leaves later frames for the
// eventual typed exchange; no EOF, authorization or mutation result is implied.
func ReadActionNegotiation(r io.Reader, id, applicationContext string) error {
	if r == nil || !validID(id) || !validApplicationContext(applicationContext) {
		return ErrProtocol
	}
	l := negotiationLimits()
	b, _ := newBudget(l)
	data, err := b.read(r)
	if err != nil {
		return err
	}
	var ready actionReady
	if err := decode(data, &ready, l); err != nil {
		return err
	}
	if ready.Version != 2 || ready.Type != "negotiated" || ready.ID != id || ready.Context != applicationContext {
		return ErrProtocol
	}
	return nil
}
