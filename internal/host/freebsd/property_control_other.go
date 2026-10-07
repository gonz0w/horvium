// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package freebsd

import (
	"context"
	"github.com/gonz0w/horvium/internal/storage"
)

func propertySetNative(context.Context, storage.PropertyIntent) error { return ErrUnsupported }
