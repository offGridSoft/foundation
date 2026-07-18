package license

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	BugSeatMemberStatusTokenActive       = "active"
	BugSeatMemberStatusTokenRemoved      = "removed"
	BugSeatInviteStatusTokenPending      = "pending"
	BugSeatInviteStatusTokenAccepted     = "accepted"
	BugSeatInviteStatusTokenCancelled    = "cancelled"
	BugSeatInviteStatusTokenExpired      = "expired"
	BugSeatAssignmentStatusTokenActive   = "active"
	BugSeatAssignmentStatusTokenRemoved  = "removed"
	BugSeatAssignmentStatusTokenTransfer = "transferred"
)

type BugSeatGeneration uint64

func (g BugSeatGeneration) IsZero() bool { return g == 0 }

func (g BugSeatGeneration) Validate() error {
	if g.IsZero() {
		return fmt.Errorf(ErrFmtSeatGeneration, core.ErrLicenseContract)
	}
	return nil
}

func (g BugSeatGeneration) Next() (BugSeatGeneration, error) {
	if err := g.Validate(); err != nil || uint64(g) == math.MaxUint64 {
		return 0, fmt.Errorf(ErrFmtSeatGeneration, core.ErrLicenseContract)
	}
	return g + 1, nil
}

func (g BugSeatGeneration) MarshalJSON() ([]byte, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(uint64(g))
}

//validate:unmarshal_ignore reason="Validation occurs before assigning the decoded generation to the receiver."
func (g *BugSeatGeneration) UnmarshalJSON(data []byte) error {
	var value uint64
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtSeatGeneration, core.ErrLicenseContract)
	}
	parsed := BugSeatGeneration(value)
	if err := parsed.Validate(); err != nil {
		return err
	}
	*g = parsed
	return nil
}

type BugSeatMemberStatus uint8

const (
	bugSeatMemberStatusInvalid BugSeatMemberStatus = iota
	BugSeatMemberStatusActive
	BugSeatMemberStatusRemoved
)

func bugSeatMemberStatusNames() [BugSeatMemberStatusRemoved + 1]string {
	return [...]string{
		BugSeatMemberStatusActive:  BugSeatMemberStatusTokenActive,
		BugSeatMemberStatusRemoved: BugSeatMemberStatusTokenRemoved,
	}
}

func (s BugSeatMemberStatus) IsValid() bool {
	return s > bugSeatMemberStatusInvalid && int(s) < len(bugSeatMemberStatusNames()) && bugSeatMemberStatusNames()[s] != ""
}

func (s BugSeatMemberStatus) String() string {
	if s.IsValid() {
		return bugSeatMemberStatusNames()[s]
	}
	return ""
}

func (s BugSeatMemberStatus) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf(ErrFmtSeatMemberStatus, core.ErrLicenseContract)
	}
	return nil
}

func ParseBugSeatMemberStatus(token string) (BugSeatMemberStatus, error) {
	for status := BugSeatMemberStatusActive; int(status) < len(bugSeatMemberStatusNames()); status++ {
		if status.String() == token {
			return status, nil
		}
	}
	return bugSeatMemberStatusInvalid, fmt.Errorf(ErrFmtSeatMemberStatus, core.ErrLicenseContract)
}

func (s BugSeatMemberStatus) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.String())
}

//validate:unmarshal_ignore reason="ParseBugSeatMemberStatus validates a temporary before assignment."
func (s *BugSeatMemberStatus) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalSeatScalar(data, ErrFmtSeatMemberStatus, ParseBugSeatMemberStatus)
	if err == nil {
		*s = parsed
	}
	return err
}

type BugSeatInviteStatus uint8

const (
	bugSeatInviteStatusInvalid BugSeatInviteStatus = iota
	BugSeatInviteStatusPending
	BugSeatInviteStatusAccepted
	BugSeatInviteStatusCancelled
	BugSeatInviteStatusExpired
)

