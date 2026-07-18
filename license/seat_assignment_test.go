package license

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	testOwnerAddress  = "owner@example.com"
	testMemberAddress = "member@example.com"
)

func TestBugSeatGeneratedScalarsAreCanonicalAndDistinct(t *testing.T) {
	t.Parallel()
	seatOne, err := NewBugSeatID()
	if err != nil {
		t.Fatalf("NewBugSeatID() error = %v", err)
	}
	seatTwo, err := NewBugSeatID()
	if err != nil {
		t.Fatalf("NewBugSeatID() second error = %v", err)
	}
	if seatOne == seatTwo || seatOne.Validate() != nil || seatTwo.Validate() != nil {
		t.Fatalf("generated seat identities are invalid or equal")
	}
	token, err := NewBugSeatInviteToken()
	if err != nil {
		t.Fatalf("NewBugSeatInviteToken() error = %v", err)
	}
	if err := token.Validate(); err != nil {
		t.Fatalf("generated token Validate() error = %v", err)
	}
	digest, err := token.Digest()
	if err != nil || digest.Validate() != nil {
		t.Fatalf("generated token Digest() = %v, %v", digest, err)
	}
	parsed, err := ParseBugSeatInviteToken(token.String())
	if err != nil || parsed != token {
		t.Fatalf("ParseBugSeatInviteToken() = %v, %v", parsed, err)
	}
}

func TestBugSeatIdentityHostileTable(t *testing.T) {
	t.Parallel()
	validDigest := strings.Repeat("a", 64)
	tests := []struct {
		name  string
		value string
	}{
		{name: "empty"},
		{name: "wrong prefix", value: BugSeatInviteIDPrefix + validDigest},
		{name: "short", value: BugSeatIDPrefix + strings.Repeat("a", 63)},
		{name: "long", value: BugSeatIDPrefix + strings.Repeat("a", 65)},
		{name: "uppercase", value: BugSeatIDPrefix + strings.Repeat("A", 64)},
		{name: "non hex", value: BugSeatIDPrefix + strings.Repeat("z", 64)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseBugSeatID(test.value); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("ParseBugSeatID(%q) error = %v, want ErrLicenseContract", test.value, err)
			}
		})
	}
	valid, err := ParseBugSeatID(BugSeatIDPrefix + validDigest)
	if err != nil || valid.String() != BugSeatIDPrefix+validDigest {
		t.Fatalf("ParseBugSeatID(valid) = %v, %v", valid, err)
	}
}

func TestBugSeatInviteAddressHostileTable(t *testing.T) {
	t.Parallel()
	tests := []string{
		"",
		" member@example.com",
		"member@example.com ",
		"Member@example.com",
		"memberexample.com",
		"member@@example.com",
		"@example.com",
		"member@",
		"member name@example.com",
		"member\n@example.com",
		"member@example.com.",
		strings.Repeat("a", BugSeatInviteAddressMax) + "@example.com",
	}
	for _, value := range tests {
		if _, err := ParseBugSeatInviteAddress(value); !errors.Is(err, core.ErrLicenseContract) {
			t.Fatalf("ParseBugSeatInviteAddress(%q) error = %v, want ErrLicenseContract", value, err)
		}
	}
	want := testMemberAddress
	got, err := ParseBugSeatInviteAddress(want)
	if err != nil || got.String() != want {
		t.Fatalf("ParseBugSeatInviteAddress(valid) = %v, %v", got, err)
	}
}

func TestBugSeatScalarJSONRejectsWrongShapesWithoutMutation(t *testing.T) {
	t.Parallel()
	address := mustSeatAddress(t, testMemberAddress)
	for _, raw := range []string{`null`, `1`, `{}`, `"Member@example.com"`} {
		copy := address
		if err := json.Unmarshal([]byte(raw), &copy); !errors.Is(err, core.ErrLicenseContract) {
			t.Fatalf("json.Unmarshal(%s) error = %v, want ErrLicenseContract", raw, err)
		}
		if copy != address {
			t.Fatalf("json.Unmarshal(%s) mutated address", raw)
		}
	}
}

