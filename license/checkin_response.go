package license

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

var (
	_ core.CanonicalBody = BugCheckInResponseBody{}
	_ core.CanonicalBody = WitnessCheckInResponseBody{}
)

const (
	BugWriterRevocationDeliveryMaximum    uint32 = 256
	BugWriterRevocationPersistenceMaximum uint32 = 16 << 10
)

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
	Grant             *BugCheckInGrant            `json:"grant,omitempty"`
	UpdateNotice      *core.UpdateNotice          `json:"update_notice,omitempty"`
	WriterRevocations BugWriterRevocationDelivery `json:"writer_revocations"`
	Schema            core.SchemaID               `json:"schema"`
	RequestNonce      CheckInNonce                `json:"request_nonce"`
	Decision          CheckInDecision             `json:"decision"`
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
	if err := validateUpdateNotice(b.UpdateNotice, core.ProductBug); err != nil {
		return err
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
	if b.UpdateNotice != nil {
		dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldUpdateNotice, b.UpdateNotice)
	}
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

// BugWriterRevocationDelivery is the bounded server-to-client check-in shape.
// Retained revocation state has a separate type and a larger lifetime bound.
type BugWriterRevocationDelivery struct {
	Values []core.Signed[BugWriterRevocationBody] `json:"values"`
}

func (d BugWriterRevocationDelivery) Validate() error {
	return validateWriterRevocations(d.Values, BugWriterRevocationDeliveryMaximum)
}

func (d BugWriterRevocationDelivery) Verify(keyring core.SigningKeyring) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if err := keyring.Validate(); err != nil {
		return checkInResponseError(err)
	}
	for _, value := range d.Values {
		if err := value.Verify(keyring); err != nil {
			return checkInResponseError(err)
		}
	}
	return nil
}

// BugWriterRevocationSet is the bounded lifetime persistence shape for every
// verified revocation observed by one installation. It deliberately exposes
// no writer-allowance method: individual signatures authenticate presence,
// not completeness, so absence cannot authorize a writer until a signed set
// commitment/progression contract exists.
type BugWriterRevocationSet struct {
	Values []core.Signed[BugWriterRevocationBody] `json:"values"`
}

func (s BugWriterRevocationSet) Validate() error {
	return validateWriterRevocations(s.Values, BugWriterRevocationPersistenceMaximum)
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

func validateWriterRevocations(values []core.Signed[BugWriterRevocationBody], maximum uint32) error {
	if err := (core.CollectionCardinality{
		Length: len(values), Maximum: maximum,
	}).Validate(); err != nil {
		return checkInResponseError(err)
	}
	var prior core.SigningKeyID
	for index, value := range values {
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

// Merge preserves the earliest signed cutoff ever observed for each writer.
// An earlier server cutoff broadens revocation and therefore replaces a later
// one; a later delivery can never narrow already-revoked history.
func (s BugWriterRevocationSet) Merge(incoming BugWriterRevocationDelivery) (BugWriterRevocationSet, error) {
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

type WitnessCheckInResponseBody struct {
	Grant        *WitnessCheckInGrant `json:"grant,omitempty"`
	UpdateNotice *core.UpdateNotice   `json:"update_notice,omitempty"`
	Schema       core.SchemaID        `json:"schema"`
	RequestNonce CheckInNonce         `json:"request_nonce"`
	Decision     CheckInDecision      `json:"decision"`
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
	return validateUpdateNotice(b.UpdateNotice, core.ProductWitness)
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
	if b.UpdateNotice != nil {
		dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldUpdateNotice, b.UpdateNotice)
	}
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

func validateUpdateNotice(notice *core.UpdateNotice, product core.Product) error {
	if notice == nil {
		return nil
	}
	if err := notice.Validate(); err != nil {
		return checkInResponseError(err)
	}
	if notice.Product != product {
		return checkInResponseError(core.ErrLicenseContract)
	}
	return nil
}

func checkInResponseErrorOptional(err error) error {
	if err == nil {
		return nil
	}
	return checkInResponseError(err)
}
