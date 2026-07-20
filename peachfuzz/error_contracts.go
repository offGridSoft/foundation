package peachfuzz

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

var (
	ErrContract    = fmt.Errorf("peachfuzz contract violation: %w", core.ErrFoundationContract)
	ErrUnavailable = errors.New("peachfuzz api unavailable")
)

const (
	ErrFmtIdentity        = "peachfuzz.%s: %w"
	ErrFmtOutcome         = "peachfuzz.RunOutcome: %w"
	ErrFmtRunStats        = "peachfuzz.RunStats: %w"
	ErrFmtProjectSnapshot = "peachfuzz.ProjectSnapshot: %w"
	ErrFmtSnapshotClient  = "peachfuzz.SnapshotClient: %w"
	ErrFmtHTTP            = "peachfuzz.HTTP: %w"
)
