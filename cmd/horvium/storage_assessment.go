// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/gonz0w/horvium/internal/host/freebsd"
	"github.com/gonz0w/horvium/internal/storage"
)

// The native source retains its 135-second bound. This foreground process also
// bounds lease handling and blocked output; it grants no recovery authority.
const storageAssessmentTimeout = 145 * time.Second
const maxStorageAssessmentBytes = 4 << 10

var errStorageAssessment = errors.New("storage assessment unavailable")

type storagePropertyAssessmentStatus struct {
	Version               int                 `json:"version"`
	Kind                  string              `json:"kind"`
	Source                string              `json:"source"`
	RecoveryVerified      bool                `json:"recovery_verified"`
	ExecutorVerified      bool                `json:"executor_verified"`
	HoldsRetained         bool                `json:"holds_retained"`
	BaselineMatches       bool                `json:"baseline_matches"`
	ExpectedEffectMatches bool                `json:"expected_effect_matches"`
	Job                   storageJobStatusRow `json:"job"`
}

func runStoragePropertyAssessment(ctx context.Context, path, id string, out io.Writer) error {
	if ctx == nil || out == nil {
		return errInput
	}
	if err := storage.ValidatePropertyJobSelection(id); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, storageAssessmentTimeout)
	defer cancel()
	state, err := storage.OpenStateDirectory(path)
	if err != nil {
		return err
	}
	defer state.Close()
	source, err := freebsd.NewPropertyJobSource(state)
	if err != nil {
		return err
	}
	assessment, err := source.Assess(ctx, id)
	if err != nil {
		return err
	}
	// Own no ledger descriptors while stdout is slow or unavailable. Opening may
	// create/sync only the existing contract's empty lock; records are never written.
	if err = state.Close(); err != nil {
		return err
	}
	return writeStoragePropertyAssessment(ctx, id, assessment, out)
}

// Only the independently collected source feeds this private projection in
// production. It exposes no full job intent, baseline, policy or native rows.
func writeStoragePropertyAssessment(ctx context.Context, id string, a storage.PropertyJobAssessment, out io.Writer) error {
	if ctx == nil || out == nil {
		return errInput
	}
	if err := storage.ValidatePropertyJobSelection(id); err != nil {
		return err
	}
	if a.Restart.ID != id || a.Restart.Last.Revision == 0 {
		return errInput
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	j := a.Restart
	last := j.Last
	result := storagePropertyAssessmentStatus{
		Version: 1, Kind: "storage-property-assessment", Source: "retained-metadata-and-native-properties",
		HoldsRetained: true, BaselineMatches: a.BaselineMatches, ExpectedEffectMatches: a.PostconditionMatches,
		Job: storageJobStatusRow{
			ID: j.ID, SubmissionID: j.SubmissionID, PoolGUID: j.PoolGUID,
			LastDurable:  storageJobStatusEvent{Revision: last.Revision, At: last.At, State: last.State, Reason: last.Reason, ObservationDigest: last.ObservationDigest},
			RestartState: j.State, RestartReason: j.Reason,
		},
	}
	data, err := json.Marshal(result)
	if err != nil {
		return errInput
	}
	if len(data)+1 > maxStorageAssessmentBytes {
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
	return ctx.Err()
}
