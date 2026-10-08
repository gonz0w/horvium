// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"sync"
)

// Private prototype: socket/worker ownership only. Every valid property operation
// is denied; no backend, collector, service or runtime consumer is registered.
type propertyServer struct {
	mu        sync.Mutex
	identity  *propertyServer
	config    Config
	attempted bool
	active    map[*propertyConnection]struct{}
}

func newPropertyServer(c Config) (*propertyServer, error)     { return nil, ErrPermission }
func (s *propertyServer) serve(ctx context.Context) error     { return ErrPermission }
func (s *propertyServer) reserve(c *propertyConnection) error { return ErrPermission }
func (s *propertyServer) finish(c *propertyConnection)        {}
