//go:build !freebsd

// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestStoragePropertyAssessmentUnsupportedHost(t *testing.T) {
	var out, errout bytes.Buffer
	code := run(context.Background(), []string{"storage-assess", "-state-dir", "/root/fixture-ledger", "-job-id", strings.Repeat("a", 64)}, statusForbiddenReader{t}, &out, &errout)
	if code != 1 || out.Len() != 0 || errout.String() != errStorageAssessment.Error()+"\n" {
		t.Fatal("unsupported host exposed result", code, errout.String())
	}
}
