package license

import (
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	RefusalTokenNone                       = "none"
	RefusalTokenKeyRevoked                 = "key_revoked"
	RefusalTokenSeatLimit                  = "seat_limit_exceeded" // #nosec G101 -- public refusal token, not a credential.
	RefusalTokenPaymentRequired            = "payment_required"
	RefusalTokenUnsupportedBuild           = "unsupported_build"     // #nosec G101 -- public refusal token, not a credential.
	RefusalTokenDeviceLimit                = "device_limit_exceeded" // #nosec G101 -- public refusal token, not a credential.
	RefusalTokenUnknownKey                 = "unknown_key"
	RefusalTokenUnknownAccountToken        = "unknown_account_token"
	RefusalTokenInactiveSubscription       = "inactive_subscription"
	RefusalTokenPaymentFailing             = "payment_failing"
	RefusalTokenMachineCapExceeded         = "machine_cap_exceeded" // #nosec G101 -- public refusal token, not a credential.
	RefusalTokenUnknownBinarySHA           = "unknown_binary_sha"
	RefusalTokenInvalidBundle              = "invalid_bundle"
	RefusalTokenDuplicateFinalizedBundle   = "duplicate_finalized_bundle"
	RefusalTokenExpiredUploadSession       = "expired_upload_session"
	RefusalTokenStorageVerificationFailure = "storage_verification_failure"

	RemediationTokenNone                     = "none"
	RemediationTokenContactSupport           = "contact_support"
	RemediationTokenReduceSeats              = "reduce_seats"
	RemediationTokenUpdatePayment            = "update_payment"
	RemediationTokenInstallSupportedBuild    = "install_supported_build"
	RemediationTokenDeactivateMachine        = "deactivate_machine"
	RemediationTokenReauthenticate           = "reauthenticate"
	RemediationTokenActivateSubscription     = "activate_subscription"
	RemediationTokenRedownloadVerifiedBinary = "redownload_verified_binary" // #nosec G101 -- public remediation token, not a credential.
	RemediationTokenRepairBundle             = "repair_bundle"
	RemediationTokenUseExistingReceipt       = "use_existing_receipt"
	RemediationTokenReopenUploadSession      = "reopen_upload_session" // #nosec G101 -- public remediation token, not a credential.
	RemediationTokenRetryUpload              = "retry_upload"

	ErrFmtRemediation  = "license.Remediation: %w"
	ErrFmtRefusalError = "license.RefusalError: refusal=%s remediation=%s: %v"
	remediationCount   = int(RemediationRetryUpload) + 1
)

type Refusal uint8

const (
	refusalInvalid Refusal = iota
	RefusalNone
	RefusalKeyRevoked
	RefusalSeatLimit
	RefusalPaymentRequired
	RefusalUnsupportedBuild
	RefusalDeviceLimit
	RefusalUnknownKey
	RefusalUnknownAccountToken
	RefusalInactiveSubscription
	RefusalPaymentFailing
	RefusalMachineCapExceeded
	RefusalUnknownBinarySHA
	RefusalInvalidBundle
	RefusalDuplicateFinalizedBundle
	RefusalExpiredUploadSession
	RefusalStorageVerificationFailure
)

func refusalNames() [RefusalStorageVerificationFailure + 1]string {
	return [...]string{
		RefusalNone:                       RefusalTokenNone,
		RefusalKeyRevoked:                 RefusalTokenKeyRevoked,
		RefusalSeatLimit:                  RefusalTokenSeatLimit,
		RefusalPaymentRequired:            RefusalTokenPaymentRequired,
		RefusalUnsupportedBuild:           RefusalTokenUnsupportedBuild,
		RefusalDeviceLimit:                RefusalTokenDeviceLimit,
		RefusalUnknownKey:                 RefusalTokenUnknownKey,
		RefusalUnknownAccountToken:        RefusalTokenUnknownAccountToken,
		RefusalInactiveSubscription:       RefusalTokenInactiveSubscription,
		RefusalPaymentFailing:             RefusalTokenPaymentFailing,
		RefusalMachineCapExceeded:         RefusalTokenMachineCapExceeded,
		RefusalUnknownBinarySHA:           RefusalTokenUnknownBinarySHA,
		RefusalInvalidBundle:              RefusalTokenInvalidBundle,
		RefusalDuplicateFinalizedBundle:   RefusalTokenDuplicateFinalizedBundle,
		RefusalExpiredUploadSession:       RefusalTokenExpiredUploadSession,
		RefusalStorageVerificationFailure: RefusalTokenStorageVerificationFailure,
	}
}

type Remediation uint8

const (
	RemediationNone Remediation = iota
	RemediationContactSupport
	RemediationReduceSeats
	RemediationUpdatePayment
	RemediationInstallSupportedBuild
	RemediationDeactivateMachine
	RemediationReauthenticate
	RemediationActivateSubscription
	RemediationRedownloadVerifiedBinary
	RemediationRepairBundle
	RemediationUseExistingReceipt
	RemediationReopenUploadSession
	RemediationRetryUpload
)

func remediationNames() [RemediationRetryUpload + 1]string {
	return [...]string{
		RemediationNone:                     RemediationTokenNone,
		RemediationContactSupport:           RemediationTokenContactSupport,
		RemediationReduceSeats:              RemediationTokenReduceSeats,
		RemediationUpdatePayment:            RemediationTokenUpdatePayment,
		RemediationInstallSupportedBuild:    RemediationTokenInstallSupportedBuild,
		RemediationDeactivateMachine:        RemediationTokenDeactivateMachine,
		RemediationReauthenticate:           RemediationTokenReauthenticate,
		RemediationActivateSubscription:     RemediationTokenActivateSubscription,
		RemediationRedownloadVerifiedBinary: RemediationTokenRedownloadVerifiedBinary,
		RemediationRepairBundle:             RemediationTokenRepairBundle,
		RemediationUseExistingReceipt:       RemediationTokenUseExistingReceipt,
		RemediationReopenUploadSession:      RemediationTokenReopenUploadSession,
		RemediationRetryUpload:              RemediationTokenRetryUpload,
	}
}

