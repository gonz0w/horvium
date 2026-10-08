// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

func (*PropertyStateClaim) PublishPropertyJobSuccessor(PropertyJob) (PropertyJob, error) {
	return PropertyJob{}, ErrUnavailable
}
