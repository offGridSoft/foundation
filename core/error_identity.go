package core

import (
	"errors"
	"fmt"
)

var (
	ErrFoundationContract       = errors.New("foundation contract violation")
	ErrLicenseContract          = fmt.Errorf("license contract violation: %w", ErrFoundationContract)
	ErrCheckInNonce             = fmt.Errorf("check-in nonce contract violation: %w", ErrLicenseContract)
	ErrLeaseGeneration          = fmt.Errorf("lease generation contract violation: %w", ErrLicenseContract)
	ErrCustodyContract          = fmt.Errorf("custody contract violation: %w", ErrFoundationContract)
	ErrReleaseContract          = fmt.Errorf("release contract violation: %w", ErrFoundationContract)
	ErrJSONContract             = fmt.Errorf("json contract violation: %w", ErrFoundationContract)
	ErrWitnessPolicyContract    = fmt.Errorf("witness policy contract violation: %w", ErrFoundationContract)
	ErrUnknownAccountToken      = fmt.Errorf("unknown account token: %w", ErrLicenseContract)
	ErrInactiveSubscription     = fmt.Errorf("inactive subscription: %w", ErrLicenseContract)
	ErrPaymentFailing           = fmt.Errorf("payment failing: %w", ErrLicenseContract)
	ErrMachineCapExceeded       = fmt.Errorf("machine cap exceeded: %w", ErrLicenseContract)
	ErrUnknownBinarySHA         = fmt.Errorf("unknown binary SHA: %w", ErrLicenseContract)
	ErrInvalidBundle            = fmt.Errorf("invalid bundle: %w", ErrCustodyContract)
	ErrDuplicateFinalizedBundle = fmt.Errorf("duplicate finalized bundle: %w", ErrCustodyContract)
	ErrExpiredUploadSession     = fmt.Errorf("expired upload session: %w", ErrCustodyContract)
	ErrStorageVerification      = fmt.Errorf("storage verification failure: %w", ErrCustodyContract)
	ErrAccessContract           = fmt.Errorf("access contract violation: %w", ErrFoundationContract)
	ErrNilContext               = fmt.Errorf("nil context: %w", ErrFoundationContract)
)

func wrapFoundationContract(format string) error {
	return fmt.Errorf(format, ErrFoundationContract)
}

const (
	ErrFmtProductVersion      = "core.ProductVersion: %w"
	ErrFmtProduct             = "core.Product: %w"
	ErrFmtPlatform            = "core.Platform: %w"
	ErrFmtBuildCommit         = "core.BuildCommit: %w"
	ErrFmtFileNameToken       = "core.FileNameToken: %w"
	ErrFmtPathToken           = "core.PathToken: %w" // #nosec G101 -- path token error label, not a credential.
	ErrFmtHTTPSURL            = "core.HTTPSURL: %w"
	ErrFmtUniqueToken         = "core.UniqueToken: %w"
	ErrFmtArtifactSet         = "core.ArtifactSet: %w"
	ErrFmtSHA256              = "core.SHA256Hex: %w"
	ErrFmtBLAKE3              = "core.BLAKE3Hex: %w"
	ErrFmtEd25519PublicKey    = "core.Ed25519PublicKeyHex: %w"
	ErrFmtEd25519Signature    = "core.Ed25519SignatureHex: %w"
	ErrFmtHTTPOutcome         = "core.HTTPOutcome: %w"
	ErrFmtHTTPHeaderName      = "core.HTTPHeaderName: %w"
	ErrFmtHTTPHeaderValue     = "core.HTTPHeaderValue: %w"
	ErrFmtStorageProvider     = "core.StorageProvider: %w"
	ErrFmtUploadMethod        = "core.UploadMethod: %w"
	ErrFmtBackoffAttempts     = "core.BackoffPolicy.MaxAttempts: %w"
	ErrFmtBackoffWindow       = "core.BackoffPolicy.Window: %w"
	ErrFmtUnixNanoTime        = "core.UnixNanoTime: %w"
	ErrFmtNanosecondsDuration = "core.NanosecondsDuration: %w"
	ErrFmtMoneyPennies        = "core.MoneyPennies: %w"
	ErrFmtJSONFieldName       = "core.JSON.FieldName: %w"
	ErrFmtJSONTrailingValue   = "core.JSON.TrailingValue: %w"
	ErrFmtJSONDuplicateField  = "core.JSON.DuplicateField: %w"
	ErrFmtJSONUnexpectedField = "core.JSON.UnexpectedField: %w"
	ErrFmtJSONUnexpectedDelim = "core.JSON.UnexpectedDelimiter: %w"
	ErrFmtJSONUnexpectedValue = "core.JSON.UnexpectedValue: %w"
	ErrFmtJSONDecode          = "core.JSON.Decode: %w"
)
