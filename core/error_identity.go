package core

import (
	"errors"
	"fmt"
)

var (
	ErrFoundationContract = errors.New("foundation contract violation")
	ErrLicenseContract    = errors.New("license contract violation")
	ErrCustodyContract    = errors.New("custody contract violation")
	ErrReleaseContract    = errors.New("release contract violation")
	ErrJSONContract       = errors.New("json contract violation")
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
	ErrFmtJSONTrailingValue   = "core.JSON.TrailingValue: %w"
	ErrFmtJSONDuplicateField  = "core.JSON.DuplicateField: %w"
	ErrFmtJSONUnexpectedField = "core.JSON.UnexpectedField: %w"
	ErrFmtJSONUnexpectedDelim = "core.JSON.UnexpectedDelimiter: %w"
	ErrFmtJSONDecode          = "core.JSON.Decode: %w"
)
