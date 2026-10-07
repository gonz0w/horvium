// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func assessmentJob(t testing.TB, i PropertyIntent) PropertyJob {
	t.Helper()
	j := jobFixture()
	r := i.Review()
	j.Intent = JobIntent{PlanID: r.ID, SubmissionID: j.Intent.SubmissionID, PolicyRevision: r.PolicyRevision, PoolGUID: r.PoolGUID, Change: r.Intent}
	j.History[0].At = uint64(i.capturedAt.Truncate(time.Millisecond).Add(time.Millisecond).UnixMilli())
	p, err := BindPropertyJob(j, i)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func assessmentStates(t testing.TB, p PropertyJob) []PropertyJob {
	t.Helper()
	result := []PropertyJob{p}
	appendState := func(from PropertyJob, state JobState, reason, digest string) PropertyJob {
		t.Helper()
		// Retained history can outlive a volatile review; assessment must not
		// recover that plan's clock or session authority.
		at := uint64(from.Intent().Review().ExpiresAt.Add(time.Hour).UnixMilli())
		next, err := TransitionPropertyJob(from, state, at, reason, digest)
		if err != nil {
			t.Fatal(err)
		}
		result = append(result, next)
		return next
	}
	appendState(p, JobCancelled, "operator-cancelled", "")
	appendState(p, JobReconcileRequired, "review-expired", "")
	a := appendState(p, JobApplying, "launch-intent", "")
	appendState(a, JobReconcileRequired, "ambiguous-effect", "")
	o := appendState(a, JobObserving, "native-observation", strings.Repeat("e", 64))
	appendState(o, JobReconcileRequired, "state-io-failure", strings.Repeat("e", 64))
	appendState(o, JobSucceeded, "verified", strings.Repeat("f", 64))
	appendState(o, JobFailed, "verified-failure", strings.Repeat("f", 64))
	return result
}

func TestPropertyJobAssessmentSeparatesFactsFromHistory(t *testing.T) {
	i, _, policy, baseline := propertyIntentFixture(t)
	for _, p := range assessmentStates(t, assessmentJob(t, i)) {
		j := p.Job()
		last := j.History[len(j.History)-1]
		t.Run(string(last.State)+"/"+last.Reason, func(t *testing.T) {
			before, err := EncodePropertyJob(p)
			if err != nil {
				t.Fatal(err)
			}
			for _, kind := range []string{"baseline", "postcondition", "neither"} {
				t.Run(kind, func(t *testing.T) {
					// Own every row independently of both the captured intent and
					// other cases. False eligibility is the real collector contract.
					var o Observation
					data, _ := json.Marshal(baseline)
					if json.Unmarshal(data, &o) != nil {
						t.Fatal("fixture copy")
					}
					o.Eligible = false
					switch kind {
					case "postcondition":
						o.Filesystems[0].Properties.Quota = Value{i.Review().Intent.Value, Local, "101", ""}
					case "neither":
						o.Filesystems[0].Properties.Quota = Value{"4194304", Local, "101", ""}
					}
					got, err := p.AssessObservation(policy, o)
					if err != nil || got.BaselineMatches != (kind == "baseline") || got.PostconditionMatches != (kind == "postcondition") {
						t.Fatal("wrong observed facts", got, err)
					}
					state, reason, _ := jobRestartProjection(j)
					want := JobRestartSummary{ID: j.ID, SubmissionID: j.Intent.SubmissionID, PoolGUID: j.Intent.PoolGUID, Last: last, State: state, Reason: reason}
					if got.Restart != want || o.Eligible {
						t.Fatal("observation replaced durable history or granted eligibility", got)
					}
					got.Restart.Last.Reason = "caller change"
					after, err := EncodePropertyJob(p)
					if err != nil || !bytes.Equal(before, after) {
						t.Fatal("assessment changed retained record", err)
					}
					if p.Intent().CompareBaseline(policy, o) == nil {
						t.Fatal("assessment relaxed precondition eligibility")
					}
				})
			}
		})
	}
}

func TestPropertyJobAssessmentAlreadyMatchingIsAmbiguous(t *testing.T) {
	policy, o, change := fixture()
	change.Property, change.Value = Compression, "lz4"
	i := postconditionIntent(t, policy, o, change)
	p := assessmentJob(t, i)
	o.Eligible = false
	got, err := p.AssessObservation(policy, o)
	if err != nil || !got.BaselineMatches || !got.PostconditionMatches || got.Restart.State != JobReconcileRequired || got.Restart.Reason != "review-expired" {
		t.Fatal("an already matching value claimed execution", got, err)
	}
}