func TestBugSeatGenerationHostileTable(t *testing.T) {
	t.Parallel()
	if err := BugSeatGeneration(0).Validate(); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("zero generation error = %v", err)
	}
	if _, err := BugSeatGeneration(math.MaxUint64).Next(); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("overflow generation error = %v", err)
	}
	if got, err := BugSeatGeneration(1).Next(); err != nil || got != 2 {
		t.Fatalf("generation Next() = %d, %v", got, err)
	}
	for _, raw := range []string{`0`, `-1`, `null`, `"1"`} {
		generation := BugSeatGeneration(7)
		if err := json.Unmarshal([]byte(raw), &generation); !errors.Is(err, core.ErrLicenseContract) {
			t.Fatalf("json.Unmarshal(%s) error = %v, want ErrLicenseContract", raw, err)
		}
		if generation != 7 {
			t.Fatalf("json.Unmarshal(%s) mutated generation", raw)
		}
	}
}

func TestBugSeatStatusesAreClosedCompilerOwnedEnums(t *testing.T) {
	t.Parallel()
	memberCases := []struct {
		status BugSeatMemberStatus
		token  string
	}{
		{status: BugSeatMemberStatusActive, token: BugSeatMemberStatusTokenActive},
		{status: BugSeatMemberStatusRemoved, token: BugSeatMemberStatusTokenRemoved},
	}
	for _, test := range memberCases {
		requireSeatStatusContract(t, test.status, test.token, ParseBugSeatMemberStatus)
	}
	inviteCases := []struct {
		status BugSeatInviteStatus
		token  string
	}{
		{status: BugSeatInviteStatusPending, token: BugSeatInviteStatusTokenPending},
		{status: BugSeatInviteStatusAccepted, token: BugSeatInviteStatusTokenAccepted},
		{status: BugSeatInviteStatusCancelled, token: BugSeatInviteStatusTokenCancelled},
		{status: BugSeatInviteStatusExpired, token: BugSeatInviteStatusTokenExpired},
	}
	for _, test := range inviteCases {
		requireSeatStatusContract(t, test.status, test.token, ParseBugSeatInviteStatus)
	}
	assignmentCases := []struct {
		status BugSeatAssignmentStatus
		token  string
	}{
		{status: BugSeatAssignmentStatusActive, token: BugSeatAssignmentStatusTokenActive},
		{status: BugSeatAssignmentStatusRemoved, token: BugSeatAssignmentStatusTokenRemoved},
		{status: BugSeatAssignmentStatusTransferred, token: BugSeatAssignmentStatusTokenTransfer},
	}
	for _, test := range assignmentCases {
		requireSeatStatusContract(t, test.status, test.token, ParseBugSeatAssignmentStatus)
	}
	if _, err := ParseBugSeatInviteStatus("future"); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("future invite status error = %v", err)
	}
}

func TestBugSeatMemberValidationHostileTable(t *testing.T) {
	t.Parallel()
	valid := activeSeatMember(t, testMemberAddress, 10)
	removedAt := seatTime(20)
	tests := []struct {
		name   string
		mutate func(*BugSeatMember)
	}{
		{name: "wrong schema", mutate: func(m *BugSeatMember) { m.Schema = core.SchemaBugSeatInvite }},
		{name: "missing member", mutate: func(m *BugSeatMember) { m.MemberID = BugSeatMemberID{} }},
		{name: "missing subject", mutate: func(m *BugSeatMember) { m.AccountSubject = BugSeatAccountSubject{} }},
		{name: "missing address", mutate: func(m *BugSeatMember) { m.InviteAddress = BugSeatInviteAddress{} }},
		{name: "invalid status", mutate: func(m *BugSeatMember) { m.Status = 99 }},
		{name: "zero generation", mutate: func(m *BugSeatMember) { m.Generation = 0 }},
		{name: "zero activation", mutate: func(m *BugSeatMember) { m.ActivatedAt = core.UnixNanoTime{} }},
		{name: "active with deactivation", mutate: func(m *BugSeatMember) { m.DeactivatedAt = &removedAt }},
		{name: "removed without deactivation", mutate: func(m *BugSeatMember) { m.Status = BugSeatMemberStatusRemoved }},
		{name: "removed before activation", mutate: func(m *BugSeatMember) {
			before := seatTime(9)
			m.Status = BugSeatMemberStatusRemoved
			m.DeactivatedAt = &before
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			member := valid
			test.mutate(&member)
			if err := member.Validate(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("Validate() error = %v, want ErrLicenseContract", err)
			}
		})
	}
}

