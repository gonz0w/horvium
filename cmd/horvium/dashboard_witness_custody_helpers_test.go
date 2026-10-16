// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gonz0w/horvium/internal/inventory"
)

type dashboardWitnessRef struct {
	Name           string                 `json:"name"`
	ID             string                 `json:"request_id"`
	Mode           string                 `json:"mode"`
	Management     string                 `json:"management_incarnation"`
	Helper         string                 `json:"helper_incarnation"`
	RawSHA256      string                 `json:"raw_sha256"`
	RawBytes       int64                  `json:"raw_bytes"`
	CaptureFailure string                 `json:"capture_failure,omitempty"`
	Complete       bool                   `json:"candidate_complete"`
	Delivered      bool                   `json:"delivered"`
	ForwardSHA256  string                 `json:"forward_sha256"`
	Forward        *witnessForwardReceipt `json:"forward,omitempty"`
}
type witnessSealed struct {
	ref  dashboardWitnessRef
	info os.FileInfo
	file *os.File
}

func witnessRawUnchanged(before, after os.FileInfo) bool {
	return before != nil && after != nil && after.Mode().IsRegular() && after.Mode().Perm() == 0400 && os.SameFile(before, after) && before.Size() == after.Size() && before.Mode() == after.Mode() && before.ModTime().Equal(after.ModTime())
}

type dashboardWitnessStore struct {
	captureBoundary *witnessCaptureBoundary
	mu              sync.Mutex
	directory       string
	root            *os.Root
	info            os.FileInfo
	refs            []*dashboardWitnessRef
	sealed          map[string]*witnessSealed
	counts          map[string]int
	keys            map[string]bool
	namespace       map[string]string
	counter         map[string]uint64
	bytes           int64
	failed          bool
	active          bool
}

