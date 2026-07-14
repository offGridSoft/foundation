package custody

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	ULIDTextLen                = 26
	ULIDMaxFirstRune           = '7'
	ULIDUpperCrockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	ErrFmtLedgerSeq            = "custody.LedgerSeq: %w"
)

type CustomerID struct {
	value string
}

func ParseCustomerID(value string) (CustomerID, error) {
	if !validULIDText(value) {
		return CustomerID{}, fmt.Errorf(ErrFmtULID, core.ErrCustodyContract)
	}
	return CustomerID{value: value}, nil
}

func (id CustomerID) String() string {
	return id.value
}

func (id CustomerID) Validate() error {
	_, err := ParseCustomerID(id.value)
	return err
}

func (id CustomerID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

func (id *CustomerID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtULID, core.ErrCustodyContract)
	}
	parsed, err := ParseCustomerID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

type SessionID struct {
	value string
}

type ReceiptID struct {
	value string
}

func ParseReceiptID(value string) (ReceiptID, error) {
	if !validULIDText(value) {
		return ReceiptID{}, fmt.Errorf(ErrFmtULID, core.ErrCustodyContract)
	}
	return ReceiptID{value: value}, nil
}

func (id ReceiptID) String() string {
	return id.value
}

func (id ReceiptID) Validate() error {
	_, err := ParseReceiptID(id.value)
	return err
}

func (id ReceiptID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

func (id *ReceiptID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtULID, core.ErrCustodyContract)
	}
	parsed, err := ParseReceiptID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

func ParseSessionID(value string) (SessionID, error) {
	if !validULIDText(value) {
		return SessionID{}, fmt.Errorf(ErrFmtULID, core.ErrCustodyContract)
	}
	return SessionID{value: value}, nil
}

func (id SessionID) String() string {
	return id.value
}

func (id SessionID) Validate() error {
	_, err := ParseSessionID(id.value)
	return err
}

func (id SessionID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

func (id *SessionID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtULID, core.ErrCustodyContract)
	}
	parsed, err := ParseSessionID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

func validULIDText(value string) bool {
	if len(value) != ULIDTextLen {
		return false
	}
	for index, r := range value {
		if index == 0 && r > ULIDMaxFirstRune {
			return false
		}
		if !validULIDRune(r) {
			return false
		}
	}
	return true
}

type LedgerSeq int64

func NewLedgerSeq(value int64) (LedgerSeq, error) {
	seq := LedgerSeq(value)
	if err := seq.Validate(); err != nil {
		return 0, err
	}
	return seq, nil
}

func (s LedgerSeq) Int64() int64 {
	return int64(s)
}

func (s LedgerSeq) Validate() error {
	if s < 1 {
		return fmt.Errorf(ErrFmtLedgerSeq, core.ErrCustodyContract)
	}
	return nil
}

func (s LedgerSeq) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(int64(s))
}

func (s *LedgerSeq) UnmarshalJSON(data []byte) error {
	var value int64
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtLedgerSeq, core.ErrCustodyContract)
	}
	parsed, err := NewLedgerSeq(value)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func validULIDRune(r rune) bool {
	return strings.ContainsRune(ULIDUpperCrockfordAlphabet, r)
}

type ArtifactName struct {
	value string
}

func ParseArtifactName(value string) (ArtifactName, error) {
	if err := core.ValidateFileNameToken(value, core.FileNameTokenMaxRunes); err != nil {
		return ArtifactName{}, fmt.Errorf(ErrFmtArtifactName, errors.Join(core.ErrCustodyContract, err))
	}
	return ArtifactName{value: value}, nil
}

func (n ArtifactName) String() string {
	return n.value
}

func (n ArtifactName) Validate() error {
	_, err := ParseArtifactName(n.value)
	return err
}

func (n ArtifactName) MarshalJSON() ([]byte, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(n.value)
}

func (n *ArtifactName) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtArtifactName, core.ErrCustodyContract)
	}
	parsed, err := ParseArtifactName(value)
	if err != nil {
		return err
	}
	*n = parsed
	return nil
}

type ObjectPath struct {
	value string
}

