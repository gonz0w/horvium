// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"reflect"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

// PropertyEnrollmentState binds a volatile review to one held root lease and
// complete inspected metadata. It is read-only evidence, never publication,
// recovered-state authority, a reservation or a transferable admission token.
type PropertyEnrollmentState struct {
	lease    *StateDirectory
	plan     PropertyEnrollmentPlan
	previous Policy
	digest   [32]byte
}

// BindPropertyEnrollment requires fresh native inputs and an explicit exact
// predecessor. Only an entirely zero predecessor requests first enrollment;
// missing/unsafe reads cannot supply that decision. No record is written.
func (s *StateDirectory) BindPropertyEnrollment(at time.Time, rootContext string, plan PropertyEnrollmentPlan, previous Policy, o Observation, disks []inventory.DiskObservation, u PoolSystemUse) (PropertyEnrollmentState, error) {
	if err := plan.Validate(at, rootContext, plan.review.Policy, o, disks, u); err != nil {
		return PropertyEnrollmentState{}, err
	}
	canonical, err := enrollmentPredecessor(plan.review.Policy, previous)
	if err != nil {
		return PropertyEnrollmentState{}, err
	}
	digest, err := s.propertyEnrollmentState(plan.review.Policy, canonical, u, defaultStateCatalogLimits, nil)
	if err != nil {
		return PropertyEnrollmentState{}, err
	}
	return PropertyEnrollmentState{lease: s, plan: plan, previous: canonical, digest: digest}, nil
}

func (p PropertyEnrollmentState) Review() PropertyEnrollmentReview { return p.plan.Review() }
func (p PropertyEnrollmentState) Predecessor() Policy              { return cloneEnrollmentPolicy(p.previous) }

// Validate independently recollects the complete metadata through the original
// lease and repeats native/expiry/context checks. It claims/releases nothing and
// cannot validate restored metadata or an absent old executor. No hydration API.
func (p PropertyEnrollmentState) Validate(at time.Time, rootContext string, o Observation, disks []inventory.DiskObservation, u PoolSystemUse) error {
	if p.lease == nil || p.plan.review.ID == "" {
		return ErrInvalid
	}
	if err := p.plan.Validate(at, rootContext, p.plan.review.Policy, o, disks, u); err != nil {
		return err
	}
	fresh, err := p.lease.propertyEnrollmentState(p.plan.review.Policy, p.previous, u, defaultStateCatalogLimits, nil)
	if err != nil {
		return err
	}
	if fresh != p.digest {
		return ErrDrift
	}
	return nil
}

func enrollmentPredecessor(next, previous Policy) (Policy, error) {
	if reflect.DeepEqual(previous, Policy{}) {
		if next.Revision != 1 {
			return Policy{}, ErrDrift
		}
		return Policy{}, nil
	}
	canonical, err := normalizeOwnership(previous)
	if err != nil {
		return Policy{}, err
	}
	if next.PoolGUID != canonical.PoolGUID || !validOwnershipPublicationRevision(next.Revision, canonical.Revision) {
		return Policy{}, ErrDrift
	}
	return canonical, nil
}

func checkEnrollmentStateCatalog(c StateCatalog, next, previous Policy, size uint64, limits stateCatalogLimits, jobs JobReservations, backing propertyStateBacking, u PoolSystemUse) error {
	if err := allowMixedOwnershipPublication(c, next, previous.Revision, size, limits); err != nil {
		return err
	}
	if err := checkPropertyStateJobs(jobs, next.PoolGUID); err != nil {
		return err
	}
	owners := map[string]bool{next.PoolGUID: true}
	for _, p := range c.Ownership {
		owners[p.PoolGUID] = true
	}
	// Include the proposed owner even before its first record exists, so state
	// cannot become data-backed as a consequence of this proposed enrollment.
	return checkPropertyStateBacking(backing, u, owners)
}