func bugSeatInviteStatusNames() [BugSeatInviteStatusExpired + 1]string {
	return [...]string{
		BugSeatInviteStatusPending:   BugSeatInviteStatusTokenPending,
		BugSeatInviteStatusAccepted:  BugSeatInviteStatusTokenAccepted,
		BugSeatInviteStatusCancelled: BugSeatInviteStatusTokenCancelled,
		BugSeatInviteStatusExpired:   BugSeatInviteStatusTokenExpired,
	}
}

func (s BugSeatInviteStatus) IsValid() bool {
	return s > bugSeatInviteStatusInvalid && int(s) < len(bugSeatInviteStatusNames()) && bugSeatInviteStatusNames()[s] != ""
}

func (s BugSeatInviteStatus) String() string {
	if s.IsValid() {
		return bugSeatInviteStatusNames()[s]
	}
	return ""
}

func (s BugSeatInviteStatus) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf(ErrFmtSeatInviteStatus, core.ErrLicenseContract)
	}
	return nil
}

func ParseBugSeatInviteStatus(token string) (BugSeatInviteStatus, error) {
	for status := BugSeatInviteStatusPending; int(status) < len(bugSeatInviteStatusNames()); status++ {
		if status.String() == token {
			return status, nil
		}
	}
	return bugSeatInviteStatusInvalid, fmt.Errorf(ErrFmtSeatInviteStatus, core.ErrLicenseContract)
}

func (s BugSeatInviteStatus) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.String())
}

//validate:unmarshal_ignore reason="ParseBugSeatInviteStatus validates a temporary before assignment."
func (s *BugSeatInviteStatus) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalSeatScalar(data, ErrFmtSeatInviteStatus, ParseBugSeatInviteStatus)
	if err == nil {
		*s = parsed
	}
	return err
}

type BugSeatAssignmentStatus uint8

const (
	bugSeatAssignmentStatusInvalid BugSeatAssignmentStatus = iota
	BugSeatAssignmentStatusActive
	BugSeatAssignmentStatusRemoved
	BugSeatAssignmentStatusTransferred
)

func bugSeatAssignmentStatusNames() [BugSeatAssignmentStatusTransferred + 1]string {
	return [...]string{
		BugSeatAssignmentStatusActive:      BugSeatAssignmentStatusTokenActive,
		BugSeatAssignmentStatusRemoved:     BugSeatAssignmentStatusTokenRemoved,
		BugSeatAssignmentStatusTransferred: BugSeatAssignmentStatusTokenTransfer,
	}
}

func (s BugSeatAssignmentStatus) IsValid() bool {
	return s > bugSeatAssignmentStatusInvalid && int(s) < len(bugSeatAssignmentStatusNames()) && bugSeatAssignmentStatusNames()[s] != ""
}

func (s BugSeatAssignmentStatus) String() string {
	if s.IsValid() {
		return bugSeatAssignmentStatusNames()[s]
	}
	return ""
}

func (s BugSeatAssignmentStatus) Validate() error {
	if !s.IsValid() {
		return fmt.Errorf(ErrFmtSeatAssignmentStatus, core.ErrLicenseContract)
	}
	return nil
}

func ParseBugSeatAssignmentStatus(token string) (BugSeatAssignmentStatus, error) {
	for status := BugSeatAssignmentStatusActive; int(status) < len(bugSeatAssignmentStatusNames()); status++ {
		if status.String() == token {
			return status, nil
		}
	}
	return bugSeatAssignmentStatusInvalid, fmt.Errorf(ErrFmtSeatAssignmentStatus, core.ErrLicenseContract)
}

func (s BugSeatAssignmentStatus) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.String())
}

//validate:unmarshal_ignore reason="ParseBugSeatAssignmentStatus validates a temporary before assignment."
func (s *BugSeatAssignmentStatus) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalSeatScalar(data, ErrFmtSeatAssignmentStatus, ParseBugSeatAssignmentStatus)
	if err == nil {
		*s = parsed
	}
	return err
}

