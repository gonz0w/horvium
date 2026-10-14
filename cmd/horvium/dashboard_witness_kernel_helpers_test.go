// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"encoding/binary"
	"math"
)

type witnessKernelLayout struct {
	Size, PID, Parent, UID, Sec, Usec, RSS, Runtime int
	PageBytes                                       uint64
}
type witnessKernelProcess struct {
	PID, Parent int
	UID         uint32
	Sec, Usec   int64
	RSS         uint64
	CPU         int64
}

func (l witnessKernelLayout) valid() bool {
	if l.Size < 48 || l.Size > 4096 || l.PageBytes < 512 || l.PageBytes > 65536 || l.PageBytes&(l.PageBytes-1) != 0 {
		return false
	}
	fields := [][2]int{{l.PID, 4}, {l.Parent, 4}, {l.UID, 4}, {l.Sec, 8}, {l.Usec, 8}, {l.RSS, 8}, {l.Runtime, 8}}
	for i, f := range fields {
		if f[0] < 0 || f[0] > l.Size-f[1] {
			return false
		}
		for _, prior := range fields[:i] {
			if f[0] < prior[0]+prior[1] && prior[0] < f[0]+f[1] {
				return false
			}
		}
	}
	return true
}
func witnessDecodeKernelProcess(b []byte, l witnessKernelLayout) (witnessKernelProcess, error) {
	if !l.valid() || len(b) != l.Size {
		return witnessKernelProcess{}, errDashboardWitness
	}
	u32 := func(offset int) uint32 { return binary.LittleEndian.Uint32(b[offset:]) }
	u64 := func(offset int) uint64 { return binary.LittleEndian.Uint64(b[offset:]) }
	p := witnessKernelProcess{PID: int(int32(u32(l.PID))), Parent: int(int32(u32(l.Parent))), UID: u32(l.UID), Sec: int64(u64(l.Sec)), Usec: int64(u64(l.Usec))}
	pages, runtime := u64(l.RSS), u64(l.Runtime)
	if p.PID < 0 || p.Parent < 0 || p.Sec < 0 || p.Usec < 0 || p.Usec >= 1000000 || int64(pages) < 0 || pages > math.MaxUint64/l.PageBytes || runtime > math.MaxInt64/1000 {
		return witnessKernelProcess{}, errDashboardWitness
	}
	p.RSS, p.CPU = pages*l.PageBytes, int64(runtime*1000)
	return p, nil
}
