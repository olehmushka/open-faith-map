// Copyright 2026 Oleh Mushka
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"time"
)

// RunStatus values match migrations/0010_congregationimport.sql's CHECK constraint verbatim.
type RunStatus string

const (
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusSucceeded RunStatus = "SUCCEEDED"
	RunStatusFailed    RunStatus = "FAILED"
)

// Run is one triggered connector execution — congregationimport_runs. Mirrors go-oikumenea's own
// hermenea import_runs concept (docs/modules/import.md), scoped to one source per run.
type Run struct {
	ID                     string
	SourceCode             string
	Status                 RunStatus
	TriggeredByPersonRID   string
	CursorAtStart          *string
	CursorAtEnd            *string
	RecordsFetched         int
	CandidatesCreated      int
	CandidatesUpdated      int
	CandidatesAutoRejected int
	Error                  *string
	StartedAt              time.Time
	FinishedAt             *time.Time
}