func TestBugSeatInviteAndAssignmentValidationHostileTable(t *testing.T) {
	t.Parallel()
	invite := pendingSeatInvite(t)
	resolvedAt := seatTime(15)
	acceptedMemberID := mustSeatMemberID(t, "b")
	invite.Status = BugSeatInviteStatusAccepted
	invite.ResolvedAt = &resolvedAt
	invite.AcceptedMemberID = &acceptedMemberID
	if err := invite.Validate(); err != nil {
		t.Fatalf("accepted invite Validate() error = %v", err)
	}
	badAccepted := invite
	badAccepted.AcceptedMemberID = nil
	if err := badAccepted.Validate(); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("accepted invite without member error = %v", err)
	}
	expired := pendingSeatInvite(t)
	expired.Status = BugSeatInviteStatusExpired
	beforeExpiry := seatTime(15)
	expired.ResolvedAt = &beforeExpiry
	if err := expired.Validate(); err == nil {
		t.Fatalf("expired invite before expiry accepted")
	}
	assignment := activeSeatAssignment(t, mustSeatMemberID(t, "b"), 1, 15)
	if err := assignment.Validate(); err != nil {
		t.Fatalf("active assignment Validate() error = %v", err)
	}
	ended := assignment
	ended.Status = BugSeatAssignmentStatusRemoved
	ended.EndedAt = nil
	if err := ended.Validate(); !errors.Is(err, core.ErrLicenseContract) {
		t.Fatalf("ended assignment without time error = %v", err)
	}
}

func TestBugSeatPurchaserSelfAssignmentAndInviteAcceptance(t *testing.T) {
	t.Parallel()
	owner := activeSeatMember(t, testOwnerAddress, 10)
	self := activeSeatAssignment(t, owner.MemberID, 1, 11)
	self.OwnerMemberID = owner.MemberID
	if err := ValidateBugSeatSelfAssignment(owner, self); err != nil {
		t.Fatalf("ValidateBugSeatSelfAssignment() error = %v", err)
	}
	other := self
	other.MemberID = mustSeatMemberID(t, "b")
	if err := ValidateBugSeatSelfAssignment(owner, other); !errors.Is(err, core.ErrSeatAssignmentConflict) {
		t.Fatalf("wrong self assignment error = %v", err)
	}

	member := activeSeatMember(t, testMemberAddress, 15)
	invite := pendingSeatInvite(t)
	invite.Status = BugSeatInviteStatusAccepted
	invite.Generation = 2
	invite.ResolvedAt = timePointer(15)
	invite.AcceptedMemberID = &member.MemberID
	assignment := activeSeatAssignment(t, member.MemberID, 2, 15)
	if err := ValidateBugSeatInviteAcceptance(invite, member, assignment); err != nil {
		t.Fatalf("ValidateBugSeatInviteAcceptance() error = %v", err)
	}
	wrongSeat := assignment
	wrongSeat.SeatID = mustSeatID(t, "c")
	if err := ValidateBugSeatInviteAcceptance(invite, member, wrongSeat); !errors.Is(err, core.ErrSeatAssignmentConflict) {
		t.Fatalf("wrong-seat acceptance error = %v", err)
	}
}

