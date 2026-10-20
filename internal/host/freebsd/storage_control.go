// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

const controlOperationLimit = 90 * time.Second

type controlSelection interface {
	Targets() (storage.ControlTargets, error)
	Check(context.Context) error
	CheckFresh(context.Context) error
	Close() error
}
type controlOwnerAttempt struct {
	owner             *commissionedExecutionOwner
	entered, released bool
	err               error
}
type controlInspectionAttempt struct {
	state    string
	released bool
	err      error
}

// Private synchronous source transfers, never request/config fields. Production
// fixes the actual platform, UIDs, original selection and native owner producers.
// Tests of these transfers establish portable control flow only.
type storageControlSources struct {
	supported    bool
	root         func() bool
	load         func(context.Context, string) (controlSelection, error)
	fresh, start func(context.Context, storage.ControlTargets) controlOwnerAttempt
	inspect      func(context.Context, storage.ControlTargets) controlInspectionAttempt
}
type storageControlOutcome struct {
	exit                 int
	action, state, class string
	detail               error // Never formatted into output.
	deliverable          bool
	// Retain retired originals on unconfirmed release; no sink receives them.
	retainedSelection controlSelection
	retainedOwner     *commissionedExecutionOwner
}

func parseStorageControl(args []string) (string, string, error) {
	if len(args) != 3 || args[1] != "--config" || !storage.ValidControlPath(args[2]) {
		return "", "", storage.ErrInvalid
	}
	switch args[0] {
	case "commission-fresh", "start-closed", "inspect":
		return args[0], args[2], nil
	}
	return "", "", storage.ErrInvalid
}
func controlRefusal(action string, e error) storageControlOutcome {
	class := "unavailable"
	if errors.Is(e, storage.ErrInvalid) {
		class = "invalid-input"
	} else if errors.Is(e, ErrPermission) {
		class = "not-root"
	} else if errors.Is(e, storage.ErrStateLocked) {
		class = "busy"
	}
	return storageControlOutcome{exit: 2, action: action, class: class, detail: e, deliverable: true}
}
func controlSelectedFailure(action string, e error) storageControlOutcome {
	if errors.Is(e, storage.ErrDrift) || errors.Is(e, storage.ErrStateUncertain) || errors.Is(e, storage.ErrControlRelease) || errors.Is(e, ErrPermission) {
		return controlRecovery(action, e)
	}
	return controlRefusal(action, e)
}
func controlRecovery(action string, e error) storageControlOutcome {
	return storageControlOutcome{exit: 3, action: action, class: "recovery-required", detail: e, deliverable: true}
}