type BugSeatMember struct {
	MemberID       BugSeatMemberID       `json:"member_id"`
	AccountSubject BugSeatAccountSubject `json:"account_subject"`
	InviteAddress  BugSeatInviteAddress  `json:"invite_address"`
	ActivatedAt    core.UnixNanoTime     `json:"activated_at"`
	DeactivatedAt  *core.UnixNanoTime    `json:"deactivated_at,omitempty"`
	Generation     BugSeatGeneration     `json:"generation"`
	Schema         core.SchemaID         `json:"schema"`
	Status         BugSeatMemberStatus   `json:"status"`
}

func (m BugSeatMember) Validate() error {
	if m.Schema != core.SchemaBugSeatMember {
		return seatMemberError(core.ErrLicenseContract)
	}
	if err := validateSeatMemberFields(m); err != nil {
		return err
	}
	return validateSeatMemberState(m)
}

func validateSeatMemberFields(member BugSeatMember) error {
	validators := []func() error{
		member.MemberID.Validate,
		member.AccountSubject.Validate,
		member.InviteAddress.Validate,
		member.Status.Validate,
		member.Generation.Validate,
		member.ActivatedAt.Validate,
	}
	for _, validate := range validators {
		if err := validate(); err != nil {
			return seatMemberError(err)
		}
	}
	return nil
}

func validateSeatMemberState(member BugSeatMember) error {
	switch member.Status {
	case BugSeatMemberStatusActive:
		if member.DeactivatedAt != nil {
			return seatMemberError(core.ErrLicenseContract)
		}
	case BugSeatMemberStatusRemoved:
		if !validSeatEndTime(member.DeactivatedAt, member.ActivatedAt) {
			return seatMemberError(core.ErrLicenseContract)
		}
	default:
		return seatMemberError(core.ErrLicenseContract)
	}
	return nil
}

type BugSeatInvite struct {
	InviteID         BugSeatInviteID      `json:"invite_id"`
	SeatID           BugSeatID            `json:"seat_id"`
	OwnerMemberID    BugSeatMemberID      `json:"owner_member_id"`
	InviteAddress    BugSeatInviteAddress `json:"invite_address"`
	TokenDigest      core.SHA256Hex       `json:"token_digest"`
	CreatedAt        core.UnixNanoTime    `json:"created_at"`
	ExpiresAt        core.UnixNanoTime    `json:"expires_at"`
	ResolvedAt       *core.UnixNanoTime   `json:"resolved_at,omitempty"`
	AcceptedMemberID *BugSeatMemberID     `json:"accepted_member_id,omitempty"`
	Generation       BugSeatGeneration    `json:"generation"`
	Schema           core.SchemaID        `json:"schema"`
	Status           BugSeatInviteStatus  `json:"status"`
}

func (i BugSeatInvite) Validate() error {
	if i.Schema != core.SchemaBugSeatInvite {
		return seatInviteError(core.ErrLicenseContract)
	}
	if err := validateSeatInviteFields(i); err != nil {
		return err
	}
	if !i.ExpiresAt.After(i.CreatedAt) {
		return seatInviteError(core.ErrLicenseContract)
	}
	return validateSeatInviteState(i)
}

func validateSeatInviteFields(invite BugSeatInvite) error {
	validators := []func() error{
		invite.InviteID.Validate,
		invite.SeatID.Validate,
		invite.OwnerMemberID.Validate,
		invite.InviteAddress.Validate,
		invite.TokenDigest.Validate,
		invite.Status.Validate,
		invite.Generation.Validate,
		invite.CreatedAt.Validate,
		invite.ExpiresAt.Validate,
	}
	for _, validate := range validators {
		if err := validate(); err != nil {
			return seatInviteError(err)
		}
	}
	return nil
}

func validateSeatInviteState(invite BugSeatInvite) error {
	switch invite.Status {
	case BugSeatInviteStatusPending:
		if invite.ResolvedAt != nil || invite.AcceptedMemberID != nil {
			return seatInviteError(core.ErrLicenseContract)
		}
	case BugSeatInviteStatusAccepted:
		return validateAcceptedSeatInviteState(invite)
	case BugSeatInviteStatusCancelled:
		if !validSeatResolution(invite, false) {
			return seatInviteError(core.ErrLicenseContract)
		}
	case BugSeatInviteStatusExpired:
		return validateExpiredSeatInviteState(invite)
	default:
		return seatInviteError(core.ErrLicenseContract)
	}
	return nil
}

