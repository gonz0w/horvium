// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"fmt"
	"io"
)

func witnessCopyCapture(target io.Writer, source io.Reader, remaining int64) (int64, string, error) {
	// Storage, like forwarding, may acknowledge bytes and return an error in
	// the same Write. Keep that exact prefix's hash; MultiWriter drops it.
	sink := newWitnessForwardWriter(target)
	n, e := io.CopyBuffer(sink, io.LimitReader(source, remaining), make([]byte, witnessFrameCap))
	return n, fmt.Sprintf("%x", sink.hash.Sum(nil)), e
}
