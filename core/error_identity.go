package core

import (
	"errors"
	"fmt"
)

var (
	ErrFoundationContract              = errors.New("foundation contract violation")
	ErrFuzzContract                    = fmt.Errorf("fuzz contract violation: %w", ErrFoundationContract)
	ErrLicenseContract                 = fmt.Errorf("license contract violation: %w", ErrFoundationContract)
	ErrCheckInNonce                    = fmt.Errorf("check-in nonce contract violation: %w", ErrLicenseContract)
	ErrLeaseGeneration                 = fmt.Errorf("lease generation contract violation: %w", ErrLicenseContract)
	ErrCustodyContract                 = fmt.Errorf("custody contract violation: %w", ErrFoundationContract)
	ErrReleaseContract                 = fmt.Errorf("release contract violation: %w", ErrFoundationContract)
	ErrJSONContract                    = fmt.Errorf("json contract violation: %w", ErrFoundationContract)
	ErrDeliveryContract                = fmt.Errorf("coalescing delivery contract violation: %w", ErrFoundationContract)
	ErrExchangeContract                = fmt.Errorf("network exchange contract violation: %w", ErrFoundationContract)
	ErrExchangeRequest                 = fmt.Errorf("network exchange request rejected: %w", ErrExchangeContract)
	ErrExchangeResponse                = fmt.Errorf("network exchange response rejected: %w", ErrExchangeContract)
	ErrExchangeBodyLimit               = fmt.Errorf("network exchange body limit exceeded: %w", ErrExchangeContract)
	ErrExchangeContentType             = fmt.Errorf("network exchange content type rejected: %w", ErrExchangeContract)
	ErrExchangeCancelled               = fmt.Errorf("network exchange cancelled: %w", ErrExchangeContract)
	ErrExchangeRedirect                = fmt.Errorf("network exchange redirect rejected: %w", ErrExchangeContract)
	ErrExchangeRetryExhausted          = fmt.Errorf("network exchange retry budget exhausted: %w", ErrExchangeContract)
	ErrExchangeTransport               = fmt.Errorf("network exchange transport failure: %w", ErrExchangeContract)
	ErrExchangeWrite                   = fmt.Errorf("network exchange write failure: %w", ErrExchangeContract)
	ErrKeygenContract                  = fmt.Errorf("keygen contract violation: %w", ErrFoundationContract)
	ErrKeygenEntropy                   = errors.New("keygen entropy failure")
	ErrWitnessPolicyContract           = fmt.Errorf("witness policy contract violation: %w", ErrFoundationContract)
	ErrUnknownAccountToken             = fmt.Errorf("unknown account token: %w", ErrLicenseContract)
	ErrInactiveSubscription            = fmt.Errorf("inactive subscription: %w", ErrLicenseContract)
	ErrPaymentFailing                  = fmt.Errorf("payment failing: %w", ErrLicenseContract)
	ErrMachineCapExceeded              = fmt.Errorf("machine cap exceeded: %w", ErrLicenseContract)
	ErrSeatCapacityExceeded            = fmt.Errorf("seat capacity exceeded: %w", ErrLicenseContract)
	ErrSeatInviteDuplicate             = fmt.Errorf("seat invite already pending: %w", ErrLicenseContract)
	ErrSeatAssignmentConflict          = fmt.Errorf("seat assignment conflict: %w", ErrLicenseContract)
	ErrSeatMemberInactive              = fmt.Errorf("seat member inactive: %w", ErrLicenseContract)
	ErrUnknownBinarySHA                = fmt.Errorf("unknown binary SHA: %w", ErrLicenseContract)
	ErrInvalidBundle                   = fmt.Errorf("invalid bundle: %w", ErrCustodyContract)
	ErrDuplicateFinalizedBundle        = fmt.Errorf("duplicate finalized bundle: %w", ErrCustodyContract)
	ErrExpiredUploadSession            = fmt.Errorf("expired upload session: %w", ErrCustodyContract)
	ErrStorageVerification             = fmt.Errorf("storage verification failure: %w", ErrCustodyContract)
	ErrAccessContract                  = fmt.Errorf("access contract violation: %w", ErrFoundationContract)
	ErrIPNetworkContract               = fmt.Errorf("ip network contract violation: %w", ErrFoundationContract)
	ErrTestSerialContract              = fmt.Errorf("test serial contract violation: %w", ErrFoundationContract)
	ErrDoctrineContract                = fmt.Errorf("doctrine contract violation: %w", ErrFoundationContract)
	ErrContextContract                 = fmt.Errorf("context contract violation: %w", ErrFoundationContract)
	ErrFilesystemContract              = fmt.Errorf("filesystem contract violation: %w", ErrFoundationContract)
	ErrDurabilityContract              = fmt.Errorf("durability contract violation: %w", ErrFilesystemContract)
	ErrDurableSizeLimit                = fmt.Errorf("durable stream size limit exceeded: %w", ErrDurabilityContract)
	ErrDurableShortWrite               = fmt.Errorf("durable short write: %w", ErrDurabilityContract)
	ErrDurableActivationIncomplete     = fmt.Errorf("durable activation incomplete: %w", ErrDurabilityContract)
	ErrDurableCleanupIncomplete        = fmt.Errorf("durable temporary cleanup incomplete: %w", ErrDurabilityContract)
	ErrHostResourceContract            = fmt.Errorf("host resource contract violation: %w", ErrFoundationContract)
	ErrDiskCapacityUnsupported         = fmt.Errorf("disk capacity unsupported: %w", ErrHostResourceContract)
	ErrDiskFloorReached                = fmt.Errorf("disk free-space floor reached: %w", ErrHostResourceContract)
	ErrMemoryLimitReached              = fmt.Errorf("memory limit trigger reached: %w", ErrHostResourceContract)
	ErrShutdownContract                = fmt.Errorf("shutdown contract violation: %w", ErrFoundationContract)
	ErrShutdownStepFailure             = fmt.Errorf("shutdown step failed: %w", ErrShutdownContract)
	ErrShutdownStepTimeout             = fmt.Errorf("shutdown step timed out: %w", ErrShutdownContract)
	ErrShutdownStepPanic               = fmt.Errorf("shutdown step panicked: %w", ErrShutdownContract)
	ErrShutdownTotalTimeout            = fmt.Errorf("shutdown total budget exhausted: %w", ErrShutdownContract)
	ErrShutdownForceFailure            = fmt.Errorf("shutdown force action failed: %w", ErrShutdownContract)
	ErrShutdownForceTimeout            = fmt.Errorf("shutdown force action timed out: %w", ErrShutdownContract)
	ErrShutdownSignalSourceClosed      = fmt.Errorf("shutdown signal source closed: %w", ErrShutdownContract)
	ErrShutdownSignalReceived          = fmt.Errorf("shutdown signal received: %w", ErrShutdownContract)
	ErrShutdownForcePanic              = fmt.Errorf("shutdown force action panicked: %w", ErrShutdownContract)
	ErrTemporalContract                = fmt.Errorf("temporal contract violation: %w", ErrFoundationContract)
	ErrPeachfuzzContributionRegression = fmt.Errorf("peachfuzz contribution regression: %w", ErrFoundationContract)
	ErrNilContext                      = fmt.Errorf("nil context: %w", ErrFoundationContract)
	ErrNumericOverflow                 = fmt.Errorf("numeric overflow: %w", ErrFoundationContract)
	ErrInvalidDecimal                  = fmt.Errorf("invalid decimal: %w", ErrFoundationContract)
)

