// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

func (*PropertyStateClaim) VerifyPropertyApplyingAcknowledgement(PropertyJob) error {
	return ErrUnavailable
}
