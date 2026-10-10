// SPDX-License-Identifier: BSD-3-Clause
package freebsd

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gonz0w/horvium/internal/storage"
	"golang.org/x/sys/unix"
)

// Fixture readback only, not an alternative runtime journal decoder or authority.
type nativeActivationRecord struct {
	Version, Generation uint64
	Kind                string
	Binding             struct {
		Installation, Directory, Lock string
		LedgerBacking                 string `json:"ledger_backing"`
		JournalBacking                string `json:"journal_backing"`
	}
	Owner    struct{ ID, Boot, State string }
	Census   string
	Metadata struct{ Pending bool }
	Native   struct{ Pending bool }
	Disarmed bool
}

func readActivationFixture(t *testing.T, path string) ([]byte, nativeActivationRecord) {
	t.Helper()
	data, e := os.ReadFile(filepath.Join(path, "execution.json"))
	if e != nil || len(data) > 64<<10 {
		t.Fatal("owned activation record", e)
	}
	var r nativeActivationRecord
	if json.Unmarshal(data, &r) != nil {
		t.Fatal("owned journal readback")
	}
	return data, r
}
func nativeActivationLockHeld(t *testing.T, path string) bool {
	t.Helper()
	fd, e := unix.Open(filepath.Join(path, "writer.lock"), unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if e != nil {
		t.Fatal("owned activation lock probe", e)
	}
	defer unix.Close(fd)
	e = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB)
	return e == unix.EWOULDBLOCK || e == unix.EAGAIN
}
func activationPathExists(path string) bool { _, e := os.Lstat(path); return e == nil }

// Invoked after all earlier system/ledger observations. Only fresh child domains
// on the declared independent metadata disk are changed; original data survives.
func nativeFreshActivationFixture(t *testing.T, ctx context.Context, journalParent, ledgerParent string, check func(string, bool)) {
	t.Helper()
	j, l := filepath.Join(journalParent, "native-activation"), filepath.Join(ledgerParent, "native-activation")
	a, e := activateNativeFreshExecutionJournal(ctx, j, l)
	check("activation_fixed_native_composition", e == nil && a != nil)
	defer a.Close()
	first, e := a.Observe(ctx)
	check("activation_original_observation", e == nil && first != (storage.ExecutionDomainsObservation{}))
	again, e := a.Observe(ctx)
	check("activation_repeated_observation", e == nil && again == first)
	data, r := readActivationFixture(t, j)
	raw, e := readNativeBoot()
	check("activation_exact_kernel_boot", e == nil && r.Owner.Boot == hex.EncodeToString(raw[:]))
	check("activation_initial_active_record", r.Version == 1 && r.Generation == 1 && r.Kind == "execution-journal" && r.Owner.State == "active" && len(r.Census) == 64 && !r.Metadata.Pending && !r.Native.Pending && !r.Disarmed)
	identity := func(s string) bool {
		v, e := hex.DecodeString(s)
		return e == nil && len(v) == 16 && strings.ToLower(s) == s
	}
	check("activation_fresh_identities", identity(r.Binding.Installation) && identity(r.Owner.ID) && r.Binding.Installation != r.Owner.ID)
	var st unix.Stat_t
	check("activation_private_record", unix.Stat(filepath.Join(j, "execution.json"), &st) == nil && st.Uid == 0 && st.Mode&07777 == 0600 && st.Nlink == 1)
	check("activation_journal_lease_held", nativeActivationLockHeld(t, j))
	check("activation_ledger_lease_held", nativeActivationLockHeld(t, l))
	// Copy the opaque value without relying on exported fields or copying a mutex
	// through source-level assignment. Its original pointer must still reject it.
	copied := new(storage.ExecutionJournalActivation)
	reflect.ValueOf(copied).Elem().Set(reflect.ValueOf(a).Elem())
	got, e := copied.Observe(ctx)
	check("activation_copied_observation_denied", e != nil && got == (storage.ExecutionDomainsObservation{}))
	check("activation_copied_close_denied", copied.Close() != nil)
	wire, e := json.Marshal(a)
	check("activation_no_serialized_authority", e == nil && string(wire) == "{}")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	for _, c := range []struct {
		name string
		ctx  context.Context
	}{{"nil", nil}, {"cancelled", cancelled}} {
		pj, pl := filepath.Join(journalParent, "activation-"+c.name), filepath.Join(ledgerParent, "activation-"+c.name)
		q, e := activateNativeFreshExecutionJournal(c.ctx, pj, pl)
		check("activation_"+c.name+"_no_creation", q == nil && e != nil && !activationPathExists(pj) && !activationPathExists(pl))
	}
	nativeActivationUIDFixture(t, ctx, j, l, filepath.Join(journalParent, "activation-uid"), filepath.Join(ledgerParent, "activation-uid"), check)
	source, e := NewSystemUseSource()
	if e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"boot-before", "boot-after", "system-after"} {
		pj, pl := filepath.Join(journalParent, "activation-"+mode), filepath.Join(ledgerParent, "activation-"+mode)
		boots, systems := 0, 0
		boot := func(c context.Context) ([16]byte, error) {
			boots++
			v, e := readNativeActivationBoot(c)
			if e != nil {
				return [16]byte{}, e
			}
			if mode == "boot-before" || mode == "boot-after" && boots == 3 {
				return [16]byte{}, storage.ErrUnavailable
			}
			return v, nil
		}
		system := func(c context.Context) (storage.PoolSystemUse, error) {
			systems++
			use, e := source.Collect(c)
			if e != nil {
				return storage.PoolSystemUse{}, e
			}
			if mode == "system-after" && systems == 3 {
				return use, storage.ErrUnavailable
			}
			return use, nil
		}
		q, e := storage.ActivateFreshExecutionJournal(ctx, pj, pl, readNativeExecutionBacking, system, boot)
		check("activation_"+mode+"_error_denied", q == nil && e != nil)
		if mode == "boot-before" {
			check("activation_"+mode+"_no_active_record", !activationPathExists(filepath.Join(pj, "execution.json")))
		} else {
			_, record := readActivationFixture(t, pj)
			check("activation_"+mode+"_active_preserved", record.Owner.State == "active" && record.Owner.Boot == hex.EncodeToString(raw[:]) && record.Generation == 1)
		}
		pair, e := storage.OpenExecutionDomainLeases(ctx, pj, pl)
		if e != nil {
			t.Fatal("failed activation did not release leases", e)
		}
		defer pair.Close()
		installation := strings.Repeat("1", 32)
		// Readback supplies metadata only, never an original initializer.
		if mode != "boot-before" {
			_, record := readActivationFixture(t, pj)
			installation = record.Binding.Installation
		}
		observation, e := pair.ObserveSystem(ctx, installation, readNativeExecutionBacking, source.Collect)
		check("activation_"+mode+"_observation_only_reopen", e == nil && observation != (storage.ExecutionDomainsObservation{}))
		if pair.Close() != nil {
			t.Fatal("owned failed activation observation close")
		}
		before := []byte(nil)
		if mode != "boot-before" {
			before, _ = readActivationFixture(t, pj)
		}
		q, e = activateNativeFreshExecutionJournal(ctx, pj, pl)
		same := true
		if mode != "boot-before" {
			after, _ := readActivationFixture(t, pj)
			same = bytes.Equal(before, after)
		}
		check("activation_"+mode+"_retry_preserves_history", q == nil && e != nil && same)
	}
	unknown := filepath.Join(l, "unknown")
	if os.WriteFile(unknown, []byte("owned-drift"), 0600) != nil {
		t.Fatal("owned activation ledger drift")
	}
	got, e = a.Observe(ctx)
	check("activation_ledger_drift_denied", e != nil && got == (storage.ExecutionDomainsObservation{}))
	if os.Remove(unknown) != nil {
		t.Fatal("owned unknown fixture removal")
	}
	got, e = a.Observe(ctx)
	check("activation_drift_permanently_retired", e != nil && got == (storage.ExecutionDomainsObservation{}))
	after, _ := readActivationFixture(t, j)
	check("activation_drift_preserves_journal", bytes.Equal(data, after))
	check("activation_original_close", a.Close() == nil && a.Close() == nil)
	check("activation_both_leases_released", !nativeActivationLockHeld(t, j) && !nativeActivationLockHeld(t, l))
	after, _ = readActivationFixture(t, j)
	check("activation_close_leaves_active", bytes.Equal(data, after))
	got, e = a.Observe(ctx)
	check("activation_closed_observation_denied", e != nil && got == (storage.ExecutionDomainsObservation{}))
	q, e := activateNativeFreshExecutionJournal(ctx, j, l)
	after, _ = readActivationFixture(t, j)
	check("activation_closed_no_recommission", q == nil && e != nil && bytes.Equal(data, after))
}

