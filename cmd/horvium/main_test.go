// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestNoArgumentCredentialDisclosure(t *testing.T) {
	for _, args := range [][]string{{"unknown-secret"}, {"inventory", "-token", "secret"}, {"serve", "-config", "relative"}} {
		var out, errout bytes.Buffer
		if run(context.Background(), args, strings.NewReader(""), &out, &errout) == 0 {
			t.Fatal("invalid command accepted")
		}
		if strings.Contains(errout.String(), "secret") || strings.Contains(out.String(), "secret") {
			t.Fatal("argument disclosed")
		}
	}
}
func TestTokenInputBoundaries(t *testing.T) {
	for _, s := range []string{"", strings.Repeat("a", 63), strings.Repeat("A", 64), strings.Repeat("a", 65), strings.Repeat("a", 64) + "\nextra"} {
		if _, e := readToken(strings.NewReader(s)); e == nil {
			t.Fatal("malformed input accepted")
		}
	}
	for _, s := range []string{strings.Repeat("a", 64), strings.Repeat("a", 64) + "\n"} {
		if _, e := readToken(strings.NewReader(s)); e != nil {
			t.Fatal("protected token rejected")
		}
	}
}