// executeStorageControl assembles one complete lifetime, retaining all output
// privately. It never registers a command or invokes a writer. No retry exists.
func executeStorageControl(ctx context.Context, args []string, s storageControlSources) storageControlOutcome {
	action, path, e := parseStorageControl(args)
	if e != nil {
		return controlRefusal("", e)
	}
	if ctx == nil || !s.supported || s.root == nil || s.load == nil || s.fresh == nil || s.start == nil || s.inspect == nil {
		return controlRefusal(action, ErrUnsupported)
	}
	if e = ctx.Err(); e != nil {
		return controlRefusal(action, e)
	}
	if !s.root() {
		return controlRefusal(action, ErrPermission)
	}
	op, cancel := context.WithTimeout(ctx, controlOperationLimit)
	defer cancel()
	selection, e := s.load(op, path)
	result := controlRefusal(action, e)
	if selection == nil {
		if e == nil {
			result = controlRefusal(action, ErrUnavailable)
		}
		return result
	}
	// Foreground cleanup replaces this with its single fresh budget. Keep it
	// alive through release and the final original selection observation.
	releaseContext := op
	admitted := e == nil
	guard := func(c context.Context) error {
		if e := c.Err(); e != nil {
			return e
		}
		if !s.root() {
			return ErrPermission
		}
		e := selection.Check(c)
		if !s.root() {
			e = errors.Join(e, ErrPermission)
		}
		return errors.Join(e, c.Err())
	}
	// Close is always attempted, even failed admission, lost root or custody drift.
	closeSelection := func(out storageControlOutcome) storageControlOutcome {
		var finalErr error
		if admitted {
			finalErr = guard(releaseContext)
		}
		rooted := s.root()
		ce := selection.Close()
		rootAfter := s.root()
		if !rooted || !rootAfter {
			ce = errors.Join(ce, ErrPermission)
		}
		if finalErr != nil || ce != nil {
			retained := out.retainedOwner
			wasDeliverable := out.deliverable
			detail := errors.Join(out.detail, finalErr, ce)
			if out.exit == 2 && ce == nil {
				out = controlSelectedFailure(action, detail)
			} else {
				out = controlRecovery(action, detail)
			}
			out.retainedOwner = retained
			out.deliverable = out.deliverable && wasDeliverable
		}
		if ce != nil {
			out.deliverable = false
			out.retainedSelection = selection
		}
		if errors.Is(out.detail, storage.ErrControlRelease) {
			out.deliverable = false
			out.retainedSelection = selection
		}
		return out
	}
	if e != nil {
		if !s.root() {
			e = errors.Join(e, ErrPermission)
		}
		return closeSelection(controlSelectedFailure(action, e))
	}
	if e = guard(op); e != nil {
		return closeSelection(controlSelectedFailure(action, e))
	}
	targets, e := selection.Targets()
	if e != nil {
		return closeSelection(controlSelectedFailure(action, e))
	}
	if action == "inspect" {
		inspected := s.inspect(op, targets)
		e = errors.Join(inspected.err, guard(op))
		result = controlRefusal(action, e)
		if e == nil {
			switch inspected.state {
			case "closed":
				result = storageControlOutcome{action: action, state: "closed", deliverable: true}
			case "recovery-required":
				result = storageControlOutcome{exit: 3, action: action, state: "recovery-required", deliverable: true}
			default:
				result = controlRecovery(action, storage.ErrStateUncertain)
			}
		} else if errors.Is(e, storage.ErrDrift) || errors.Is(e, storage.ErrStateUncertain) || errors.Is(e, ErrPermission) {
			result = controlRecovery(action, e)
		}
		if !inspected.released {
			result = controlRecovery(action, e)
			result.deliverable = false
		}
		return closeSelection(result)
	}
	if action == "commission-fresh" {
		if e = selection.CheckFresh(op); e != nil {
			return closeSelection(controlSelectedFailure(action, e))
		}
	}
	// Final actual UID/custody/context guard precedes original registry admission.
	if e = guard(op); e != nil {
		return closeSelection(controlSelectedFailure(action, e))
	}
	var attempt controlOwnerAttempt
	if action == "commission-fresh" {
		attempt = s.fresh(op, targets)
	} else {
		attempt = s.start(op, targets)
	}
	o := attempt.owner
	e = attempt.err
	if o == nil {
		if e == nil {
			e = ErrUnavailable
		}
		result = controlRefusal(action, e)
		if attempt.entered || errors.Is(e, storage.ErrDrift) || errors.Is(e, storage.ErrStateUncertain) {
			result = controlRecovery(action, e)
		}
		if !attempt.released {
			result = controlRecovery(action, e)
			result.deliverable = false
		}
		return closeSelection(result)
	}
	// A returned original active owner has entered a state-changing lifetime.
	e = errors.Join(e, guard(op))
	if e == nil {
		_, e = o.Observe(op)
		e = errors.Join(e, guard(op))
	}
	if e == nil && action == "start-closed" {
		// No startup output, poll, worker or dispatcher. The first cancellation is
		// latched by this single wait and cannot renew the one fresh cleanup budget.
		cancel()
		<-ctx.Done()
		cleanup, stop := context.WithTimeout(context.Background(), controlOperationLimit)
		releaseContext = cleanup
		defer stop()
		if e = guard(cleanup); e == nil {
			e = o.FinishIdle(cleanup)
			e = errors.Join(e, guard(cleanup))
		}
	} else if e == nil {
		if e = guard(op); e == nil {
			e = o.FinishIdle(op)
			e = errors.Join(e, guard(op))
		}
	}
	// Finish is never repeated; always attempt original release, including errors.
	ce := o.Close()
	e = errors.Join(e, ce)
	result = storageControlOutcome{action: action, state: "closed", deliverable: true}
	if e != nil {
		result = controlRecovery(action, e)
	}
	if ce != nil {
		result.deliverable = false
		result.retainedOwner = o
	}
	return closeSelection(result)
}

// deliverStorageControl is reached only after confirmed original release. A
// short/failing write is terminal and never retries the structured result.
func deliverStorageControl(r storageControlOutcome, stdout, stderr io.Writer) int {
	if !r.deliverable {
		return r.exit
	}
	if r.state != "" {
		if _, _, e := parseStorageControl([]string{r.action, "--config", "/valid/config"}); e != nil {
			return 3
		}
		if r.state != "closed" && r.state != "recovery-required" {
			return 3
		}
		data := []byte("{\"format\":\"horvium-storage-control-result\",\"version\":1,\"action\":\"" + r.action + "\",\"state\":\"" + r.state + "\",\"writes_enabled\":false}\n")
		if len(data) > 4096 || stdout == nil {
			return 3
		}
		n, e := stdout.Write(data)
		if e == nil && n == len(data) {
			return r.exit
		}
		if stderr != nil {
			_, _ = io.WriteString(stderr, "recovery-required\n")
		}
		return 3
	}
	switch r.class {
	case "invalid-input", "not-root", "busy", "unavailable", "recovery-required":
	default:
		return 3
	}
	if stderr == nil {
		return 3
	}
	data := r.class + "\n"
	n, e := io.WriteString(stderr, data)
	if e != nil || n != len(data) {
		return 3
	}
	return r.exit
}

// Deliberately unregistered: cmd/horvium, helper and management dispatch never
// call this consumer until separately admitted native qualification completes.
func runStorageControl(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if ctx == nil {
		return deliverStorageControl(controlRefusal("", ErrUnavailable), stdout, stderr)
	}
	foreground, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	outcome := executeStorageControl(foreground, args, nativeStorageControlSources())
	return deliverStorageControl(outcome, stdout, stderr)
}

func nativeStorageControlSources() storageControlSources {
	return storageControlSources{supported: nativeSupported, root: executionRoot,
		load: func(ctx context.Context, path string) (controlSelection, error) {
			s, e := storage.OpenControlSelection(ctx, path)
			if s == nil {
				return nil, e
			}
			return s, e
		},
		fresh: func(ctx context.Context, t storage.ControlTargets) controlOwnerAttempt {
			return nativeControlOwner(ctx, t, "commission-fresh")
		},
		start: func(ctx context.Context, t storage.ControlTargets) controlOwnerAttempt {
			return nativeControlOwner(ctx, t, "start-closed")
		},
		inspect: nativeControlInspect,
	}
}
