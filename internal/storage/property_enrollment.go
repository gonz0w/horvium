// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

// PropertyEnrollmentReview shows an explicit root proposal and the complete
// observed pool/system context. It is an ephemeral internal model, not a wire
// format, enrollment record, publication permission or authenticated root actor.
type PropertyEnrollmentReview struct {
	ID                   string
	CreatedAt, ExpiresAt time.Time
	Policy               Policy
	Observation          Observation
	System               SystemDeviceUse
	Pools                []ImportedPoolUse
}

// PropertyEnrollmentPlan owns a volatile read-only review. No implemented caller
// provisions policy from it. Its shared one-attempt claim can be used only by a separately root-bound
// metadata publisher. Runtime authorization and full recovery remain future gates.
type PropertyEnrollmentPlan struct {
	review      PropertyEnrollmentReview
	rootContext string
	baseline    [32]byte
	claim       *enrollmentPublicationClaim
}

// BuildPropertyEnrollment requires a fully explicit proposal from a future
// trusted OS-root workflow, independently fresh bounded native observations and
// provider/system correspondence. The context binds only this volatile review;
// a client-supplied string cannot authenticate root. No scope/member is inferred.
func BuildPropertyEnrollment(at time.Time, rootContext string, proposed Policy, o Observation, disks []inventory.DiskObservation, u PoolSystemUse) (PropertyEnrollmentPlan, error) {
	if at.IsZero() || !token(rootContext) {
		return PropertyEnrollmentPlan{}, ErrInvalid
	}
	r, baseline, err := preparePropertyEnrollment(proposed, o, disks, u)
	if err != nil {
		return PropertyEnrollmentPlan{}, err
	}
	var id [32]byte
	if _, err = rand.Read(id[:]); err != nil {
		return PropertyEnrollmentPlan{}, ErrUnavailable
	}
	r.ID, r.CreatedAt, r.ExpiresAt = hex.EncodeToString(id[:]), at, at.Add(5*time.Minute)
	data, err := json.Marshal(r)
	if err != nil {
		return PropertyEnrollmentPlan{}, ErrInvalid
	}
	if len(data) > maxPlanBytes {
		return PropertyEnrollmentPlan{}, ErrLimit
	}
	return PropertyEnrollmentPlan{review: r, rootContext: rootContext, baseline: baseline, claim: &enrollmentPublicationClaim{}}, nil
}

func (p PropertyEnrollmentPlan) Review() PropertyEnrollmentReview {
	r := p.review
	r.Policy = cloneEnrollmentPolicy(r.Policy)
	r.Observation = cloneEnrollmentObservation(r.Observation)
	u := cloneDeviceUse(DeviceUse{Mounts: r.System.Mounts, Swap: r.System.Swap, Pools: r.Pools})
	r.System, r.Pools = SystemDeviceUse{Mounts: u.Mounts, Swap: u.Swap}, u.Pools
	return r
}

// Validate repeats the prerequisite with independently fresh inputs. Repeated
// validation never claims, persists or enables anything. Cached inputs or an
// unchanged revision alone cannot establish freshness or recovery authority.
func (p PropertyEnrollmentPlan) Validate(at time.Time, rootContext string, proposed Policy, o Observation, disks []inventory.DiskObservation, u PoolSystemUse) error {
	if p.review.ID == "" {
		return ErrInvalid
	}
	if at.Before(p.review.CreatedAt) || !at.Before(p.review.ExpiresAt) {
		return ErrExpired
	}
	if rootContext != p.rootContext {
		return ErrDenied
	}
	_, digest, err := preparePropertyEnrollment(proposed, o, disks, u)
	if err != nil || digest != p.baseline {
		return ErrDrift
	}
	return nil
}

