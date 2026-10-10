// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"context"
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

// Exclusive namespace provenance only. This does not initialize a journal,
// establish boot/restore provenance or register a runtime/global writer.
type freshExecutionDomains struct {
	mu              sync.Mutex
	original        *freshExecutionDomains
	pair            *ExecutionDomainLeases
	anchors         [2]freshExecutionAnchor
	closed, blocked bool
}
type freshExecutionAnchor struct {
	parent                          *os.File
	path, name                      string
	parentIdentity, directory, lock unix.Stat_t
}

func provisionFreshExecutionDomains(ctx context.Context, journal, ledger string, step func(string, string) error) (*freshExecutionDomains, error) {
	return nil, ErrDenied
}
func (f *freshExecutionDomains) inspect(ctx context.Context) error { return ErrDenied }
func (f *freshExecutionDomains) close() error                      { return ErrDenied }