func ParseObjectPath(value string) (ObjectPath, error) {
	if err := core.ValidatePathToken(value, core.PathTokenMaxRunes); err != nil {
		return ObjectPath{}, fmt.Errorf(ErrFmtObjectPath, errors.Join(core.ErrCustodyContract, err))
	}
	return ObjectPath{value: value}, nil
}

func (p ObjectPath) String() string {
	return p.value
}

func (p ObjectPath) Validate() error {
	_, err := ParseObjectPath(p.value)
	return err
}

func (p ObjectPath) ValidateWitnessIdentity(customer CustomerID, bundleRoot core.BLAKE3Hex, artifact ArtifactName) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if err := customer.Validate(); err != nil {
		return err
	}
	if err := bundleRoot.Validate(); err != nil {
		return err
	}
	if err := artifact.Validate(); err != nil {
		return err
	}
	parts := strings.Split(p.value, "/")
	if !matchesWitnessObjectIdentity(parts, customer, bundleRoot, artifact) {
		return fmt.Errorf(ErrFmtObjectPath, core.ErrCustodyContract)
	}
	if !validCustodyDatePath(parts[2], parts[3]) {
		return fmt.Errorf(ErrFmtObjectPath, core.ErrCustodyContract)
	}
	return nil
}

func matchesWitnessObjectIdentity(parts []string, customer CustomerID, bundleRoot core.BLAKE3Hex, artifact ArtifactName) bool {
	return len(parts) == 6 &&
		parts[0] == core.WitnessCustodyPathRoot &&
		parts[1] == customer.String() &&
		parts[4] == bundleRoot.String() &&
		parts[5] == artifact.String()
}

func validCustodyDatePath(year, month string) bool {
	if len(year) != core.CustodyYearTextLength || len(month) != core.CustodyMonthTextLength {
		return false
	}
	if !decimalDigitsOnly(year) || !decimalDigitsOnly(month) {
		return false
	}
	yearNumber, yearErr := strconv.Atoi(year)
	monthNumber, monthErr := strconv.Atoi(month)
	return yearErr == nil && monthErr == nil && yearNumber > 0 && monthNumber >= core.CustodyMonthMinimum && monthNumber <= core.CustodyMonthMaximum
}

func decimalDigitsOnly(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return value != ""
}

func (p ObjectPath) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.value)
}

func (p *ObjectPath) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtObjectPath, core.ErrCustodyContract)
	}
	parsed, err := ParseObjectPath(value)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

// Field order is signature-load-bearing when nested inside ReceiptBody.
type ReleaseIdentity struct {
	Version     core.ProductVersion `json:"version"`
	Commit      core.BuildCommit    `json:"commit"`
	ManifestSHA core.SHA256Hex      `json:"tool_manifest_sha"`
}

func (r ReleaseIdentity) Validate() error {
	if err := r.Version.Validate(); err != nil {
		return fmt.Errorf(ErrFmtRelease, err)
	}
	if err := r.Commit.Validate(); err != nil {
		return fmt.Errorf(ErrFmtRelease, err)
	}
	if err := r.ManifestSHA.Validate(); err != nil {
		return fmt.Errorf(ErrFmtRelease, err)
	}
	return nil
}

type OpenLeaseRef struct {
	LeaseID           core.LeaseID           `json:"lease_id"`
	DeviceFingerprint core.DeviceFingerprint `json:"device_fingerprint"`
}

func (r OpenLeaseRef) Validate() error {
	if err := r.LeaseID.Validate(); err != nil {
		return err
	}
	return r.DeviceFingerprint.Validate()
}

type Generation struct {
	value string
}

func ParseGeneration(value string) (Generation, error) {
	if err := core.ValidateOpaqueToken(value, core.OpaqueTokenDefaultMaxRunes); err != nil {
		return Generation{}, fmt.Errorf(ErrFmtGeneration, errors.Join(core.ErrCustodyContract, err))
	}
	return Generation{value: value}, nil
}

func (g Generation) String() string {
	return g.value
}

func (g Generation) Validate() error {
	_, err := ParseGeneration(g.value)
	return err
}

func (g Generation) MarshalJSON() ([]byte, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(g.value)
}

func (g *Generation) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtGeneration, core.ErrCustodyContract)
	}
	parsed, err := ParseGeneration(value)
	if err != nil {
		return err
	}
	*g = parsed
	return nil
}
