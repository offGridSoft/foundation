package license

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const BugWriterRevocationDeliveryMax = 256

// CheckInResponseBody is the exact compiler-owned response contract consumed
// by the check-in transport. Products expose concrete response structures;
// there is no optional cross-product field bag.
type CheckInResponseBody interface {
	core.Validatable
	Verify(core.SigningKeyring) error
	VerifyRequestNonce(CheckInNonce) error
}

// CheckInDecision owns the mutually exclusive grant/refusal state shared by
// every product check-in response.
type CheckInDecision struct {
	Remediation Remediation `json:"remediation"`
	Refusal     Refusal     `json:"refusal"`
	Granted     bool        `json:"granted"`
}

func (d CheckInDecision) Validate() error {
	if d.Granted {
		if d.Refusal != RefusalNone || d.Remediation != RemediationNone {
			return checkInResponseError(core.ErrLicenseContract)
		}
		return nil
	}
	if err := d.Refusal.Validate(); err != nil || d.Refusal == RefusalNone {
		return checkInResponseError(core.ErrLicenseContract)
	}
	want, err := RemediationForRefusal(d.Refusal)
	if err != nil || d.Remediation != want {
		return checkInResponseError(core.ErrLicenseContract)
	}
	return nil
}

type BugCheckInResponseBody struct {
	Grant             *BugCheckInGrant       `json:"grant,omitempty"`
	WriterRevocations BugWriterRevocationSet `json:"writer_revocations"`
	Decision          CheckInDecision        `json:"decision"`
	RequestNonce      CheckInNonce           `json:"request_nonce"`
	Schema            core.SchemaID          `json:"schema"`
}

func (b BugCheckInResponseBody) Validate() error {
	if b.Schema != core.SchemaBugCheckInResponse {
		return checkInResponseError(core.ErrLicenseContract)
	}
	if err := b.RequestNonce.Validate(); err != nil {
		return checkInResponseError(err)
	}
	if err := b.Decision.Validate(); err != nil {
		return err
	}
	if b.Decision.Granted != (b.Grant != nil) {
		return checkInResponseError(core.ErrLicenseContract)
	}
	if b.Grant != nil {
		if err := b.Grant.Validate(); err != nil {
			return checkInResponseError(err)
		}
	}
	return b.WriterRevocations.Validate()
}

func (b BugCheckInResponseBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRequestNonce, b.RequestNonce)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDecision, b.Decision)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldGrant, b.Grant)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldWriterRevocations, b.WriterRevocations)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (b BugCheckInResponseBody) SigningSchema() core.SchemaID { return b.Schema }

func (b BugCheckInResponseBody) MarshalJSON() ([]byte, error) {
	return b.Canonical(nil)
}

type BugCheckInResponse struct {
	Authority core.Signed[BugCheckInResponseBody] `json:"authority"`
}

// APIBody lets the response satisfy lfw/api.Body structurally without a
// dependency from Foundation into the transport package.
func (BugCheckInResponse) APIBody() {}

func (r BugCheckInResponse) Validate() error {
	return checkInResponseErrorOptional(r.Authority.Validate())
}

func (r BugCheckInResponse) Verify(keyring core.SigningKeyring) error {
	if err := r.Authority.Verify(keyring); err != nil {
		return checkInResponseError(err)
	}
	body := r.Authority.Body
	if body.Grant != nil {
		if err := body.Grant.Verify(keyring); err != nil {
			return checkInResponseError(err)
		}
	}
	return body.WriterRevocations.Verify(keyring)
}

func (r BugCheckInResponse) VerifyRequestNonce(nonce CheckInNonce) error {
	if err := nonce.Validate(); err != nil || r.Authority.Body.RequestNonce != nonce {
		return checkInResponseError(core.ErrCheckInNonce)
	}
	return nil
}

// BugWriterRevocationSet is the bounded, deterministic delivery and
// persistence shape for server-signed Bug writer cutoffs.
type BugWriterRevocationSet struct {
	Values []core.Signed[BugWriterRevocationBody] `json:"values"`
}

func (s BugWriterRevocationSet) Validate() error {
	if len(s.Values) > BugWriterRevocationDeliveryMax {
		return checkInResponseError(core.ErrLicenseContract)
	}
	var prior core.SigningKeyID
	for index, value := range s.Values {
		if err := value.Validate(); err != nil {
			return checkInResponseError(err)
		}
		current := value.Body.WriterKeyID
		if index > 0 && current.String() <= prior.String() {
			return checkInResponseError(core.ErrLicenseContract)
		}
		prior = current
	}
	return nil
}

func (s BugWriterRevocationSet) Verify(keyring core.SigningKeyring) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if err := keyring.Validate(); err != nil {
		return checkInResponseError(err)
	}
	for _, value := range s.Values {
		if err := value.Verify(keyring); err != nil {
			return checkInResponseError(err)
		}
	}
	return nil
}