func newDashboardWitnessStore(dir string) (*dashboardWitnessStore, error) {
	info, e := os.Lstat(dir)
	abs, ae := filepath.EvalSymlinks(dir)
	if e != nil || ae != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0700 {
		return nil, errDashboardWitness
	}
	root, e := os.OpenRoot(abs)
	if e != nil {
		return nil, errDashboardWitness
	}
	opened, e := root.Stat(".")
	if e != nil || !os.SameFile(info, opened) {
		root.Close()
		return nil, errDashboardWitness
	}
	return &dashboardWitnessStore{directory: abs, root: root, info: info, counts: map[string]int{}, keys: map[string]bool{}, sealed: map[string]*witnessSealed{}, namespace: map[string]string{}, counter: map[string]uint64{}}, nil
}
func (s *dashboardWitnessStore) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var e error
	for _, v := range s.sealed {
		if v.file != nil {
			if ce := v.file.Close(); ce != nil {
				e = ce
			}
			v.file = nil
		}
	}
	if s.root != nil {
		if ce := s.root.Close(); ce != nil {
			e = ce
		}
		s.root = nil
	}
	return e
}
func (s *dashboardWitnessStore) location() error {
	if s == nil || s.root == nil {
		return errDashboardWitness
	}
	i, e := os.Lstat(s.directory)
	if e != nil || !i.IsDir() || i.Mode().Perm() != 0700 || !os.SameFile(i, s.info) {
		return errDashboardWitness
	}
	return nil
}
func (s *dashboardWitnessStore) capture(r io.Reader, id, mode, management, helper string) (out *dashboardWitnessRef, err error) {
	s.mu.Lock()
	key := management + "/" + helper + "/" + id
	if s.failed || s.active || s.bytes < 0 || s.bytes > witnessArchiveCap || s.location() != nil || !dashboardGenerationPattern.MatchString(id) || mode != "native-small" && mode != "projection-10000" || management == "" || helper == "" || len(management) > 256 || len(helper) > 256 || s.keys[key] || s.counts[mode] >= witnessAttemptCap {
		s.failed = true
		s.mu.Unlock()
		return nil, errDashboardWitness
	}
	parts := strings.Split(id, "-")
	counter, e := strconv.ParseUint(parts[1], 10, 64)
	if e != nil || counter == 0 || counter <= s.counter[management] || s.namespace[management] != "" && s.namespace[management] != parts[0] || s.namespace[parts[0]] != "" && s.namespace[parts[0]] != management {
		s.failed = true
		s.mu.Unlock()
		return nil, errDashboardWitness
	}
	s.counter[management] = counter
	s.namespace[management] = parts[0]
	s.namespace[parts[0]] = management
	s.active = true
	s.keys[key] = true
	s.counts[mode]++
	name := fmt.Sprintf("original-%03d.wire", len(s.refs)+1)
	remaining := min(int64(witnessRawCap), int64(witnessArchiveCap)-s.bytes)
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.active = false
		if err != nil {
			s.failed = true
		}
		s.mu.Unlock()
	}()
	boundary := &witnessCaptureBoundary{ID: id, Management: management, Helper: helper, Stage: "open"}
	var captureIO error
	defer func() {
		boundary.Error = witnessClassifyForwardError(captureIO)
		s.mu.Lock()
		s.captureBoundary = boundary
		s.mu.Unlock()
	}()
	f, e := s.root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		captureIO = e
		return nil, errDashboardWitness
	}
	writerClosed := false
	defer func() {
		if !writerClosed {
			_ = f.Close()
		}
	}()
	boundary.Stage = "copy"
	n, digest, e := witnessCopyCapture(f, r, remaining)
	boundary.AcknowledgedBytes = n
	captureIO = e
	captureFailure := ""
	if e != nil {
		captureFailure = "capture-io"
	}
	if e == nil && n == remaining && witnessEOF(r) != nil {
		captureFailure = "raw-limit"
		if remaining < witnessRawCap {
			captureFailure = "archive-limit"
		}
	}
	// Retain and seal the bounded bytes already written even when capture fails.
	// No delivery/publication may use this original; the candidate stays failed.
	if s.location() != nil {
		return nil, errDashboardWitness
	}
	if se := f.Sync(); se != nil {
		boundary.Stage = "sync"
		captureIO = se
		return nil, errDashboardWitness
	}
	if me := f.Chmod(0400); me != nil {
		boundary.Stage = "chmod"
		captureIO = me
		return nil, errDashboardWitness
	}
	info, e := f.Stat()
	if e != nil {
		boundary.Stage = "stat"
		captureIO = e
		return nil, errDashboardWitness
	}
	if ce := f.Close(); ce != nil {
		boundary.Stage = "close"
		captureIO = ce
		return nil, errDashboardWitness
	}
	writerClosed = true
	read, e := s.root.Open(name)
	if e != nil {
		boundary.Stage = "reopen"
		captureIO = e
		return nil, errDashboardWitness
	}
	ref := dashboardWitnessRef{Name: name, ID: id, Mode: mode, Management: management, Helper: helper, RawSHA256: digest, RawBytes: n, CaptureFailure: captureFailure}
	sealed := &witnessSealed{ref: ref, info: info, file: read}
	if s.check(sealed) != nil {
		read.Close()
		return nil, errDashboardWitness
	}
	_, e = read.Seek(0, io.SeekStart)
	if e != nil {
		read.Close()
		return nil, errDashboardWitness
	}
	snapshot, de := dashboardDecodeWitness(read, id)
	sealed.ref.Complete = captureFailure == "" && de == nil && snapshot.Complete()
	// Partial producer snapshots can be valid; normal exact-value eligibility is
	// complete-only. Neither validity nor EOF supplies publication authority.
	outRef := sealed.ref
	s.mu.Lock()
	s.bytes += n
	s.sealed[name] = sealed
	s.refs = append(s.refs, &outRef)
	s.mu.Unlock()
	if captureFailure != "" {
		return &outRef, witnessFailure(captureFailure)
	}
	boundary.Stage = "sealed"
	return &outRef, nil
}
func (s *dashboardWitnessStore) check(v *witnessSealed) error {
	if s.location() != nil || v == nil || v.file == nil {
		return errDashboardWitness
	}
	path, e := s.root.Lstat(v.ref.Name)
	if e != nil || !path.Mode().IsRegular() || path.Mode().Perm() != 0400 || !os.SameFile(path, v.info) {
		return errDashboardWitness
	}
	opened, e := v.file.Stat()
	if e != nil || !os.SameFile(opened, path) || opened.Size() != v.ref.RawBytes || !opened.ModTime().Equal(v.info.ModTime()) {
		return errDashboardWitness
	}
	h := sha256.New()
	n, e := io.CopyBuffer(h, io.NewSectionReader(v.file, 0, v.ref.RawBytes+1), make([]byte, witnessFrameCap))
	after, ae := v.file.Stat()
	if e != nil || ae != nil || n != v.ref.RawBytes || fmt.Sprintf("%x", h.Sum(nil)) != v.ref.RawSHA256 || !os.SameFile(after, path) || after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) {
		return errDashboardWitness
	}
	return nil
}
func (s *dashboardWitnessStore) resolve(ref *dashboardWitnessRef) (*witnessSealed, error) {
	if ref == nil {
		return nil, errDashboardWitness
	}
	v := s.sealed[ref.Name]
	if v == nil || v.ref.ID != ref.ID || v.ref.Mode != ref.Mode || v.ref.Management != ref.Management || v.ref.Helper != ref.Helper || v.ref.RawSHA256 != ref.RawSHA256 || v.ref.RawBytes != ref.RawBytes || v.ref.Complete != ref.Complete || v.ref.CaptureFailure != ref.CaptureFailure || s.check(v) != nil {
		return nil, errDashboardWitness
	}
	return v, nil
}
func (s *dashboardWitnessStore) forward(ref *dashboardWitnessRef, w io.Writer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	start := time.Now()
	v, e := s.resolve(ref)
	if e != nil || s.failed || v.ref.Delivered {
		s.failed = true
		return errDashboardWitness
	}
	sink := newWitnessForwardWriter(w)
	stage, closed := "copy", false
	defer func() {
		v.ref.Forward = sink.receipt(stage, e, closed, start)
		ref.Forward = v.ref.Forward
	}()
	n, e := io.CopyBuffer(sink, io.NewSectionReader(v.file, 0, v.ref.RawBytes), make([]byte, witnessFrameCap))
	if e != nil {
		s.failed = true
		return errDashboardWitness
	}
	stage = "verify"
	if n != v.ref.RawBytes || fmt.Sprintf("%x", sink.hash.Sum(nil)) != v.ref.RawSHA256 || s.check(v) != nil {
		e = errDashboardWitness
		s.failed = true
		return errDashboardWitness
	}
	// Keep the custody mutex through native write-half EOF so an original-owner
	// readback cannot observe delivery before the management client can publish.
	// Plain buffers in portable tests model byte delivery only; native eligibility
	// additionally requires the verified Unix endpoint and this actual half-close.
	if endpoint, ok := w.(interface{ CloseWrite() error }); ok {
		stage = "close-write"
		if e = endpoint.CloseWrite(); e != nil {
			s.failed = true
			return errDashboardWitness
		}
		closed = true
	}
	stage = "delivered"
	v.ref.Delivered = true
	v.ref.ForwardSHA256 = v.ref.RawSHA256
	ref.Delivered = true
	ref.ForwardSHA256 = v.ref.ForwardSHA256
	return nil
}
func (s *dashboardWitnessStore) original(ref *dashboardWitnessRef) (inventory.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, e := s.resolve(ref)
	if e != nil || s.failed || !v.ref.Complete || !v.ref.Delivered || ref.ForwardSHA256 != v.ref.ForwardSHA256 {
		return inventory.Snapshot{}, errDashboardWitness
	}
	snapshot, e := dashboardDecodeWitness(io.NewSectionReader(v.file, 0, v.ref.RawBytes), v.ref.ID)
	if e != nil || s.check(v) != nil {
		return inventory.Snapshot{}, errDashboardWitness
	}
	return snapshot, nil
}
