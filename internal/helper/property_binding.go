// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"errors"
	"sync"
)

// Private original-request provenance, never human/session, policy or launch
// permission. Only a received request on an original admitted root peer issues it.
type propertyRequestBinding struct {
	identity *propertyRequestBinding
	owner    *propertyRequestOwner
	peer     *admittedPropertyPeer
	request  propertyRequest
	actor    actionIdentity
}
type propertyRequestOwner struct {
	mu      sync.Mutex
	peer    *admittedPropertyPeer
	binding *propertyRequestBinding
	retired bool
	close   sync.Once
	failure error
}

func receivePropertyBinding(a *admittedPropertyPeer) (*propertyRequestBinding, error) {
	return nil, errors.Join(ErrPermission, a.Close())
}
func (r *propertyRequestBinding) check(ctx context.Context) error { return ErrPermission }
func (r *propertyRequestBinding) snapshot(ctx context.Context) (propertyRequest, actionIdentity, error) {
	return propertyRequest{}, actionIdentity{}, ErrPermission
}
func (r *propertyRequestBinding) Close() error { return nil }
