// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDashboardWitnessSealedCustodyAndExactDelivery(t *testing.T) {
	dir := witnessPrivateDirectory(t)
	store, err := newDashboardWitnessStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.close()
	raw := witnessFrames(witnessGolden())
	ref, err := store.capture(bytes.NewReader(raw), witnessTestID, "native-small", "management-1", "helper-1")
	if err != nil || !ref.Complete || ref.Delivered {
		t.Fatal("capture must not imply delivery/publication", err)
	}
	var delivered bytes.Buffer
	if err = store.forward(ref, &delivered); err != nil || !bytes.Equal(raw, delivered.Bytes()) || !ref.Delivered || ref.RawSHA256 != ref.ForwardSHA256 {
		t.Fatal("unchanged original forwarding", err)
	}
	s, err := store.original(ref)
	if err != nil || s.Datasets.Data[0].Used != 9007199254740993 {
		t.Fatal("original readback", err)
	}

	// Same bytes at another object cannot inherit original custody.
	path := filepath.Join(dir, ref.Name)
	if os.Remove(path) != nil || os.WriteFile(path, raw, 0400) != nil {
		t.Fatal("mutation preparation")
	}
	if _, err = store.original(ref); err == nil {
		t.Fatal("replacement inode accepted")
	}
}

func TestDashboardWitnessRejectedPrefixAndFailedDelivery(t *testing.T) {
	store, err := newDashboardWitnessStore(witnessPrivateDirectory(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.close()
	raw := witnessFrames(witnessGolden())[:80]
	ref, err := store.capture(bytes.NewReader(raw), witnessTestID, "native-small", "management-1", "helper-1")
	if err != nil || ref.Complete {
		t.Fatal("rejected producer prefix lost", err)
	}
	var out bytes.Buffer
	if store.forward(ref, &out) != nil || !bytes.Equal(raw, out.Bytes()) {
		t.Fatal("producer prefix was repaired")
	}
	if _, err = store.original(ref); err == nil {
		t.Fatal("invalid original eligible")
	}
	id := strings.Replace(witnessTestID, "-1", "-2", 1)
	ref, err = store.capture(bytes.NewReader(bytes.ReplaceAll(witnessFrames(witnessGolden()), []byte(witnessTestID), []byte(id))), id, "native-small", "management-1", "helper-1")
	if err != nil {
		t.Fatal(err)
	}
	if store.forward(ref, witnessShortWriter{}) == nil || ref.Delivered {
		t.Fatal("partial delivery accepted")
	}
	if _, err = store.original(ref); err == nil {
		t.Fatal("undelivered original eligible")
	}
}

type witnessShortWriter struct{}

func (witnessShortWriter) Write(b []byte) (int, error) { return len(b) / 2, nil }

func TestDashboardWitnessCustodyMutationAndBounds(t *testing.T) {
	for _, change := range []string{"content", "symlink", "directory"} {
		t.Run(change, func(t *testing.T) {
			dir := witnessPrivateDirectory(t)
			store, e := newDashboardWitnessStore(dir)
			if e != nil {
				t.Fatal(e)
			}
			defer store.close()
			raw := witnessFrames(witnessGolden())
			ref, e := store.capture(bytes.NewReader(raw), witnessTestID, "native-small", "management-1", "helper-1")
			if e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(dir, ref.Name)
			switch change {
			case "content":
				if os.Chmod(path, 0600) != nil || os.WriteFile(path, append([]byte{1}, raw[1:]...), 0400) != nil {
					t.Fatal("prepare content")
				}
			case "symlink":
				if os.Remove(path) != nil || os.Symlink("missing", path) != nil {
					t.Fatal("prepare link")
				}
			case "directory":
				if os.Rename(dir, dir+"-old") != nil || os.Mkdir(dir, 0700) != nil {
					t.Fatal("prepare directory")
				}
				defer os.RemoveAll(dir + "-old")
			}
			if store.forward(ref, &bytes.Buffer{}) == nil {
				t.Fatal("uncertain custody accepted")
			}
		})
	}
	store, e := newDashboardWitnessStore(witnessPrivateDirectory(t))
	if e != nil {
		t.Fatal(e)
	}
	defer store.close()
	for n := 1; n <= witnessAttemptCap; n++ {
		id := witnessTestID[:32] + "-" + strconv.Itoa(n)
		raw := bytes.ReplaceAll(witnessFrames(witnessGolden()), []byte(witnessTestID), []byte(id))
		if _, e := store.capture(bytes.NewReader(raw), id, "native-small", "management-1", "helper-1"); e != nil {
			t.Fatal("within attempt budget", e)
		}
	}
	if _, e := store.capture(bytes.NewReader(nil), witnessTestID[:32]+"-33", "native-small", "management-1", "helper-1"); e == nil {
		t.Fatal("attempt limit evicted original")
	}
	if _, e := store.capture(bytes.NewReader(nil), witnessTestID[:32]+"-1", "projection-10000", "management-2", "helper-2"); e == nil {
		t.Fatal("failed candidate resumed")
	}
}

func witnessPrivateDirectory(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if os.Chmod(dir, 0700) != nil {
		t.Fatal("private fixture directory")
	}
	return dir
}

func TestDashboardWitnessReadOnlyRawObjectIsNotExecutable(t *testing.T) {
	f, e := os.CreateTemp(witnessPrivateDirectory(t), "original-")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if _, e := f.Write([]byte("original bytes")); e != nil || f.Chmod(0400) != nil {
		t.Fatal("original object")
	}
	before, e := f.Stat()
	if e != nil {
		t.Fatal(e)
	}
	after, e := f.Stat()
	if e != nil || !witnessRawUnchanged(before, after) {
		t.Fatal("sealed raw object incorrectly required executable mode")
	}
}

// A cancelled or failed source must retain its exact captured prefix without
// making that original eligible or resetting the candidate's failure.
func TestDashboardWitnessInterruptedCaptureRetainsOriginal(t *testing.T) {
	for _, terminal := range []error{context.Canceled, io.ErrUnexpectedEOF} {
		t.Run(terminal.Error(), func(t *testing.T) {
			dir := witnessPrivateDirectory(t)
			store, e := newDashboardWitnessStore(dir)
			if e != nil {
				t.Fatal(e)
			}
			defer store.close()
			prefix := witnessFrames(witnessGolden())[:80]
			ref, e := store.capture(&witnessTerminalReader{bytes.NewReader(prefix), terminal}, witnessTestID, "native-small", "management-1", "helper-1")
			if e == nil || ref == nil || ref.Complete || ref.Delivered || len(store.refs) != 1 {
				t.Fatal("interrupted original missing or eligible", e)
			}
			raw, e := os.ReadFile(filepath.Join(dir, ref.Name))
			if e != nil || !bytes.Equal(raw, prefix) || ref.RawBytes != int64(len(prefix)) || store.check(store.sealed[ref.Name]) != nil {
				t.Fatal("original prefix custody lost", e)
			}
			if store.forward(ref, &bytes.Buffer{}) == nil {
				t.Fatal("interrupted original forwarded")
			}
			if _, e := store.capture(bytes.NewReader(nil), witnessTestID[:32]+"-2", "native-small", "management-1", "helper-2"); e == nil {
				t.Fatal("failed candidate resumed")
			}
		})
	}
}

type witnessTerminalReader struct {
	source   *bytes.Reader
	terminal error
}

func (r *witnessTerminalReader) Read(b []byte) (int, error) {
	if r.source.Len() == 0 {
		return 0, r.terminal
	}
	return r.source.Read(b)
}

func TestDashboardWitnessArchiveLimitRetainsBoundedPrefix(t *testing.T) {
	store, e := newDashboardWitnessStore(witnessPrivateDirectory(t))
	if e != nil {
		t.Fatal(e)
	}
	defer store.close()
	const remaining = int64(80)
	store.bytes = witnessArchiveCap - remaining
	raw := witnessFrames(witnessGolden())
	ref, e := store.capture(bytes.NewReader(raw), witnessTestID, "native-small", "management-1", "helper-1")
	if e == nil || ref == nil || ref.Complete || ref.Delivered || ref.RawBytes != remaining || store.bytes != witnessArchiveCap || len(store.refs) != 1 {
		t.Fatal("archive cap discarded prefix or exceeded bounds", e)
	}
	retained, e := os.ReadFile(filepath.Join(store.directory, ref.Name))
	if e != nil || !bytes.Equal(retained, raw[:remaining]) || store.check(store.sealed[ref.Name]) != nil {
		t.Fatal("bounded prefix lost", e)
	}
}

type witnessHalfCloseFailure struct{ bytes.Buffer }

func (*witnessHalfCloseFailure) CloseWrite() error { return io.ErrClosedPipe }
func TestDashboardWitnessHalfCloseFailureNeverDelivers(t *testing.T) {
	store, e := newDashboardWitnessStore(witnessPrivateDirectory(t))
	if e != nil {
		t.Fatal(e)
	}
	defer store.close()
	raw := witnessFrames(witnessGolden())
	ref, e := store.capture(bytes.NewReader(raw), witnessTestID, "native-small", "management-1", "helper-1")
	if e != nil {
		t.Fatal(e)
	}
	sink := &witnessHalfCloseFailure{}
	if store.forward(ref, sink) == nil || ref.Delivered || ref.ForwardSHA256 != "" || !bytes.Equal(sink.Bytes(), raw) {
		t.Fatal("byte delivery substituted for successful EOF")
	}
	if _, e := store.original(ref); e == nil {
		t.Fatal("failed EOF accepted")
	}
}
func TestDashboardWitnessFailureStagesExcludePrivateErrors(t *testing.T) {
	private := errors.New("PRIVATE /raw/path identity credentials")
	if got := witnessFailureFrom(private, "publication-original"); !errors.Is(got, errDashboardWitness) || strings.Contains(got.Error(), "PRIVATE") {
		t.Fatal("private error escaped classification", got)
	}
	if got := witnessFailure("PRIVATE /raw/path identity credentials"); got.Error() != "invalid original dashboard witness: unclassified" {
		t.Fatal("untrusted stage escaped allowlist", got)
	}
	if got := witnessFailureFrom(witnessFailure("receipt-custody"), "publication-original"); !strings.Contains(got.Error(), "receipt-custody") {
		t.Fatal("nested bounded cause lost", got)
	}
}
