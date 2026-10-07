// SPDX-License-Identifier: BSD-3-Clause
package storage

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PoolScan contains only reported legacy scan facts. Display sizes/rates are
// rounded native text, never exact byte counters. Date labels have no timezone
// or boot/process provenance. Completion/cancellation attributes no job/executor.
// No percentage is fabricated for completed/cancelled/never-scanned observations.
type PoolScan struct {
	Kind, State                                                        string
	StartedOn, PausedOn, EndedOn                                       string
	ProcessedDisplay, DurationDisplay                                  string
	ScannedDisplay, ScanTotalDisplay, IssuedDisplay, IssueTotalDisplay string
	ScanRateDisplay, IssueRateDisplay, EstimateDisplay                 string
	ErrorsReported, ProgressReported                                   bool
	Errors                                                             uint64
	ProgressHundredths                                                 uint16
}

const scanSize = `(?:0|[1-9][0-9]*)(?:\.[0-9]{1,2})?[BKMGTPE]`
const scanDuration = `(?:[1-9][0-9]* days )?(?:[01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]`

var scanDone = regexp.MustCompile(`^(scrub repaired|resilvered) (` + scanSize + `) in (` + scanDuration + `) with (0|[1-9][0-9]*) errors on (.+)$`)
var scanHeader = regexp.MustCompile(`^(scrub|resilver) (canceled on|in progress since|paused since) (.+)$`)
var scanWork = regexp.MustCompile(`^(` + scanSize + `) / (` + scanSize + `) scanned(?: at (` + scanSize + `)/s)?, (` + scanSize + `) / (` + scanSize + `) issued(?: at (` + scanSize + `)/s)?$`)
var scanProgress = regexp.MustCompile(`^(` + scanSize + `) (repaired|resilvered), ((?:0|[1-9][0-9]*)\.[0-9]{2})% done(?:, (` + scanDuration + ` to go|no estimated completion time))?$`)

func scanDateLabel(s string) bool {
	const layout = "Mon Jan _2 15:04:05 2006"
	at, err := time.Parse(layout, s)
	return err == nil && at.Format(layout) == s
}
func scanDisplay(s string) bool { return len(s) <= 6 }

// ParsePoolScan accepts only complete C-locale scrub/ordinary-resilver text
// emitted by both pinned legacy status formatters. Error scrubs, sequential
// rebuilds, unknown/localized/truncated/contradictory formats fail with zero.
// The caller must bind this text to freshly collected pool identity/topology.
func ParsePoolScan(text string) (PoolScan, error) {
	zero := PoolScan{}
	if len(text) > maxFieldBytes {
		return zero, ErrLimit
	}
	if text == "" || !statusText(text) {
		return zero, ErrUnavailable
	}
	if text == "none requested" {
		return PoolScan{Kind: "none", State: "none"}, nil
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		if m := scanDone.FindStringSubmatch(text); m != nil {
			errors, err := counter(m[4])
			if err != nil || !scanDateLabel(m[5]) || !scanDisplay(m[2]) || len(m[3]) > 31 {
				return zero, ErrUnavailable
			}
			kind := "scrub"
			if m[1] == "resilvered" {
				kind = "resilver"
			}
			return PoolScan{Kind: kind, State: "completed", ProcessedDisplay: m[2], DurationDisplay: m[3], ErrorsReported: true, Errors: errors, EndedOn: m[5]}, nil
		}
	}
	h := scanHeader.FindStringSubmatch(lines[0])
	if h == nil || !scanDateLabel(h[3]) {
		return zero, ErrUnavailable
	}
	result := PoolScan{Kind: h[1]}
	switch h[2] {
	case "canceled on":
		if len(lines) != 1 {
			return zero, ErrUnavailable
		}
		result.State = "cancelled"
		result.EndedOn = h[3]
		return result, nil
	case "in progress since":
		result.State = "running"
		result.StartedOn = h[3]
	case "paused since":
		if result.Kind != "scrub" || len(lines) != 4 {
			return zero, ErrUnavailable
		}
		result.State = "paused"
		result.PausedOn = h[3]
		started, ok := strings.CutPrefix(lines[1], "scrub started on ")
		if !ok || !scanDateLabel(started) {
			return zero, ErrUnavailable
		}
		result.StartedOn = started
		lines = append([]string{lines[0]}, lines[2:]...)
	default:
		return zero, ErrUnavailable
	}
	if len(lines) != 3 {
		return zero, ErrUnavailable
	}
	work := scanWork.FindStringSubmatch(lines[1])
	progress := scanProgress.FindStringSubmatch(lines[2])
	if work == nil || progress == nil {
		return zero, ErrUnavailable
	}
	for _, s := range []string{work[1], work[2], work[3], work[4], work[5], work[6], progress[1]} {
		if s != "" && !scanDisplay(s) {
			return zero, ErrUnavailable
		}
	}
	verb := "repaired"
	if result.Kind == "resilver" {
		verb = "resilvered"
	}
	if progress[2] != verb {
		return zero, ErrUnavailable
	}
	n, err := strconv.ParseUint(strings.ReplaceAll(progress[3], ".", ""), 10, 16)
	if err != nil || n > 10000 {
		return zero, ErrUnavailable
	}
	if result.State == "paused" {
		if work[3] != "" || work[6] != "" || progress[4] != "" {
			return zero, ErrUnavailable
		}
	} else if progress[4] == "" {
		return zero, ErrUnavailable
	}
	result.ScannedDisplay, result.ScanTotalDisplay, result.ScanRateDisplay = work[1], work[2], work[3]
	result.IssuedDisplay, result.IssueTotalDisplay, result.IssueRateDisplay = work[4], work[5], work[6]
	result.ProcessedDisplay = progress[1]
	result.ProgressReported = true
	result.ProgressHundredths = uint16(n)
	// Retain only a reported estimate label; no elapsed/rate-derived prediction.
	if progress[4] != "no estimated completion time" {
		result.EstimateDisplay = strings.TrimSuffix(progress[4], " to go")
		if len(result.EstimateDisplay) > 31 {
			return zero, ErrUnavailable
		}
	}
	return result, nil
}
