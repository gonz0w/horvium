// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import "context"

type controlSelectionNative struct{}

func OpenControlSelection(context.Context, string) (*ControlSelection, error) {
	return nil, ErrUnavailable
}
func (*ControlSelection) Check(context.Context) error      { return ErrUnavailable }
func (*ControlSelection) CheckFresh(context.Context) error { return ErrUnavailable }
func (*ControlSelection) Close() error                     { return ErrUnavailable }