func (r Remediation) String() string {
	if r.IsValid() {
		return remediationNames()[r]
	}
	return ""
}

func (r Remediation) IsValid() bool {
	return int(r) < remediationCount && remediationNames()[r] != ""
}

func (r Remediation) Validate() error {
	if !r.IsValid() {
		return fmt.Errorf(ErrFmtRemediation, core.ErrLicenseContract)
	}
	return nil
}

func ParseRemediation(token string) (Remediation, error) {
	for remediation := RemediationNone; int(remediation) < remediationCount; remediation++ {
		if remediationNames()[remediation] == token {
			return remediation, nil
		}
	}
	return Remediation(remediationCount), fmt.Errorf(ErrFmtRemediation, core.ErrLicenseContract)
}

func (r Remediation) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r.String())
}

func (r *Remediation) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtRemediation, core.ErrLicenseContract)
	}
	parsed, err := ParseRemediation(token)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}

type RefusalError struct {
	Refusal     Refusal
	Remediation Remediation
}

func NewRefusalError(refusal Refusal) (RefusalError, error) {
	remediation, err := RemediationForRefusal(refusal)
	if err != nil || refusal == RefusalNone {
		return RefusalError{}, fmt.Errorf(ErrFmtCheckInRefusal, core.ErrLicenseContract)
	}
	return RefusalError{Refusal: refusal, Remediation: remediation}, nil
}

func (e RefusalError) Error() string {
	return fmt.Sprintf(ErrFmtRefusalError, e.Refusal, e.Remediation, e.identity())
}

func (e RefusalError) Unwrap() error {
	return e.identity()
}

func (e RefusalError) Validate() error {
	want, err := RemediationForRefusal(e.Refusal)
	if err != nil || e.Refusal == RefusalNone || e.Remediation != want {
		return fmt.Errorf(ErrFmtCheckInRefusal, core.ErrLicenseContract)
	}
	return nil
}

func RemediationForRefusal(refusal Refusal) (Remediation, error) {
	if err := refusal.Validate(); err != nil {
		return RemediationNone, err
	}
	return remediationByRefusal()[refusal], nil
}

func remediationByRefusal() [RefusalStorageVerificationFailure + 1]Remediation {
	return [...]Remediation{
		RefusalNone:                       RemediationNone,
		RefusalKeyRevoked:                 RemediationContactSupport,
		RefusalSeatLimit:                  RemediationReduceSeats,
		RefusalPaymentRequired:            RemediationUpdatePayment,
		RefusalUnsupportedBuild:           RemediationInstallSupportedBuild,
		RefusalDeviceLimit:                RemediationDeactivateMachine,
		RefusalUnknownKey:                 RemediationReauthenticate,
		RefusalUnknownAccountToken:        RemediationReauthenticate,
		RefusalInactiveSubscription:       RemediationActivateSubscription,
		RefusalPaymentFailing:             RemediationUpdatePayment,
		RefusalMachineCapExceeded:         RemediationDeactivateMachine,
		RefusalUnknownBinarySHA:           RemediationRedownloadVerifiedBinary,
		RefusalInvalidBundle:              RemediationRepairBundle,
		RefusalDuplicateFinalizedBundle:   RemediationUseExistingReceipt,
		RefusalExpiredUploadSession:       RemediationReopenUploadSession,
		RefusalStorageVerificationFailure: RemediationRetryUpload,
	}
}

func (e RefusalError) identity() error {
	switch e.Refusal {
	case RefusalUnknownAccountToken:
		return core.ErrUnknownAccountToken
	case RefusalInactiveSubscription:
		return core.ErrInactiveSubscription
	case RefusalPaymentFailing:
		return core.ErrPaymentFailing
	case RefusalMachineCapExceeded:
		return core.ErrMachineCapExceeded
	case RefusalUnknownBinarySHA:
		return core.ErrUnknownBinarySHA
	case RefusalInvalidBundle:
		return core.ErrInvalidBundle
	case RefusalDuplicateFinalizedBundle:
		return core.ErrDuplicateFinalizedBundle
	case RefusalExpiredUploadSession:
		return core.ErrExpiredUploadSession
	case RefusalStorageVerificationFailure:
		return core.ErrStorageVerification
	default:
		return core.ErrLicenseContract
	}
}

func (r Refusal) String() string {
	if r.IsValid() {
		return refusalNames()[r]
	}
	return ""
}

func (r Refusal) IsValid() bool {
	return r > refusalInvalid && int(r) < len(refusalNames()) && refusalNames()[r] != ""
}

func (r Refusal) Validate() error {
	if !r.IsValid() {
		return fmt.Errorf(ErrFmtCheckInRefusal, core.ErrLicenseContract)
	}
	return nil
}

func ParseRefusal(token string) (Refusal, error) {
	for refusal := RefusalNone; int(refusal) < len(refusalNames()); refusal++ {
		if refusalNames()[refusal] == token {
			return refusal, nil
		}
	}
	return refusalInvalid, fmt.Errorf(ErrFmtCheckInRefusal, core.ErrLicenseContract)
}

func (r Refusal) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r.String())
}

func (r *Refusal) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtCheckInRefusal, core.ErrLicenseContract)
	}
	parsed, err := ParseRefusal(token)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}
