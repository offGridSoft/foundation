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

type BugCheckInResponse struct {
	Grant             *BugCheckInGrant       `json:"grant,omitempty"`
	WriterRevocations BugWriterRevocationSet `json:"writer_revocations"`
	Decision          CheckInDecision        `json:"decision"`
}

// APIBody lets the response satisfy lfw/api.Body structurally without a
// dependency from Foundation into the transport package.
func (BugCheckInResponse) APIBody() {}

func (r BugCheckInResponse) Validate() error {
	if err := r.Decision.Validate(); err != nil {
		return err
	}
	if r.Decision.Granted != (r.Grant != nil) {
		return checkInResponseError(core.ErrLicenseContract)
	}
	if r.Grant != nil {
		if err := r.Grant.Validate(); err != nil {
			return checkInResponseError(err)
		}
	}
	return r.WriterRevocations.Validate()
}

func (r BugCheckInResponse) Verify(keyring core.SigningKeyring) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if err := keyring.Validate(); err != nil {
		return checkInResponseError(err)
	}
	if r.Grant != nil {
		if err := r.Grant.Verify(keyring); err != nil {
			return checkInResponseError(err)
		}
	}
	return r.WriterRevocations.Verify(keyring)
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

// Merge preserves every previously observed cutoff. Re-delivery is
// idempotent, while a second signed value for the same writer is a contract
// conflict rather than authority to rewrite history.
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
			if leftValue != rightValue {
				return BugWriterRevocationSet{}, checkInResponseError(core.ErrLicenseContract)
			}
			merged.Values = append(merged.Values, leftValue)
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

type WitnessCheckInResponse struct {
	Grant    *WitnessCheckInGrant `json:"grant,omitempty"`
	Decision CheckInDecision      `json:"decision"`
}

func (WitnessCheckInResponse) APIBody() {}

func (r WitnessCheckInResponse) Validate() error {
	if err := r.Decision.Validate(); err != nil {
		return err
	}
	if r.Decision.Granted != (r.Grant != nil) {
		return checkInResponseError(core.ErrLicenseContract)
	}
	if r.Grant != nil {
		if err := r.Grant.Validate(); err != nil {
			return checkInResponseError(err)
		}
	}
	return nil
}

func (r WitnessCheckInResponse) Verify(keyring core.SigningKeyring) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if err := keyring.Validate(); err != nil {
		return checkInResponseError(err)
	}
	if r.Grant != nil {
		if err := r.Grant.Verify(keyring); err != nil {
			return checkInResponseError(err)
		}
	}
	return nil
}

func checkInResponseError(err error) error {
	return fmt.Errorf(ErrFmtCheckInResponse, errors.Join(core.ErrLicenseContract, err))
}