func preparePropertyEnrollment(p Policy, o Observation, disks []inventory.DiskObservation, u PoolSystemUse) (PropertyEnrollmentReview, [32]byte, error) {
	var zero [32]byte
	// Bound worst-case JSON/structure costs before maps, copies, sorting or
	// encoding. Six times the review ceiling is a conservative working budget;
	// the final review itself must still fit the exact 64 KiB ceiling.
	remaining := 6*maxPlanBytes - 4096
	for _, input := range []any{p, o, disks, u.system, u.pools, u.topologies} {
		if err := enrollmentBudget(reflect.ValueOf(input), &remaining); err != nil {
			return PropertyEnrollmentReview{}, zero, err
		}
	}
	if o.Eligible {
		return PropertyEnrollmentReview{}, zero, ErrDenied
	}
	canonical, err := normalizeOwnership(p)
	if err != nil {
		return PropertyEnrollmentReview{}, zero, err
	}
	if err = ValidatePropertyMembers(canonical, o, disks); err != nil {
		return PropertyEnrollmentReview{}, zero, err
	}
	if err = CheckPropertySystemExclusions(o, u); err != nil {
		return PropertyEnrollmentReview{}, zero, err
	}
	if err = CheckPropertySystemProviders(o.PoolGUID, u, disks); err != nil {
		return PropertyEnrollmentReview{}, zero, err
	}
	o = cloneEnrollmentObservation(o)
	sort.Slice(o.Filesystems, func(i, j int) bool { return o.Filesystems[i].GUID < o.Filesystems[j].GUID })
	sort.Slice(o.Identities, func(i, j int) bool { return o.Identities[i].Name < o.Identities[j].Name })
	sort.Slice(o.States, func(i, j int) bool { return o.States[i].GUID < o.States[j].GUID })
	sort.Slice(o.Features.Values, func(i, j int) bool { return o.Features.Values[i].Name < o.Features.Values[j].Name })
	o.Topology = normalizedTopology(o.Topology)
	use := normalizeDeviceUse(DeviceUse{Mounts: u.system.Mounts, Swap: u.system.Swap, Pools: u.pools})
	sortProviders := func(d []inventory.Provider) { sort.Slice(d, func(i, j int) bool { return d[i].Name < d[j].Name }) }
	for i := range use.Mounts {
		sortProviders(use.Mounts[i].Disks)
	}
	for i := range use.Swap {
		sortProviders(use.Swap[i].Disks)
	}
	for i := range use.Pools {
		sortProviders(use.Pools[i].Disks)
	}
	sort.Slice(use.Mounts, func(i, j int) bool { return use.Mounts[i].Target < use.Mounts[j].Target })
	sort.Slice(use.Swap, func(i, j int) bool { return use.Swap[i].Source < use.Swap[j].Source })
	sort.Slice(use.Pools, func(i, j int) bool { return use.Pools[i].GUID < use.Pools[j].GUID })
	topologies := make([]PoolTopology, len(u.topologies))
	for i := range u.topologies {
		topologies[i] = normalizedTopology(u.topologies[i])
	}
	sort.Slice(topologies, func(i, j int) bool { return topologies[i].PoolGUID < topologies[j].PoolGUID })
	r := PropertyEnrollmentReview{Policy: canonical, Observation: o, System: SystemDeviceUse{Mounts: use.Mounts, Swap: use.Swap}, Pools: use.Pools}
	// Private sizing/hash encoding, never a durable or transport contract.
	data, err := json.Marshal(struct {
		Review     PropertyEnrollmentReview
		Topologies []PoolTopology
	}{r, topologies})
	if err != nil {
		return PropertyEnrollmentReview{}, zero, ErrInvalid
	}
	return r, sha256.Sum256(data), nil
}

func cloneEnrollmentPolicy(p Policy) Policy {
	p.Scopes = append([]Scope(nil), p.Scopes...)
	for i := range p.Scopes {
		p.Scopes[i].Properties = append([]Property(nil), p.Scopes[i].Properties...)
	}
	p.Members = append([]PoolMember(nil), p.Members...)
	return p
}
func cloneEnrollmentObservation(o Observation) Observation {
	o.Filesystems = append([]Filesystem(nil), o.Filesystems...)
	o.Identities = append([]DatasetIdentity(nil), o.Identities...)
	o.States = append([]FilesystemState(nil), o.States...)
	o.Features.Values = append([]PoolFeature(nil), o.Features.Values...)
	o.Topology.Nodes = append([]Vdev(nil), o.Topology.Nodes...)
	return o
}

// Walk only these fixed typed models; no user-defined marshaler, map, pointer or
// recursive object is accepted. Stop before traversing oversized arrays/strings.
func enrollmentBudget(v reflect.Value, remaining *int) error {
	take := func(n int) error {
		if n > *remaining {
			return ErrLimit
		}
		*remaining -= n
		return nil
	}
	switch v.Kind() {
	case reflect.String:
		if v.Len() > *remaining/6 {
			return ErrLimit
		}
		return take(2 + 6*v.Len())
	case reflect.Bool:
		return take(5)
	case reflect.Int, reflect.Int64, reflect.Uint64:
		return take(20)
	case reflect.Slice:
		if v.Len() > *remaining/2 {
			return ErrLimit
		}
		if err := take(4); err != nil {
			return err
		}
		for i := 0; i < v.Len(); i++ {
			if err := take(1); err != nil {
				return err
			}
			if err := enrollmentBudget(v.Index(i), remaining); err != nil {
				return err
			}
		}
	case reflect.Struct:
		if err := take(2); err != nil {
			return err
		}
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			// Inventory models have JSON field names longer than Go names.
			keyBytes := max(len(field.Name), len(field.Tag.Get("json")))
			if err := take(keyBytes + 4); err != nil {
				return err
			}
			if err := enrollmentBudget(v.Field(i), remaining); err != nil {
				return err
			}
		}
	default:
		return ErrInvalid
	}
	return nil
}