func TestPropertyJobAssessmentInvalidInputsReturnNoPartialFacts(t *testing.T) {
	for _, mode := range []string{"zero-job", "binding", "history", "incomplete", "identity", "missing-state", "invalid-value", "oversized", "policy-invalid", "target-missing"} {
		t.Run(mode, func(t *testing.T) {
			i, _, policy, o := propertyIntentFixture(t)
			p := assessmentJob(t, i)
			o.Eligible = false
			switch mode {
			case "zero-job":
				p = PropertyJob{}
			case "binding":
				p.job.Intent.Change.Value = "none"
			case "history":
				p.job.History[0].State = JobSucceeded
			case "incomplete":
				o.Complete = false
			case "identity":
				o.Identities[0].GUID = "999"
			case "missing-state":
				o.States = nil
			case "invalid-value":
				o.Filesystems[0].Properties.Quota.Effective = "not-a-value"
			case "oversized":
				o.Filesystems[0].Name = strings.Repeat("x", maxFieldBytes+1)
			case "policy-invalid":
				policy.Scopes[0].GUID = "999"
			case "target-missing":
				// A complete, internally consistent but different pool graph
				// cannot yield a property assessment for the missing target.
				i.review.Intent.FilesystemGUID = "999"
				p.intent = i
			}
			got, err := p.AssessObservation(policy, o)
			if err == nil || got != (PropertyJobAssessment{}) {
				t.Fatal("partial facts from unusable input", got, err)
			}
		})
	}
}

func TestPropertyJobAssessmentContextDriftAndOwnership(t *testing.T) {
	for _, mode := range []string{"policy", "members", "pool", "topology", "state", "versions", "received", "other-property"} {
		t.Run(mode, func(t *testing.T) {
			i, _, policy, o := propertyIntentFixture(t)
			p := assessmentJob(t, i)
			o.Eligible = false
			switch mode {
			case "policy":
				policy.Revision++
			case "members":
				policy.Members = []PoolMember{{VdevGUID: "999", Serial: "different", SerialSource: "geom.config.ident", MediaSize: 2097152, SectorSize: 512}}
			case "pool":
				o.Pool.LoadGUID = "999"
			case "topology":
				o.Topology.Nodes[1].Address = "/dev/different"
			case "state":
				o.States[0].CanMount = Value{"off", Local, "101", ""}
			case "versions":
				o.Identities[len(o.Identities)-1].Name = "tank/data@different"
			case "received":
				o.Filesystems[0].Properties.Quota.ReceivedValue = "1048576"
			case "other-property":
				o.Filesystems[0].Properties.RefQuota = Value{"1048576", Local, "101", ""}
			}
			inputs, _ := json.Marshal([]any{policy, o})
			encoded, _ := EncodePropertyJob(p)
			decoded, err := DecodePropertyJob(encoded)
			if err != nil {
				t.Fatal(err)
			}
			got, err := decoded.AssessObservation(policy, o)
			if err != nil || got.BaselineMatches || got.PostconditionMatches || got.Restart.ID != p.Job().ID {
				t.Fatal("drift matched a captured review", got, err)
			}
			after, _ := json.Marshal([]any{policy, o})
			record, _ := EncodePropertyJob(decoded)
			if !bytes.Equal(inputs, after) || !bytes.Equal(encoded, record) {
				t.Fatal("changed inputs or restored record")
			}
		})
	}
}

func FuzzPropertyJobAssessment(f *testing.F) {
	i, _, policy, o := propertyIntentFixture(f)
	p := assessmentJob(f, i)
	for _, seed := range []string{"none", i.Review().Intent.Value, "4194304", "0", "", "18446744073709551615"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		observation := o
		observation.Filesystems = append([]Filesystem(nil), o.Filesystems...)
		observation.Eligible = false
		observation.Filesystems[0].Properties.Quota = Value{value, Local, "101", ""}
		encoded, _ := EncodePropertyJob(p)
		got, err := p.AssessObservation(policy, observation)
		if err != nil {
			if got != (PropertyJobAssessment{}) {
				t.Fatal("partial assessment on failure")
			}
		} else if got.BaselineMatches || got.PostconditionMatches != (value == i.Review().Intent.Value) || got.Restart.State != JobReconcileRequired {
			t.Fatal("wrong observed value or restart authority", got)
		}
		after, _ := EncodePropertyJob(p)
		if !bytes.Equal(encoded, after) || !reflect.DeepEqual(observation.Filesystems[0].Properties.Quota, Value{value, Local, "101", ""}) || observation.Eligible {
			t.Fatal("changed caller inputs or immutable history")
		}
	})
}
