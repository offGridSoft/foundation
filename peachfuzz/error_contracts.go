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
	ErrFmtIdentity               = "peachfuzz.%s: %w"
	ErrFmtOutcome                = "peachfuzz.RunOutcome: %w"
	ErrFmtRunEvidence            = "peachfuzz.RunEvidence: %w"
	ErrFmtProjectSnapshot        = "peachfuzz.ProjectSnapshot: %w"
	ErrFmtMachineContribution    = "peachfuzz.MachineContribution: %w"
	ErrFmtProjectContributionSet = "peachfuzz.ProjectContributionSet: %w"
	ErrFmtContributionReceipt    = "peachfuzz.ContributionReceipt: %w"
	ErrFmtSnapshotClient         = "peachfuzz.SnapshotClient: %w"
	ErrFmtHTTP                   = "peachfuzz.HTTP: %w"
)
