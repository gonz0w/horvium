// SPDX-License-Identifier: BSD-3-Clause
//go:build !freebsd

package storage

func (*StateDirectory) PublishPropertySubmission(PropertyJob) (PropertyJob, bool, error) {
	return PropertyJob{}, false, ErrUnavailable
}
