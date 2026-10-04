//go:build ignore

// SPDX-License-Identifier: BSD-3-Clause
// Disposable research probe; not a production inventory adapter.
package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
)

type provider struct {
	Name       string `xml:"name"`
	MediaSize  uint64 `xml:"mediasize"`
	SectorSize uint64 `xml:"sectorsize"`
}

type mesh struct {
	Providers []provider `xml:"class>geom>provider"`
}

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 8<<20 {
		return 0, fmt.Errorf("GEOM fixture output exceeds 8 MiB")
	}
	return b.Buffer.Write(p)
}

func main() {
	if len(os.Args) != 2 {
		panic("expected one generated fixture device")
	}
	var output boundedOutput
	cmd := exec.Command("/sbin/sysctl", "-b", "kern.geom.confxml")
	cmd.Stdout = &output
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	var inventory mesh
	if err := xml.Unmarshal(output.Bytes(), &inventory); err != nil {
		panic(err)
	}
	for _, p := range inventory.Providers {
		if p.Name == os.Args[1] {
			if p.MediaSize != 512<<20 || p.SectorSize == 0 {
				panic("unexpected fixture geometry")
			}
			fmt.Println("PASS: GEOM XML decoded by Go; fixture size and sector size validated")
			return
		}
	}
	panic("fixture provider absent")
}
