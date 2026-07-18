package license

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	BugSeatIDPrefix           = "bug-seat-"
	BugSeatMemberIDPrefix     = "bug-member-"
	BugSeatInviteIDPrefix     = "bug-invite-"
	BugSeatAssignmentIDPrefix = "bug-assignment-"
	BugSeatInviteTokenPrefix  = "bug-invite-token-"
	BugSeatIdentityBytes      = 32
	BugSeatInviteTokenBytes   = 32
	BugSeatInviteAddressMax   = 254
)

type BugSeatID struct{ value string }
type BugSeatMemberID struct{ value string }
type BugSeatInviteID struct{ value string }
type BugSeatAssignmentID struct{ value string }
type BugSeatAccountSubject struct{ value string }
type BugSeatInviteAddress struct{ value string }
type BugSeatInviteToken struct{ value string }

func NewBugSeatID() (BugSeatID, error) {
	value, err := newBugSeatIdentity(BugSeatIDPrefix)
	return BugSeatID{value: value}, err
}

func ParseBugSeatID(value string) (BugSeatID, error) {
	if err := validateBugSeatIdentity(value, BugSeatIDPrefix); err != nil {
		return BugSeatID{}, err
	}
	return BugSeatID{value: value}, nil
}

func (id BugSeatID) String() string { return id.value }
func (id BugSeatID) IsZero() bool   { return id.value == "" }
func (id BugSeatID) Validate() error {
	_, err := ParseBugSeatID(id.value)
	return err
}

func (id BugSeatID) MarshalJSON() ([]byte, error) { return marshalSeatScalar(id.value, id.Validate) }
func (id *BugSeatID) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalSeatScalar(data, ErrFmtSeatIdentity, ParseBugSeatID)
	if err == nil {
		*id = parsed
	}
	return err
}

func NewBugSeatMemberID() (BugSeatMemberID, error) {
	value, err := newBugSeatIdentity(BugSeatMemberIDPrefix)
	return BugSeatMemberID{value: value}, err
}

func ParseBugSeatMemberID(value string) (BugSeatMemberID, error) {
	if err := validateBugSeatIdentity(value, BugSeatMemberIDPrefix); err != nil {
		return BugSeatMemberID{}, err
	}
	return BugSeatMemberID{value: value}, nil
}

func (id BugSeatMemberID) String() string { return id.value }
func (id BugSeatMemberID) IsZero() bool   { return id.value == "" }
func (id BugSeatMemberID) Validate() error {
	_, err := ParseBugSeatMemberID(id.value)
	return err
}

func (id BugSeatMemberID) MarshalJSON() ([]byte, error) {
	return marshalSeatScalar(id.value, id.Validate)
}
func (id *BugSeatMemberID) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalSeatScalar(data, ErrFmtSeatIdentity, ParseBugSeatMemberID)
	if err == nil {
		*id = parsed
	}
	return err
}

func NewBugSeatInviteID() (BugSeatInviteID, error) {
	value, err := newBugSeatIdentity(BugSeatInviteIDPrefix)
	return BugSeatInviteID{value: value}, err
}

func ParseBugSeatInviteID(value string) (BugSeatInviteID, error) {
	if err := validateBugSeatIdentity(value, BugSeatInviteIDPrefix); err != nil {
		return BugSeatInviteID{}, err
	}
	return BugSeatInviteID{value: value}, nil
}

func (id BugSeatInviteID) String() string { return id.value }
func (id BugSeatInviteID) IsZero() bool   { return id.value == "" }
func (id BugSeatInviteID) Validate() error {
	_, err := ParseBugSeatInviteID(id.value)
	return err
}

func (id BugSeatInviteID) MarshalJSON() ([]byte, error) {
	return marshalSeatScalar(id.value, id.Validate)
}
func (id *BugSeatInviteID) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalSeatScalar(data, ErrFmtSeatIdentity, ParseBugSeatInviteID)
	if err == nil {
		*id = parsed
	}
	return err
}

func NewBugSeatAssignmentID() (BugSeatAssignmentID, error) {
	value, err := newBugSeatIdentity(BugSeatAssignmentIDPrefix)
	return BugSeatAssignmentID{value: value}, err
}

func ParseBugSeatAssignmentID(value string) (BugSeatAssignmentID, error) {
	if err := validateBugSeatIdentity(value, BugSeatAssignmentIDPrefix); err != nil {
		return BugSeatAssignmentID{}, err
	}
	return BugSeatAssignmentID{value: value}, nil
}

func (id BugSeatAssignmentID) String() string { return id.value }
func (id BugSeatAssignmentID) IsZero() bool   { return id.value == "" }
func (id BugSeatAssignmentID) Validate() error {
	_, err := ParseBugSeatAssignmentID(id.value)
	return err
}

func (id BugSeatAssignmentID) MarshalJSON() ([]byte, error) {
	return marshalSeatScalar(id.value, id.Validate)
}
func (id *BugSeatAssignmentID) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalSeatScalar(data, ErrFmtSeatIdentity, ParseBugSeatAssignmentID)
	if err == nil {
		*id = parsed
	}
	return err
}

func ParseBugSeatAccountSubject(value string) (BugSeatAccountSubject, error) {
	if err := core.ValidateOpaqueToken(value, core.OpaqueTokenDefaultMaxRunes); err != nil {
		return BugSeatAccountSubject{}, fmt.Errorf(ErrFmtSeatIdentity, core.ErrLicenseContract)
	}
	return BugSeatAccountSubject{value: value}, nil
}

func (subject BugSeatAccountSubject) String() string { return subject.value }
func (subject BugSeatAccountSubject) IsZero() bool   { return subject.value == "" }
func (subject BugSeatAccountSubject) Validate() error {
	_, err := ParseBugSeatAccountSubject(subject.value)
	return err
}