func validateAcceptedSeatInviteState(invite BugSeatInvite) error {
	if !validSeatResolution(invite, true) || invite.ResolvedAt.After(invite.ExpiresAt) {
		return seatInviteError(core.ErrLicenseContract)
	}
	return nil
}

func validateExpiredSeatInviteState(invite BugSeatInvite) error {
	if !validSeatResolution(invite, false) || invite.ResolvedAt.Before(invite.ExpiresAt) {
		return seatInviteError(core.ErrLicenseContract)
	}
	return nil
}

func validSeatResolution(invite BugSeatInvite, accepted bool) bool {
	if !validSeatEndTime(invite.ResolvedAt, invite.CreatedAt) {
		return false
	}
	if accepted {
		return invite.AcceptedMemberID != nil && invite.AcceptedMemberID.Validate() == nil
	}
	return invite.AcceptedMemberID == nil
}

type BugSeatAssignment struct {
	AssignmentID  BugSeatAssignmentID     `json:"assignment_id"`
	SeatID        BugSeatID               `json:"seat_id"`
	OwnerMemberID BugSeatMemberID         `json:"owner_member_id"`
	MemberID      BugSeatMemberID         `json:"member_id"`
	AssignedAt    core.UnixNanoTime       `json:"assigned_at"`
	EndedAt       *core.UnixNanoTime      `json:"ended_at,omitempty"`
	Generation    BugSeatGeneration       `json:"generation"`
	Schema        core.SchemaID           `json:"schema"`
	Status        BugSeatAssignmentStatus `json:"status"`
}

func (a BugSeatAssignment) Validate() error {
	if a.Schema != core.SchemaBugSeatAssignment {
		return seatAssignmentError(core.ErrLicenseContract)
	}
	if err := validateSeatAssignmentFields(a); err != nil {
		return err
	}
	return validateSeatAssignmentState(a)
}

func validateSeatAssignmentFields(assignment BugSeatAssignment) error {
	validators := []func() error{
		assignment.AssignmentID.Validate,
		assignment.SeatID.Validate,
		assignment.OwnerMemberID.Validate,
		assignment.MemberID.Validate,
		assignment.Status.Validate,
		assignment.Generation.Validate,
		assignment.AssignedAt.Validate,
	}
	for _, validate := range validators {
		if err := validate(); err != nil {
			return seatAssignmentError(err)
		}
	}
	return nil
}

func validateSeatAssignmentState(assignment BugSeatAssignment) error {
	switch assignment.Status {
	case BugSeatAssignmentStatusActive:
		if assignment.EndedAt != nil {
			return seatAssignmentError(core.ErrLicenseContract)
		}
	case BugSeatAssignmentStatusRemoved, BugSeatAssignmentStatusTransferred:
		if !validSeatEndTime(assignment.EndedAt, assignment.AssignedAt) {
			return seatAssignmentError(core.ErrLicenseContract)
		}
	default:
		return seatAssignmentError(core.ErrLicenseContract)
	}
	return nil
}

func ValidateBugSeatMemberTransition(previous, next BugSeatMember) error {
	if err := previous.Validate(); err != nil {
		return seatTransitionError(err)
	}
	if err := next.Validate(); err != nil {
		return seatTransitionError(err)
	}
	if !sameSeatMemberIdentity(previous, next) || !nextSeatGeneration(previous.Generation, next.Generation) {
		return seatTransitionError(core.ErrSeatAssignmentConflict)
	}
	if previous.Status == BugSeatMemberStatusActive && next.Status == BugSeatMemberStatusRemoved {
		return nil
	}
	if previous.Status == BugSeatMemberStatusRemoved && next.Status == BugSeatMemberStatusActive && next.ActivatedAt.After(*previous.DeactivatedAt) {
		return nil
	}
	return seatTransitionError(core.ErrSeatAssignmentConflict)
}

