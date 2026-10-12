// SPDX-License-Identifier: BSD-3-Clause
package storage

// DeadlineSite and DeadlinePhase identify reviewed return boundaries only.
// These identifiers do not select commands or confer runtime authority.
type DeadlineSite uint8
type DeadlinePhase uint8

const (
	DeadlineReview DeadlineSite = iota + 1
	DeadlineActivation
	DeadlineBackingCollector
	DeadlineBackingCommand
	DeadlinePoolCollector
	DeadlinePoolCommand
	DeadlineProviderCollector
	DeadlineSystemCollector
	DeadlineSystemCommand
)

const (
	DeadlineGuard DeadlinePhase = iota + 1
	DeadlineInspect
	DeadlineFirstInspection
	DeadlineSecondInspection
	DeadlinePublication
	DeadlineFinalInspection
	DeadlineCollection
	DeadlineCapture
)
