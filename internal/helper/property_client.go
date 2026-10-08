// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
	"net"
	"time"
)

// PropertyClient owns one application-side exchange. It is not root-side session,
// policy, durable admission or launch authority; there is no runtime consumer.
type PropertyClient struct {
	identity *PropertyClient
	exchange *propertyPeer
}

// ConnectPropertyClient transfers the original socket, including on failure.
// check is a trusted application callback, never a replacement kernel UID hook.
func ConnectPropertyClient(ctx context.Context, c *net.UnixConn, uid uint32, id, scope string, timeout time.Duration, check func(context.Context) error) (*PropertyClient, error) {
	if c != nil {
		_ = c.Close()
	}
	return nil, ErrPermission
}
func (c *PropertyClient) Review(pool string, intent storage.Intent) (storage.Review, error) {
	return storage.Review{}, ErrPermission
}
func (c *PropertyClient) Submit(pool, plan, submission string) (storage.Job, bool, error) {
	return storage.Job{}, false, ErrPermission
}
func (c *PropertyClient) Close() error { return nil }