func TestActivationNativeUIDChild(t *testing.T) {
	if os.Getenv("HORVIUM_ACTIVATION_UID") != "1" {
		t.Skip("owned inherited-descriptor fixture only")
	}
	if os.Getenv("HORVIUM_NATIVE_VM") != "1" || os.Getenv("HORVIUM_EXECUTION_SYSTEM_VM") != "1" || os.Geteuid() != 0 {
		t.Fatal("owned root VM required")
	}
	if unix.Setgroups([]int{}) != nil || unix.Setgid(65534) != nil || unix.Setuid(65534) != nil || os.Geteuid() != 65534 {
		t.Fatal("actual credential drop")
	}
	for _, fd := range []int{3, 4} {
		var st unix.Stat_t
		if unix.Fstat(fd, &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != 0 || st.Mode&07777 != 0700 {
			t.Fatal("original root descriptors not inherited")
		}
	}
	a, e := activateNativeFreshExecutionJournal(context.Background(), os.Getenv("HORVIUM_ACTIVATION_JOURNAL"), os.Getenv("HORVIUM_ACTIVATION_LEDGER"))
	if a != nil || !errors.Is(e, storage.ErrDenied) {
		t.Fatal("nonroot activation accepted", e)
	}
	os.Stdout.WriteString("ACTIVATION_UID_RESULT inherited=pass denied=pass\n")
}
func nativeActivationUIDFixture(t *testing.T, ctx context.Context, j, l, pj, pl string, check func(string, bool)) {
	t.Helper()
	jf, e := os.Open(j)
	if e != nil {
		t.Fatal(e)
	}
	defer jf.Close()
	lf, e := os.Open(l)
	if e != nil {
		t.Fatal(e)
	}
	defer lf.Close()
	c, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	cmd := exec.CommandContext(c, os.Args[0], "-test.v", "-test.run=^TestActivationNativeUIDChild$")
	cmd.Env = append(os.Environ(), "HORVIUM_ACTIVATION_UID=1", "HORVIUM_ACTIVATION_JOURNAL="+pj, "HORVIUM_ACTIVATION_LEDGER="+pl)
	cmd.ExtraFiles = []*os.File{jf, lf}
	output, e := cmd.CombinedOutput()
	check("activation_uid_child_reaped", e == nil && cmd.ProcessState != nil && cmd.ProcessState.Exited())
	check("activation_uid_inherited_denied", len(output) < 4096 && bytes.Contains(output, []byte("ACTIVATION_UID_RESULT inherited=pass denied=pass\n")))
	check("activation_uid_no_creation", !activationPathExists(pj) && !activationPathExists(pl))
}