// Merge preserves the earliest signed cutoff ever observed for each writer.
// An earlier server cutoff broadens revocation and therefore replaces a later
// one; a later delivery can never narrow already-revoked history.
func (s BugWriterRevocationSet) Merge(incoming BugWriterRevocationSet) (BugWriterRevocationSet, error) {
	if err := s.Validate(); err != nil {
		return BugWriterRevocationSet{}, err
	}
	if err := incoming.Validate(); err != nil {
		return BugWriterRevocationSet{}, err
	}
	merged := BugWriterRevocationSet{Values: make([]core.Signed[BugWriterRevocationBody], 0, len(s.Values)+len(incoming.Values))}
	left, right := 0, 0
	for left < len(s.Values) && right < len(incoming.Values) {
		leftValue, rightValue := s.Values[left], incoming.Values[right]
		switch {
		case leftValue.Body.WriterKeyID.String() < rightValue.Body.WriterKeyID.String():
			merged.Values = append(merged.Values, leftValue)
			left++
		case rightValue.Body.WriterKeyID.String() < leftValue.Body.WriterKeyID.String():
			merged.Values = append(merged.Values, rightValue)
			right++
		default:
			merged.Values = append(merged.Values, earliestWriterRevocation(leftValue, rightValue))
			left++
			right++
		}
	}
	merged.Values = append(merged.Values, s.Values[left:]...)
	merged.Values = append(merged.Values, incoming.Values[right:]...)
	if err := merged.Validate(); err != nil {
		return BugWriterRevocationSet{}, err
	}
	return merged, nil
}

func earliestWriterRevocation(left, right core.Signed[BugWriterRevocationBody]) core.Signed[BugWriterRevocationBody] {
	if right.Body.RevokedAt.Before(left.Body.RevokedAt) {
		return right
	}
	return left
}

func (s BugWriterRevocationSet) VerifyWriterAllowed(writerKeyID core.SigningKeyID, occurredAt core.UnixNanoTime) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if err := writerKeyID.Validate(); err != nil {
		return checkInResponseError(err)
	}
	if err := core.ValidateRequiredUnixNanoTime(occurredAt); err != nil || occurredAt.IsZero() {
		return checkInResponseError(err)
	}
	for _, value := range s.Values {
		if value.Body.WriterKeyID.String() > writerKeyID.String() {
			return nil
		}
		if err := value.Body.VerifyWriterAllowed(writerKeyID, occurredAt); err != nil {
			return checkInResponseError(err)
		}
	}
	return nil
}

type WitnessCheckInResponseBody struct {
	Grant        *WitnessCheckInGrant `json:"grant,omitempty"`
	Decision     CheckInDecision      `json:"decision"`
	RequestNonce CheckInNonce         `json:"request_nonce"`
	Schema       core.SchemaID        `json:"schema"`
}

func (b WitnessCheckInResponseBody) Validate() error {
	if b.Schema != core.SchemaWitnessCheckInResponse {
		return checkInResponseError(core.ErrLicenseContract)
	}
	if err := b.RequestNonce.Validate(); err != nil {
		return checkInResponseError(err)
	}
	if err := b.Decision.Validate(); err != nil {
		return err
	}
	if b.Decision.Granted != (b.Grant != nil) {
		return checkInResponseError(core.ErrLicenseContract)
	}
	if b.Grant != nil {
		if err := b.Grant.Validate(); err != nil {
			return checkInResponseError(err)
		}
	}
	return nil
}

func (b WitnessCheckInResponseBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldRequestNonce, b.RequestNonce)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldDecision, b.Decision)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldGrant, b.Grant)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (b WitnessCheckInResponseBody) SigningSchema() core.SchemaID { return b.Schema }

func (b WitnessCheckInResponseBody) MarshalJSON() ([]byte, error) {
	return b.Canonical(nil)
}

type WitnessCheckInResponse struct {
	Authority core.Signed[WitnessCheckInResponseBody] `json:"authority"`
}

func (WitnessCheckInResponse) APIBody() {}

func (r WitnessCheckInResponse) Validate() error {
	return checkInResponseErrorOptional(r.Authority.Validate())
}

func (r WitnessCheckInResponse) Verify(keyring core.SigningKeyring) error {
	if err := r.Authority.Verify(keyring); err != nil {
		return checkInResponseError(err)
	}
	if r.Authority.Body.Grant != nil {
		if err := r.Authority.Body.Grant.Verify(keyring); err != nil {
			return checkInResponseError(err)
		}
	}
	return nil
}

func (r WitnessCheckInResponse) VerifyRequestNonce(nonce CheckInNonce) error {
	if err := nonce.Validate(); err != nil || r.Authority.Body.RequestNonce != nonce {
		return checkInResponseError(core.ErrCheckInNonce)
	}
	return nil
}

func checkInResponseError(err error) error {
	return fmt.Errorf(ErrFmtCheckInResponse, errors.Join(core.ErrLicenseContract, err))
}

func checkInResponseErrorOptional(err error) error {
	if err == nil {
		return nil
	}
	return checkInResponseError(err)
}