func TestBugSeatTransitionsRejectReplayForkAndLaundering(t *testing.T) {
	t.Parallel()
	active := activeSeatMember(t, testMemberAddress, 10)
	removed := active
	removed.Status = BugSeatMemberStatusRemoved
	removed.Generation = 2
	removed.DeactivatedAt = timePointer(20)
	if err := ValidateBugSeatMemberTransition(active, removed); err != nil {
		t.Fatalf("active-to-removed transition error = %v", err)
	}
	if err := ValidateBugSeatMemberTransition(removed, removed); !errors.Is(err, core.ErrSeatAssignmentConflict) {
		t.Fatalf("replayed member transition error = %v", err)
	}
	recovered := active
	recovered.Generation = 3
	recovered.ActivatedAt = seatTime(21)
	if err := ValidateBugSeatMemberTransition(removed, recovered); err != nil {
		t.Fatalf("removed-to-active transition error = %v", err)
	}
	recoveryAssignment := activeSeatAssignment(t, recovered.MemberID, 3, 21)
	if err := ValidateBugSeatMemberRecovery(removed, recovered, recoveryAssignment); err != nil {
		t.Fatalf("ValidateBugSeatMemberRecovery() error = %v", err)
	}
	wrongRecovery := recoveryAssignment
	wrongRecovery.MemberID = mustSeatMemberID(t, "b")
	if err := ValidateBugSeatMemberRecovery(removed, recovered, wrongRecovery); !errors.Is(err, core.ErrSeatAssignmentConflict) {
		t.Fatalf("wrong-member recovery error = %v", err)
	}

	pending := pendingSeatInvite(t)
	accepted := pending
	accepted.Status = BugSeatInviteStatusAccepted
	accepted.Generation = 2
	accepted.ResolvedAt = timePointer(15)
	acceptedMember := mustSeatMemberID(t, "b")
	accepted.AcceptedMemberID = &acceptedMember
	if err := ValidateBugSeatInviteTransition(pending, accepted); err != nil {
		t.Fatalf("pending-to-accepted transition error = %v", err)
	}
	tampered := accepted
	tampered.TokenDigest = mustSHA256(t, "c")
	if err := ValidateBugSeatInviteTransition(pending, tampered); !errors.Is(err, core.ErrSeatAssignmentConflict) {
		t.Fatalf("tampered invite transition error = %v", err)
	}

	previousActive := activeSeatAssignment(t, active.MemberID, 1, 11)
	transferred := previousActive
	transferred.Status = BugSeatAssignmentStatusTransferred
	transferred.Generation = 2
	transferred.EndedAt = timePointer(30)
	if err := ValidateBugSeatAssignmentTransition(previousActive, transferred); err != nil {
		t.Fatalf("active-to-transferred transition error = %v", err)
	}
	next := activeSeatAssignment(t, acceptedMember, 3, 30)
	next.AssignmentID = mustSeatAssignmentID(t, "c")
	if err := ValidateBugSeatTransfer(transferred, next); err != nil {
		t.Fatalf("ValidateBugSeatTransfer() error = %v", err)
	}
	fork := next
	fork.Generation = 2
	if err := ValidateBugSeatTransfer(transferred, fork); !errors.Is(err, core.ErrSeatAssignmentConflict) {
		t.Fatalf("generation fork transfer error = %v", err)
	}
}

func TestBugSeatStrictJSONRejectsUnknownAndDuplicateFields(t *testing.T) {
	t.Parallel()
	member := activeSeatMember(t, testMemberAddress, 10)
	raw, err := json.Marshal(member)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	decoded, err := core.DecodeStrictJSON[BugSeatMember](raw)
	if err != nil || decoded != member {
		t.Fatalf("DecodeStrictJSON() = %+v, %v", decoded, err)
	}
	unknown := strings.TrimSuffix(string(raw), "}") + `,"future":true}`
	if _, err := core.DecodeStrictJSON[BugSeatMember]([]byte(unknown)); !errors.Is(err, core.ErrJSONContract) {
		t.Fatalf("unknown field error = %v, want ErrJSONContract", err)
	}
	duplicate := strings.TrimSuffix(string(raw), "}") + `,"generation":2}`
	if _, err := core.DecodeStrictJSON[BugSeatMember]([]byte(duplicate)); !errors.Is(err, core.ErrJSONContract) {
		t.Fatalf("duplicate field error = %v, want ErrJSONContract", err)
	}
}

type seatStatus interface {
	~uint8
	String() string
	Validate() error
}

func requireSeatStatusContract[T seatStatus](t *testing.T, status T, token string, parse func(string) (T, error)) {
	t.Helper()
	if err := status.Validate(); err != nil || status.String() != token {
		t.Fatalf("status %d = %q, %v", status, status.String(), err)
	}
	parsed, err := parse(token)
	if err != nil || parsed != status {
		t.Fatalf("parse(%q) = %d, %v", token, parsed, err)
	}
}