func ValidateBugSeatInviteTransition(previous, next BugSeatInvite) error {
	if err := previous.Validate(); err != nil {
		return seatTransitionError(err)
	}
	if err := next.Validate(); err != nil {
		return seatTransitionError(err)
	}
	if previous.Status != BugSeatInviteStatusPending || next.Status == BugSeatInviteStatusPending {
		return seatTransitionError(core.ErrSeatAssignmentConflict)
	}
	if !sameSeatInviteIdentity(previous, next) || !nextSeatGeneration(previous.Generation, next.Generation) {
		return seatTransitionError(core.ErrSeatAssignmentConflict)
	}
	return nil
}

func ValidateBugSeatAssignmentTransition(previous, next BugSeatAssignment) error {
	if err := previous.Validate(); err != nil {
		return seatTransitionError(err)
	}
	if err := next.Validate(); err != nil {
		return seatTransitionError(err)
	}
	if previous.Status != BugSeatAssignmentStatusActive || next.Status == BugSeatAssignmentStatusActive {
		return seatTransitionError(core.ErrSeatAssignmentConflict)
	}
	if !sameSeatAssignmentIdentity(previous, next) || !nextSeatGeneration(previous.Generation, next.Generation) {
		return seatTransitionError(core.ErrSeatAssignmentConflict)
	}
	return nil
}

func ValidateBugSeatSelfAssignment(owner BugSeatMember, assignment BugSeatAssignment) error {
	if err := validateActiveSeatMember(owner); err != nil {
		return seatTransitionError(err)
	}
	if err := assignment.Validate(); err != nil {
		return seatTransitionError(err)
	}
	if assignment.Status != BugSeatAssignmentStatusActive || assignment.OwnerMemberID != owner.MemberID || assignment.MemberID != owner.MemberID {
		return seatTransitionError(core.ErrSeatAssignmentConflict)
	}
	return nil
}

func ValidateBugSeatInviteAcceptance(invite BugSeatInvite, member BugSeatMember, assignment BugSeatAssignment) error {
	if err := invite.Validate(); err != nil {
		return seatTransitionError(err)
	}
	if err := validateActiveSeatMember(member); err != nil {
		return seatTransitionError(err)
	}
	if err := assignment.Validate(); err != nil {
		return seatTransitionError(err)
	}
	if !acceptedSeatBindingsMatch(invite, member, assignment) {
		return seatTransitionError(core.ErrSeatAssignmentConflict)
	}
	return nil
}

func ValidateBugSeatTransfer(previous, next BugSeatAssignment) error {
	if previous.Status != BugSeatAssignmentStatusTransferred {
		return seatTransitionError(core.ErrSeatAssignmentConflict)
	}
	if err := validateSeatTransferRecords(previous, next); err != nil {
		return err
	}
	if !seatTransferBindingsMatch(previous, next) {
		return seatTransitionError(core.ErrSeatAssignmentConflict)
	}
	return nil
}

func validateSeatTransferRecords(previous, next BugSeatAssignment) error {
	if err := previous.Validate(); err != nil {
		return seatTransitionError(err)
	}
	if err := next.Validate(); err != nil {
		return seatTransitionError(err)
	}
	return nil
}

func seatTransferBindingsMatch(previous, next BugSeatAssignment) bool {
	if next.Status != BugSeatAssignmentStatusActive || previous.SeatID != next.SeatID || previous.OwnerMemberID != next.OwnerMemberID {
		return false
	}
	if previous.MemberID == next.MemberID || previous.AssignmentID == next.AssignmentID || !nextSeatGeneration(previous.Generation, next.Generation) {
		return false
	}
	return previous.EndedAt != nil && previous.EndedAt.Equal(next.AssignedAt)
}

