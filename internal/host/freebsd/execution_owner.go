// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"context"
	"sync"

	"github.com/gonz0w/horvium/internal/storage"
)

// One commissioning attempt for the root helper lifetime. Never reset or copy.
type executionOwnerRegistry struct {
	mu       sync.Mutex
	original *executionOwnerRegistry
	claimed  bool
	owner    *commissionedExecutionOwner
}

func newExecutionOwnerRegistry() *executionOwnerRegistry {
	r := &executionOwnerRegistry{}
	r.original = r
	return r
}

var nativeExecutionOwners = newExecutionOwnerRegistry()

type executionOwnerSource struct {
	observe func(context.Context) (storage.ExecutionDomainsObservation, error)
	finish  func(context.Context) error
	release func() error
}
type commissionedExecutionOwner struct {
	mu              sync.Mutex
	original        *commissionedExecutionOwner
	registry        *executionOwnerRegistry
	source          executionOwnerSource
	root            func() bool
	blocked, closed bool
	closeErr        error
}

// Private synchronous seams only. Production fixes the process registry, actual
// root check and original native activation. No caller-facing factory is added.
func commissionExecutionOwner(context.Context, *executionOwnerRegistry, func() bool, func(context.Context) (executionOwnerSource, error)) (*commissionedExecutionOwner, error) {
	return nil, storage.ErrDenied
}
func (o *commissionedExecutionOwner) Observe(context.Context) (storage.ExecutionDomainsObservation, error) {
	return storage.ExecutionDomainsObservation{}, storage.ErrDenied
}
func (o *commissionedExecutionOwner) FinishIdle(context.Context) error { return storage.ErrDenied }
func (o *commissionedExecutionOwner) Close() error                     { return storage.ErrDenied }
