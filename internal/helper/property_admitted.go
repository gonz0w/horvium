// SPDX-License-Identifier: BSD-3-Clause
package helper

import "github.com/gonz0w/horvium/internal/storage"

// Original admission lease plus one original kernel-authenticated exchange.
// Private and unregistered; no storage/session or worker quiescence authority.
type admittedPropertyPeer struct {
	identity *admittedPropertyPeer
	lease    *propertyConnection
	exchange *propertyPeer
}

func (c *propertyConnection) accept(uint32) (*admittedPropertyPeer, error) {
	_ = c.Close()
	return nil, ErrPermission
}
func (a *admittedPropertyPeer) receive() (propertyRequest, actionIdentity, error) {
	return propertyRequest{}, actionIdentity{}, ErrPermission
}
func (a *admittedPropertyPeer) replyReview(storage.Review) error        { return ErrPermission }
func (a *admittedPropertyPeer) replySubmission(storage.Job, bool) error { return ErrPermission }
func (a *admittedPropertyPeer) refuse(string) error                     { return ErrPermission }
func (a *admittedPropertyPeer) Close() error                            { return nil }