func activeSeatMember(t *testing.T, address string, activatedAt int64) BugSeatMember {
	t.Helper()
	return BugSeatMember{
		Schema:         core.SchemaBugSeatMember,
		MemberID:       mustSeatMemberID(t, "a"),
		AccountSubject: mustSeatSubject(t, "account-a"),
		InviteAddress:  mustSeatAddress(t, address),
		Status:         BugSeatMemberStatusActive,
		Generation:     1,
		ActivatedAt:    seatTime(activatedAt),
	}
}

func pendingSeatInvite(t *testing.T) BugSeatInvite {
	t.Helper()
	return BugSeatInvite{
		Schema:        core.SchemaBugSeatInvite,
		InviteID:      mustSeatInviteID(t, "d"),
		SeatID:        mustSeatID(t, "a"),
		OwnerMemberID: mustSeatMemberID(t, "a"),
		InviteAddress: mustSeatAddress(t, testMemberAddress),
		TokenDigest:   mustSHA256(t, "e"),
		Status:        BugSeatInviteStatusPending,
		Generation:    1,
		CreatedAt:     seatTime(10),
		ExpiresAt:     seatTime(20),
	}
}

func activeSeatAssignment(t *testing.T, memberID BugSeatMemberID, generation BugSeatGeneration, assignedAt int64) BugSeatAssignment {
	t.Helper()
	return BugSeatAssignment{
		Schema:        core.SchemaBugSeatAssignment,
		AssignmentID:  mustSeatAssignmentID(t, "f"),
		SeatID:        mustSeatID(t, "a"),
		OwnerMemberID: mustSeatMemberID(t, "a"),
		MemberID:      memberID,
		Status:        BugSeatAssignmentStatusActive,
		Generation:    generation,
		AssignedAt:    seatTime(assignedAt),
	}
}

func mustSeatID(t *testing.T, digit string) BugSeatID {
	t.Helper()
	value, err := ParseBugSeatID(BugSeatIDPrefix + strings.Repeat(digit, 64))
	if err != nil {
		t.Fatalf("ParseBugSeatID() error = %v", err)
	}
	return value
}

func mustSeatMemberID(t *testing.T, digit string) BugSeatMemberID {
	t.Helper()
	value, err := ParseBugSeatMemberID(BugSeatMemberIDPrefix + strings.Repeat(digit, 64))
	if err != nil {
		t.Fatalf("ParseBugSeatMemberID() error = %v", err)
	}
	return value
}

func mustSeatInviteID(t *testing.T, digit string) BugSeatInviteID {
	t.Helper()
	value, err := ParseBugSeatInviteID(BugSeatInviteIDPrefix + strings.Repeat(digit, 64))
	if err != nil {
		t.Fatalf("ParseBugSeatInviteID() error = %v", err)
	}
	return value
}

func mustSeatAssignmentID(t *testing.T, digit string) BugSeatAssignmentID {
	t.Helper()
	value, err := ParseBugSeatAssignmentID(BugSeatAssignmentIDPrefix + strings.Repeat(digit, 64))
	if err != nil {
		t.Fatalf("ParseBugSeatAssignmentID() error = %v", err)
	}
	return value
}

func mustSeatSubject(t *testing.T, value string) BugSeatAccountSubject {
	t.Helper()
	parsed, err := ParseBugSeatAccountSubject(value)
	if err != nil {
		t.Fatalf("ParseBugSeatAccountSubject() error = %v", err)
	}
	return parsed
}

func mustSeatAddress(t *testing.T, value string) BugSeatInviteAddress {
	t.Helper()
	parsed, err := ParseBugSeatInviteAddress(value)
	if err != nil {
		t.Fatalf("ParseBugSeatInviteAddress() error = %v", err)
	}
	return parsed
}

func mustSHA256(t *testing.T, digit string) core.SHA256Hex {
	t.Helper()
	value, err := core.ParseSHA256Hex(strings.Repeat(digit, 64))
	if err != nil {
		t.Fatalf("ParseSHA256Hex() error = %v", err)
	}
	return value
}

func seatTime(nanos int64) core.UnixNanoTime {
	return core.UnixNanoTimeFromInt64(nanos)
}

func timePointer(nanos int64) *core.UnixNanoTime {
	value := seatTime(nanos)
	return &value
}
