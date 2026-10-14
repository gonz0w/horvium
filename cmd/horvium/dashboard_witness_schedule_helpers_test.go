// SPDX-License-Identifier: BSD-3-Clause
package main

type witnessScheduleSample struct{ ScheduledNS, AtNS int64 }
type witnessScheduleReport struct{ Planned, Missing, Duplicates, Late int }

func witnessScheduleCoverage(samples []witnessScheduleSample, end, interval int64) (witnessScheduleReport, error) {
	if len(samples) == 0 || end < 0 || interval <= 0 || end/interval >= 100000 {
		return witnessScheduleReport{}, errDashboardWitness
	}
	report := witnessScheduleReport{Planned: int(end/interval) + 1}
	seen := make(map[int64]bool)
	var previous int64
	for i, s := range samples {
		if s.ScheduledNS < 0 || s.ScheduledNS%interval != 0 || s.AtNS < s.ScheduledNS || s.AtNS > end || i > 0 && (s.ScheduledNS < samples[i-1].ScheduledNS || s.AtNS < previous) {
			return witnessScheduleReport{}, errDashboardWitness
		}
		slot := s.ScheduledNS / interval
		if seen[slot] {
			report.Duplicates++
		}
		seen[slot] = true
		if s.AtNS-s.ScheduledNS >= interval {
			report.Late++
		}
		previous = s.AtNS
	}
	report.Missing = report.Planned - len(seen)
	return report, nil
}
