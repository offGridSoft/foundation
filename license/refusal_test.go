package license

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestWitnessRefusalContracts(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		identity    error
		name        string
		refusal     Refusal
		remediation Remediation
	}{
		{name: "unknown account token requires reauthentication", refusal: RefusalUnknownAccountToken, remediation: RemediationReauthenticate, identity: core.ErrUnknownAccountToken},
		{name: "inactive subscription requires activation", refusal: RefusalInactiveSubscription, remediation: RemediationActivateSubscription, identity: core.ErrInactiveSubscription},
		{name: "failing payment requires payment update", refusal: RefusalPaymentFailing, remediation: RemediationUpdatePayment, identity: core.ErrPaymentFailing},
		{name: "machine cap requires deactivation", refusal: RefusalMachineCapExceeded, remediation: RemediationDeactivateMachine, identity: core.ErrMachineCapExceeded},
		{name: "unknown binary requires verified download", refusal: RefusalUnknownBinarySHA, remediation: RemediationRedownloadVerifiedBinary, identity: core.ErrUnknownBinarySHA},
		{name: "invalid bundle requires repair", refusal: RefusalInvalidBundle, remediation: RemediationRepairBundle, identity: core.ErrInvalidBundle},
		{name: "duplicate finalized bundle reuses receipt", refusal: RefusalDuplicateFinalizedBundle, remediation: RemediationUseExistingReceipt, identity: core.ErrDuplicateFinalizedBundle},
		{name: "expired upload session requires reopen", refusal: RefusalExpiredUploadSession, remediation: RemediationReopenUploadSession, identity: core.ErrExpiredUploadSession},
		{name: "storage verification failure retries upload", refusal: RefusalStorageVerificationFailure, remediation: RemediationRetryUpload, identity: core.ErrStorageVerification},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewRefusalError(tc.refusal)
			if err != nil {
				t.Fatalf("NewRefusalError() error = %v, want nil", err)
			}
			want := RefusalError{Refusal: tc.refusal, Remediation: tc.remediation}
			if got != want {
				t.Fatalf("NewRefusalError() = %+v, want %+v", got, want)
			}
			if !errors.Is(got, tc.identity) {
				t.Fatalf("errors.Is(RefusalError, identity) = false, want true")
			}
			var typed RefusalError
			if !errors.As(got, &typed) || typed != want {
				t.Fatalf("errors.As(RefusalError) = %+v, want %+v", typed, want)
			}
		})
	}
}

func TestRefusalAndRemediationWireContracts(t *testing.T) {
	t.Parallel()

	refusal := RefusalStorageVerificationFailure
	remediation := RemediationRetryUpload
	refusalJSON, err := json.Marshal(refusal)
	if err != nil {
		t.Fatalf("json.Marshal(Refusal) error = %v, want nil", err)
	}
	remediationJSON, err := json.Marshal(remediation)
	if err != nil {
		t.Fatalf("json.Marshal(Remediation) error = %v, want nil", err)
	}
	var gotRefusal Refusal
	if err := json.Unmarshal(refusalJSON, &gotRefusal); err != nil {
		t.Fatalf("json.Unmarshal(Refusal) error = %v, want nil", err)
	}
	var gotRemediation Remediation
	if err := json.Unmarshal(remediationJSON, &gotRemediation); err != nil {
		t.Fatalf("json.Unmarshal(Remediation) error = %v, want nil", err)
	}
	if gotRefusal != refusal || gotRemediation != remediation {
		t.Fatalf("wire round trip = %s/%s, want %s/%s", gotRefusal, gotRemediation, refusal, remediation)
	}
}

func TestRefusalErrorRejectsContractDrift(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		error RefusalError
	}{
		{name: "none is not a refusal", error: RefusalError{Refusal: RefusalNone}},
		{name: "wrong remediation is rejected", error: RefusalError{Refusal: RefusalPaymentFailing, Remediation: RemediationDeactivateMachine}},
		{name: "unknown refusal ordinal is rejected", error: RefusalError{Refusal: Refusal(RefusalStorageVerificationFailure + 1), Remediation: RemediationRetryUpload}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.error.Validate(); !errors.Is(err, core.ErrLicenseContract) {
				t.Fatalf("RefusalError.Validate() error = %v, want ErrLicenseContract", err)
			}
		})
	}
}