func (subject BugSeatAccountSubject) MarshalJSON() ([]byte, error) {
	return marshalSeatScalar(subject.value, subject.Validate)
}
func (subject *BugSeatAccountSubject) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalSeatScalar(data, ErrFmtSeatIdentity, ParseBugSeatAccountSubject)
	if err == nil {
		*subject = parsed
	}
	return err
}

func ParseBugSeatInviteAddress(value string) (BugSeatInviteAddress, error) {
	if value != strings.TrimSpace(value) || value != strings.ToLower(value) || len(value) > BugSeatInviteAddressMax {
		return BugSeatInviteAddress{}, fmt.Errorf(ErrFmtSeatInviteAddress, core.ErrLicenseContract)
	}
	if strings.Count(value, "@") != 1 || strings.ContainsFunc(value, invalidSeatInviteAddressRune) {
		return BugSeatInviteAddress{}, fmt.Errorf(ErrFmtSeatInviteAddress, core.ErrLicenseContract)
	}
	at := strings.IndexByte(value, '@')
	if at <= 0 || at == len(value)-1 {
		return BugSeatInviteAddress{}, fmt.Errorf(ErrFmtSeatInviteAddress, core.ErrLicenseContract)
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value {
		return BugSeatInviteAddress{}, fmt.Errorf(ErrFmtSeatInviteAddress, core.ErrLicenseContract)
	}
	return BugSeatInviteAddress{value: value}, nil
}

func invalidSeatInviteAddressRune(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsControl(r)
}

func (address BugSeatInviteAddress) String() string { return address.value }
func (address BugSeatInviteAddress) IsZero() bool   { return address.value == "" }
func (address BugSeatInviteAddress) Validate() error {
	_, err := ParseBugSeatInviteAddress(address.value)
	return err
}

func (address BugSeatInviteAddress) MarshalJSON() ([]byte, error) {
	return marshalSeatScalar(address.value, address.Validate)
}
func (address *BugSeatInviteAddress) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalSeatScalar(data, ErrFmtSeatInviteAddress, ParseBugSeatInviteAddress)
	if err == nil {
		*address = parsed
	}
	return err
}

func NewBugSeatInviteToken() (BugSeatInviteToken, error) {
	bytes := make([]byte, BugSeatInviteTokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return BugSeatInviteToken{}, fmt.Errorf(ErrFmtSeatInviteToken, errors.Join(core.ErrLicenseContract, err))
	}
	return BugSeatInviteToken{value: BugSeatInviteTokenPrefix + base64.RawURLEncoding.EncodeToString(bytes)}, nil
}

func ParseBugSeatInviteToken(value string) (BugSeatInviteToken, error) {
	if !strings.HasPrefix(value, BugSeatInviteTokenPrefix) {
		return BugSeatInviteToken{}, fmt.Errorf(ErrFmtSeatInviteToken, core.ErrLicenseContract)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, BugSeatInviteTokenPrefix))
	if err != nil || len(decoded) != BugSeatInviteTokenBytes {
		return BugSeatInviteToken{}, fmt.Errorf(ErrFmtSeatInviteToken, core.ErrLicenseContract)
	}
	if value != BugSeatInviteTokenPrefix+base64.RawURLEncoding.EncodeToString(decoded) {
		return BugSeatInviteToken{}, fmt.Errorf(ErrFmtSeatInviteToken, core.ErrLicenseContract)
	}
	return BugSeatInviteToken{value: value}, nil
}

func (token BugSeatInviteToken) String() string { return token.value }
func (token BugSeatInviteToken) IsZero() bool   { return token.value == "" }
func (token BugSeatInviteToken) Validate() error {
	_, err := ParseBugSeatInviteToken(token.value)
	return err
}

func (token BugSeatInviteToken) Digest() (core.SHA256Hex, error) {
	if err := token.Validate(); err != nil {
		return core.SHA256Hex{}, err
	}
	digest := sha256.Sum256([]byte(token.value))
	return core.ParseSHA256Hex(hex.EncodeToString(digest[:]))
}

func (token BugSeatInviteToken) MarshalJSON() ([]byte, error) {
	return marshalSeatScalar(token.value, token.Validate)
}
func (token *BugSeatInviteToken) UnmarshalJSON(data []byte) error {
	parsed, err := unmarshalSeatScalar(data, ErrFmtSeatInviteToken, ParseBugSeatInviteToken)
	if err == nil {
		*token = parsed
	}
	return err
}

func newBugSeatIdentity(prefix string) (string, error) {
	bytes := make([]byte, BugSeatIdentityBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf(ErrFmtSeatIdentity, errors.Join(core.ErrLicenseContract, err))
	}
	return prefix + hex.EncodeToString(bytes), nil
}

func validateBugSeatIdentity(value, prefix string) error {
	if !strings.HasPrefix(value, prefix) {
		return fmt.Errorf(ErrFmtSeatIdentity, core.ErrLicenseContract)
	}
	digest := strings.TrimPrefix(value, prefix)
	if len(digest) != hex.EncodedLen(BugSeatIdentityBytes) {
		return fmt.Errorf(ErrFmtSeatIdentity, core.ErrLicenseContract)
	}
	if _, err := core.ParseSHA256Hex(digest); err != nil {
		return fmt.Errorf(ErrFmtSeatIdentity, core.ErrLicenseContract)
	}
	return nil
}

func marshalSeatScalar(value string, validate func() error) ([]byte, error) {
	if err := validate(); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func unmarshalSeatScalar[T any](data []byte, format string, parse func(string) (T, error)) (T, error) {
	var zero T
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return zero, fmt.Errorf(format, core.ErrLicenseContract)
	}
	return parse(value)
}
