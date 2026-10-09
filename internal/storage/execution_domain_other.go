// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import "context"

func (*StateDirectory) ObserveExecutionDomain(context.Context, func(context.Context, int) (ExecutionBackingObservation, error)) (ExecutionDomainObservation, error) {
	return ExecutionDomainObservation{}, ErrUnavailable
}
