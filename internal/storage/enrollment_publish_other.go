// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import (
	"context"
	"github.com/gonz0w/horvium/internal/inventory"
	"time"
)

func (PropertyEnrollmentState) publishEnrollmentMetadata(context.Context, string, Observation, []inventory.DiskObservation, PoolSystemUse, func() time.Time, stateCatalogLimits, func(ownershipPublication) ownershipPublication) error {
	return ErrUnavailable
}
