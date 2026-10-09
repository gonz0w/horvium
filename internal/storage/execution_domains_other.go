// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

import "context"

type executionDomainLeaseNative struct{}

func OpenExecutionDomainLeases(context.Context, string, string) (*ExecutionDomainLeases, error) {
	return nil, ErrUnavailable
}
func (*ExecutionDomainLeases) Observe(context.Context, string, func(context.Context, int) (ExecutionBackingObservation, error)) (ExecutionDomainsObservation, error) {
	return ExecutionDomainsObservation{}, ErrUnavailable
}
func (p *ExecutionDomainLeases) Close() error {
	if p == nil {
		return nil
	}
	return ErrUnavailable
}
