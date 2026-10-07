// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
)

const maxStorageStatusBytes = 1 << 20
const maxStorageStatusJobs = 1064 // Accepted 64 nonterminal + 1,000 terminal store bounds.

// This diagnostic envelope is not an executable job, transport capability or
// verification of current native effects. Never expose a full stored intent.
type storageJobStatus struct {
	Version          int                   `json:"version"`
	Kind             string                `json:"kind"`
	Source           string                `json:"source"`
	RecoveryVerified bool                  `json:"recovery_verified"`
	HoldsRetained    bool                  `json:"holds_retained"`
	Jobs             []storageJobStatusRow `json:"jobs"`
}
type storageJobStatusRow struct {
	ID            string                `json:"id"`
	SubmissionID  string                `json:"submission_id"`
	PoolGUID      string                `json:"pool_guid"`
	LastDurable   storageJobStatusEvent `json:"last_durable"`
	RestartState  storage.JobState      `json:"restart_state"`
	RestartReason string                `json:"restart_reason"`
}
type storageJobStatusEvent struct {
	Revision          uint64           `json:"revision"`
	At                uint64           `json:"at"`
	State             storage.JobState `json:"state"`
	Reason            string           `json:"reason"`
	ObservationDigest string           `json:"observation_digest"`
}

func runStorageJobs(ctx context.Context, path string, out io.Writer) error {
	if ctx == nil || out == nil {
		return errInput
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	// Actual FreeBSD effective UID/ancestor/ACL/lock checks stay in the existing
	// root lease opener. No store is provisioned, repaired or held after this call.
	state, err := storage.OpenStateDirectory(path)
	if err != nil {
		return err
	}
	defer state.Close()
	if err = ctx.Err(); err != nil {
		return err
	}
	view, err := state.InspectJobReservations()
	if err != nil {
		return err
	}
	// Finish all native reads and release the lease before any status output.
	if err = state.Close(); err != nil {
		return err
	}
	return writeStorageJobStatus(ctx, view, out)
}

func writeStorageJobStatus(ctx context.Context, view storage.JobReservations, out io.Writer) error {
	if ctx == nil || out == nil {
		return errInput
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	result := storageJobStatus{Version: 1, Kind: "storage-job-status", Source: "retained-metadata", HoldsRetained: true, Jobs: make([]storageJobStatusRow, 0)}
	after := ""
	for {
		page, err := view.RestartPage(after, 64)
		if err != nil {
			return err
		}
		if len(result.Jobs)+len(page) > maxStorageStatusJobs {
			return storage.ErrLimit
		}
		for _, j := range page {
			last := j.Last
			result.Jobs = append(result.Jobs, storageJobStatusRow{ID: j.ID, SubmissionID: j.SubmissionID, PoolGUID: j.PoolGUID, LastDurable: storageJobStatusEvent{Revision: last.Revision, At: last.At, State: last.State, Reason: last.Reason, ObservationDigest: last.ObservationDigest}, RestartState: j.State, RestartReason: j.Reason})
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(page) == 0 {
			break
		}
		after = page[len(page)-1].ID
	}
	// Stage the complete bounded projection. Validation/cancellation failures never
	// expose a partial page; writer failure cannot undo already delivered bytes.
	data, err := json.Marshal(result)
	if err != nil {
		return errInput
	}
	if len(data)+1 > maxStorageStatusBytes {
		return storage.ErrLimit
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	data = append(data, '\n')
	n, err := out.Write(data)
	if err != nil || n != len(data) {
		return errors.New("output unavailable")
	}
	return nil
}
