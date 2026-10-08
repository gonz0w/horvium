// SPDX-License-Identifier: BSD-3-Clause
package helper

import (
	"context"
	"errors"
	"net"
	"os"
)

// DialPropertyClient owns one configured application connection. Configuration
// is trusted; application checks supplement actual kernel UID and socket policy.
// This is unregistered and grants no root session/backend or launch authority.
func DialPropertyClient(parent context.Context, c Config, id, scope string, check func(context.Context) error) (_ *PropertyClient, err error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	if parent == nil || check == nil || !validID(id) || !validApplicationContext(scope) {
		return nil, ErrConfig
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	if !nativeSupported {
		return nil, ErrUnsupported
	}
	if uint32(os.Getuid()) != c.ManagementUID || uint32(os.Geteuid()) != c.ManagementUID {
		return nil, ErrPermission
	}
	ctx, cancel := context.WithTimeout(parent, c.Timeout)
	var raw *net.UnixConn
	defer func() {
		if err != nil {
			cancel()
			err = errors.Join(err, closePropertySocket(raw))
		}
	}()
	if err = check(ctx); err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if err = safeSocket(c); err != nil {
		return nil, err
	}
	original, e := os.Lstat(c.Socket)
	if e != nil || original.Mode()&os.ModeSocket == 0 {
		return nil, ErrConfig
	}
	var dialer net.Dialer
	connection, e := dialer.DialContext(ctx, "unix", c.Socket)
	if e != nil {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		return nil, ErrProtocol
	}
	var ok bool
	raw, ok = connection.(*net.UnixConn)
	if !ok {
		return nil, errors.Join(ErrProtocol, connection.Close())
	}
	if err = check(ctx); err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if err = safeSocket(c); err != nil {
		return nil, err
	}
	current, e := os.Lstat(c.Socket)
	if e != nil || current.Mode()&os.ModeSocket == 0 || !os.SameFile(original, current) {
		return nil, ErrConfig
	}
	client, e := ConnectPropertyClient(ctx, raw, c.ManagementUID, id, scope, c.Timeout, check)
	if e != nil {
		return nil, e
	}
	client.cancel = cancel
	return client, nil
}