func wrapFoundationContract(format string) error {
	return fmt.Errorf(format, ErrFoundationContract)
}

const (
	ErrFmtProductVersion        = "core.ProductVersion: %w"
	ErrFmtProduct               = "core.Product: %w"
	ErrFmtCodeLanguage          = "core.CodeLanguage: %w"
	ErrFmtPlatform              = "core.Platform: %w"
	ErrFmtBuildCommit           = "core.BuildCommit: %w"
	ErrFmtFileNameToken         = "core.FileNameToken: %w"
	ErrFmtPathToken             = "core.PathToken: %w" // #nosec G101 -- path token error label, not a credential.
	ErrFmtAbsoluteFilePath      = "core.AbsoluteFilePath: %w"
	ErrFmtAbsoluteDirectoryPath = "core.AbsoluteDirectoryPath: %w"
	ErrFmtHTTPSURL              = "core.HTTPSURL: %w"
	ErrFmtIPNetwork             = "core.IPNetwork: %w"
	ErrFmtTestSerialReason      = "core.TestSerialReason: %w"
	ErrFmtDoctrineLayer         = "core.DoctrinePackageLayer: %w"
	ErrFmtDoctrineCapability    = "core.DoctrinePackageCapability: %w"
	ErrFmtAPIEndpoint           = "core.APIEndpoint: %w"
	ErrFmtUniqueToken           = "core.UniqueToken: %w"
	ErrFmtArtifactSet           = "core.ArtifactSet: %w"
	ErrFmtSHA256                = "core.SHA256Hex: %w"
	ErrFmtBLAKE3                = "core.BLAKE3Hex: %w"
	ErrFmtEd25519PublicKey      = "core.Ed25519PublicKeyHex: %w"
	ErrFmtEd25519Signature      = "core.Ed25519SignatureHex: %w"
	ErrFmtHTTPOutcome           = "core.HTTPOutcome: %w"
	ErrFmtHTTPHeaderName        = "core.HTTPHeaderName: %w"
	ErrFmtHTTPHeaderValue       = "core.HTTPHeaderValue: %w"
	ErrFmtStorageProvider       = "core.StorageProvider: %w"
	ErrFmtUploadMethod          = "core.UploadMethod: %w"
	ErrFmtSignedUploadURL       = "core.SignedUploadURL: %w"
	ErrFmtUploadHeader          = "core.UploadHeader: %w"
	ErrFmtBackoffAttempts       = "core.BackoffPolicy.MaxAttempts: %w"
	ErrFmtBackoffWindow         = "core.BackoffPolicy.Window: %w"
	ErrFmtCoalescingDelivery    = "core.CoalescingDelivery: %w"
	ErrFmtDeliveryPhase         = "core.DeliveryPhase: %w"
	ErrFmtHTTPMethod            = "core.HTTPMethod: %w"
	ErrFmtHTTPRedirectPolicy    = "core.HTTPRedirectPolicy: %w"
	ErrFmtHTTPReplaySafety      = "core.HTTPReplaySafety: %w"
	ErrFmtHTTPIdempotencyKey    = "core.HTTPIdempotencyKey: %w"
	ErrFmtHTTPMediaType         = "core.HTTPMediaType: %w"
	ErrFmtHTTPRequestSemantics  = "core.HTTPRequestSemantics: %w"
	ErrFmtHTTPRouteSemantics    = "core.HTTPRouteSemantics: %w"
	ErrFmtHTTPHeader            = "core.HTTPHeader: %w"
	ErrFmtHTTPHeaders           = "core.HTTPHeaders: %w"
	ErrFmtHTTPQueryParameter    = "core.HTTPQueryParameter: %w"
	ErrFmtHTTPQuery             = "core.HTTPQuery: %w"
	ErrFmtHTTPStatusCode        = "core.HTTPStatusCode: %w"
	ErrFmtHTTPRetryDirective    = "core.HTTPRetryDirective: %w"
	ErrFmtHTTPRetryPolicy       = "core.HTTPRetryPolicy: %w"
	ErrFmtUnixNanoTime          = "core.UnixNanoTime: %w"
	ErrFmtNanosecondsDuration   = "core.NanosecondsDuration: %w"
	ErrFmtMoneyPennies          = "core.MoneyPennies: %w"
	ErrFmtJSONFieldName         = "core.JSON.FieldName: %w"
	ErrFmtJSONTrailingValue     = "core.JSON.TrailingValue: %w"
	ErrFmtJSONDuplicateField    = "core.JSON.DuplicateField: %w"
	ErrFmtJSONUnexpectedField   = "core.JSON.UnexpectedField: %w"
	ErrFmtJSONUnexpectedDelim   = "core.JSON.UnexpectedDelimiter: %w"
	ErrFmtJSONUnexpectedValue   = "core.JSON.UnexpectedValue: %w"
	ErrFmtJSONEncode            = "core.JSON.Encode: %w"
	ErrFmtJSONDecode            = "core.JSON.Decode: %w"
)