func ValidateBugSeatMemberRecovery(previous, next BugSeatMember, assignment BugSeatAssignment) error {
	if err := ValidateBugSeatMemberTransition(previous, next); err != nil {
		return err
	}
	if previous.Status != BugSeatMemberStatusRemoved || next.Status != BugSeatMemberStatusActive {
		return seatTransitionError(core.ErrSeatAssignmentConflict)
	}
	if err := assignment.Validate(); err != nil {
		return seatTransitionError(err)
	}
	if assignment.Status != BugSeatAssignmentStatusActive || assignment.MemberID != next.MemberID || !assignment.AssignedAt.Equal(next.ActivatedAt) {
		return seatTransitionError(core.ErrSeatAssignmentConflict)
	}
	return nil
}

func validateActiveSeatMember(member BugSeatMember) error {
	if err := member.Validate(); err != nil {
		return err
	}
	if member.Status != BugSeatMemberStatusActive {
		return core.ErrSeatMemberInactive
	}
	return nil
}

func acceptedSeatBindingsMatch(invite BugSeatInvite, member BugSeatMember, assignment BugSeatAssignment) bool {
	if invite.Status != BugSeatInviteStatusAccepted || assignment.Status != BugSeatAssignmentStatusActive || invite.AcceptedMemberID == nil || invite.ResolvedAt == nil {
		return false
	}
	return acceptedSeatMemberBindingsMatch(invite, member) && acceptedSeatAssignmentBindingsMatch(invite, member, assignment)
}

func acceptedSeatMemberBindingsMatch(invite BugSeatInvite, member BugSeatMember) bool {
	return *invite.AcceptedMemberID == member.MemberID && invite.InviteAddress == member.InviteAddress && invite.ResolvedAt.Equal(member.ActivatedAt)
}

func acceptedSeatAssignmentBindingsMatch(invite BugSeatInvite, member BugSeatMember, assignment BugSeatAssignment) bool {
	return invite.SeatID == assignment.SeatID && invite.OwnerMemberID == assignment.OwnerMemberID &&
		member.MemberID == assignment.MemberID && invite.ResolvedAt.Equal(assignment.AssignedAt)
}

func sameSeatMemberIdentity(previous, next BugSeatMember) bool {
	return previous.Schema == next.Schema && previous.MemberID == next.MemberID &&
		previous.AccountSubject == next.AccountSubject && previous.InviteAddress == next.InviteAddress
}

func sameSeatInviteIdentity(previous, next BugSeatInvite) bool {
	return previous.Schema == next.Schema && previous.InviteID == next.InviteID && previous.SeatID == next.SeatID &&
		previous.OwnerMemberID == next.OwnerMemberID && previous.InviteAddress == next.InviteAddress &&
		previous.TokenDigest == next.TokenDigest && previous.CreatedAt.Equal(next.CreatedAt) && previous.ExpiresAt.Equal(next.ExpiresAt)
}

func sameSeatAssignmentIdentity(previous, next BugSeatAssignment) bool {
	return previous.Schema == next.Schema && previous.AssignmentID == next.AssignmentID && previous.SeatID == next.SeatID &&
		previous.OwnerMemberID == next.OwnerMemberID && previous.MemberID == next.MemberID && previous.AssignedAt.Equal(next.AssignedAt)
}

func nextSeatGeneration(previous, next BugSeatGeneration) bool {
	want, err := previous.Next()
	return err == nil && next == want
}

func validSeatEndTime(end *core.UnixNanoTime, start core.UnixNanoTime) bool {
	return end != nil && end.Validate() == nil && !end.Before(start)
}

func seatMemberError(cause error) error {
	return fmt.Errorf(ErrFmtSeatMember, errors.Join(core.ErrLicenseContract, cause))
}

func seatInviteError(cause error) error {
	return fmt.Errorf(ErrFmtSeatInvite, errors.Join(core.ErrLicenseContract, cause))
}

func seatAssignmentError(cause error) error {
	return fmt.Errorf(ErrFmtSeatAssignment, errors.Join(core.ErrLicenseContract, cause))
}

func seatTransitionError(cause error) error {
	return fmt.Errorf(ErrFmtSeatTransition, cause)
}
