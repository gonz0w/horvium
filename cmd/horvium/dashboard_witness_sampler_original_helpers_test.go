// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"io"
	"os"
)

const witnessSamplerBytes = 32 << 20

func witnessSamplerReadOriginal(f *os.File) ([]byte, error) {
	before, e := f.Stat()
	if e != nil || before.Size() <= 0 || before.Size() > witnessSamplerBytes {
		return nil, errDashboardWitness
	}
	if _, e := f.Seek(0, 0); e != nil {
		return nil, errDashboardWitness
	}
	data, e := io.ReadAll(io.LimitReader(f, witnessSamplerBytes+1))
	after, ae := f.Stat()
	if e != nil || ae != nil || !witnessRawUnchanged(before, after) || int64(len(data)) != before.Size() {
		return nil, errDashboardWitness
	}
	return data, nil
}
