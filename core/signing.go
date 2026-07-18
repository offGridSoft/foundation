package core

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
)

const (
	ErrFmtSigningKeyID    = "core.SigningKeyID: %w"
	ErrFmtSigningKeyring  = "core.SigningKeyring: %w"
	ErrFmtSignedSignature = "core.Signed.Signature: %w"
	SignedMessageDomain   = "foundation-signed-" + ContractYear
	SignedMessageSep      = byte(0)
	SigningKeyringMaxKeys = 16
)

type CanonicalValue interface {
	Validatable
	Canonical(dst []byte) ([]byte, error)
}

type CanonicalBody interface {
	CanonicalValue
	SigningSchema() SchemaID
}

type SigningDomain uint16

const (
	SigningDomainUnknown                        = SigningDomain(SchemaUnknown)
	SigningDomainBugCheckInResponse             = SigningDomain(SchemaBugCheckInResponse)
	SigningDomainBugCheckInTimeCommitment       = SigningDomain(SchemaBugCheckInTimeCommitment)
	SigningDomainBugSeatLease                   = SigningDomain(SchemaBugSeatLease)
	SigningDomainBugWriterAttestation           = SigningDomain(SchemaBugWriterAttestation)
	SigningDomainBugWriterCertificate           = SigningDomain(SchemaBugWriterCertificate)
	SigningDomainBugWriterRevocation            = SigningDomain(SchemaBugWriterRevocation)
	SigningDomainWitnessSubscriptionLease       = SigningDomain(SchemaWitnessSubscription)
	SigningDomainWitnessCheckInResponse         = SigningDomain(SchemaWitnessCheckInResponse)
	SigningDomainWitnessCheckInTimeCommitment   = SigningDomain(SchemaWitnessCheckInTimeCommitment)
	SigningDomainWitnessCustodyReceipt          = SigningDomain(SchemaCustodyReceipt)
	SigningDomainReleaseManifest                = SigningDomain(SchemaReleaseManifest)
	SigningDomainReleaseUploadReceipt           = SigningDomain(SchemaReleaseUploadReceipt)
	SigningDomainReleaseDownloadIndex           = SigningDomain(SchemaReleaseDownloadIndex)
	SigningDomainReleaseDeployPlan              = SigningDomain(SchemaReleaseDeployPlan)
	SigningDomainReleasePlan                    = SigningDomain(SchemaReleasePlan)
	SigningDomainReleaseRootLayout              = SigningDomain(SchemaReleaseRootLayout)
	SigningDomainReleaseCommandRun              = SigningDomain(SchemaReleaseCommandRun)
	SigningDomainReleaseSeedGrant               = SigningDomain(SchemaReleaseSeedGrant)
	SigningDomainReleaseUpdateCheckResponse     = SigningDomain(SchemaReleaseUpdateCheckResponse)
	SigningDomainReleaseUpdateDiagnosticReceipt = SigningDomain(SchemaReleaseUpdateDiagnosticReceipt)
	SigningDomainReleaseSeedGrantAccess         = SigningDomain(SchemaReleaseSeedGrantAccess)
	SigningDomainWitnessCustodyDownloadGrant    = SigningDomain(SchemaCustodyDownloadGrant)
)

const (
	SigningDomainTokenBugSeatLease                   = "bug-seat-lease-" + ContractYear // #nosec G101 -- public signing-domain token, not a credential.
	SigningDomainTokenBugCheckInResponse             = "bug-check-in-response-" + ContractYear
	SigningDomainTokenBugCheckInTimeCommitment       = "bug-check-in-time-commitment-" + ContractYear
	SigningDomainTokenBugWriterAttestation           = "bug-writer-attestation-" + ContractYear
	SigningDomainTokenBugWriterCertificate           = "bug-writer-certificate-" + ContractYear
	SigningDomainTokenBugWriterRevocation            = "bug-writer-revocation-" + ContractYear
	SigningDomainTokenWitnessSubscriptionLease       = "witness-subscription-lease-" + ContractYear
	SigningDomainTokenWitnessCheckInResponse         = "witness-check-in-response-" + ContractYear
	SigningDomainTokenWitnessCheckInTimeCommitment   = "witness-check-in-time-commitment-" + ContractYear
	SigningDomainTokenWitnessCustodyReceipt          = "witness-custody-receipt-" + ContractYear
	SigningDomainTokenReleaseManifest                = "release-manifest-" + ContractYear
	SigningDomainTokenReleaseUploadReceipt           = "release-upload-receipt-" + ContractYear
	SigningDomainTokenReleaseDownloadIndex           = "release-download-index-" + ContractYear
	SigningDomainTokenReleaseDeployPlan              = "release-deploy-plan-" + ContractYear
	SigningDomainTokenReleasePlan                    = "release-plan-" + ContractYear
	SigningDomainTokenReleaseRootLayout              = "release-root-layout-" + ContractYear
	SigningDomainTokenReleaseCommandRun              = "release-command-run-" + ContractYear
	SigningDomainTokenReleaseSeedGrant               = "release-seed-grant-" + ContractYear
	SigningDomainTokenReleaseUpdateCheckResponse     = "release-update-check-response-" + ContractYear
	SigningDomainTokenReleaseUpdateDiagnosticReceipt = "release-update-diagnostic-receipt-" + ContractYear
	SigningDomainTokenReleaseSeedGrantAccess         = "release-seed-grant-access-" + ContractYear
	SigningDomainTokenWitnessCustodyDownloadGrant    = "witness-custody-download-grant-" + ContractYear
	ErrFmtSigningDomain                              = "core.SigningDomain: %w"
)

func signingDomainNames() [SchemaCustodyDownloadGrant + 1]string {
	return [SchemaCustodyDownloadGrant + 1]string{
		SigningDomainBugCheckInResponse:             SigningDomainTokenBugCheckInResponse,
		SigningDomainBugCheckInTimeCommitment:       SigningDomainTokenBugCheckInTimeCommitment,
		SigningDomainBugSeatLease:                   SigningDomainTokenBugSeatLease,
		SigningDomainBugWriterAttestation:           SigningDomainTokenBugWriterAttestation,
		SigningDomainBugWriterCertificate:           SigningDomainTokenBugWriterCertificate,
		SigningDomainBugWriterRevocation:            SigningDomainTokenBugWriterRevocation,
		SigningDomainWitnessSubscriptionLease:       SigningDomainTokenWitnessSubscriptionLease,
		SigningDomainWitnessCheckInResponse:         SigningDomainTokenWitnessCheckInResponse,
		SigningDomainWitnessCheckInTimeCommitment:   SigningDomainTokenWitnessCheckInTimeCommitment,
		SigningDomainWitnessCustodyReceipt:          SigningDomainTokenWitnessCustodyReceipt,
		SigningDomainReleaseManifest:                SigningDomainTokenReleaseManifest,
		SigningDomainReleaseUploadReceipt:           SigningDomainTokenReleaseUploadReceipt,
		SigningDomainReleaseDownloadIndex:           SigningDomainTokenReleaseDownloadIndex,
		SigningDomainReleaseDeployPlan:              SigningDomainTokenReleaseDeployPlan,
		SigningDomainReleasePlan:                    SigningDomainTokenReleasePlan,
		SigningDomainReleaseRootLayout:              SigningDomainTokenReleaseRootLayout,
		SigningDomainReleaseCommandRun:              SigningDomainTokenReleaseCommandRun,
		SigningDomainReleaseSeedGrant:               SigningDomainTokenReleaseSeedGrant,
		SigningDomainReleaseUpdateCheckResponse:     SigningDomainTokenReleaseUpdateCheckResponse,
		SigningDomainReleaseUpdateDiagnosticReceipt: SigningDomainTokenReleaseUpdateDiagnosticReceipt,
		SigningDomainReleaseSeedGrantAccess:         SigningDomainTokenReleaseSeedGrantAccess,
		SigningDomainWitnessCustodyDownloadGrant:    SigningDomainTokenWitnessCustodyDownloadGrant,
	}
}

func (d SigningDomain) String() string {
	if d.IsValid() {
		return signingDomainNames()[d]
	}
	return ""
}

func (d SigningDomain) IsValid() bool {
	return d > SigningDomainUnknown && int(d) < len(signingDomainNames()) && signingDomainNames()[d] != ""
}

func (d SigningDomain) Validate() error {
	if !d.IsValid() {
		return fmt.Errorf(ErrFmtSigningDomain, ErrFoundationContract)
	}
	return nil
}

func ParseSigningDomain(value string) (SigningDomain, error) {
	for domain := SigningDomainUnknown + 1; int(domain) < len(signingDomainNames()); domain++ {
		if domain.IsValid() && domain.String() == value {
			return domain, nil
		}
	}
	return SigningDomainUnknown, fmt.Errorf(ErrFmtSigningDomain, ErrFoundationContract)
}

func (d SigningDomain) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(d.String())
}

//validate:unmarshal_ignore reason="ParseSigningDomain validates a temporary before assignment so rejected input cannot mutate the receiver."
func (d *SigningDomain) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtSigningDomain, ErrFoundationContract)
	}
	parsed, err := ParseSigningDomain(value)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

func (id SchemaID) ResolveSigningDomain() SigningDomain {
	domain := SigningDomain(id)
	if domain.IsValid() {
		return domain
	}
	return SigningDomainUnknown
}

type SigningKeyID struct {
	value string
}

func ParseSigningKeyID(value string) (SigningKeyID, error) {
	if err := ValidateOpaqueToken(value, OpaqueTokenDefaultMaxRunes); err != nil {
		return SigningKeyID{}, fmt.Errorf(ErrFmtSigningKeyID, ErrFoundationContract)
	}
	return SigningKeyID{value: value}, nil
}

func (id SigningKeyID) String() string {
	return id.value
}

func (id SigningKeyID) Validate() error {
	_, err := ParseSigningKeyID(id.value)
	return err
}

func (id SigningKeyID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

func (id *SigningKeyID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtSigningKeyID, ErrFoundationContract)
	}
	parsed, err := ParseSigningKeyID(value)
	if err != nil {
		return err
	}
	*id = parsed
	if err := id.Validate(); err != nil {
		return err
	}
	return nil
}

type SigningPublicKey struct {
	ID        SigningKeyID
	PublicKey Ed25519PublicKeyHex
}

func (k SigningPublicKey) Validate() error {
	if err := k.ID.Validate(); err != nil {
		return err
	}
	return k.PublicKey.Validate()
}

type SigningKeyring struct {
	Keys []SigningPublicKey
}

// NewPinnedAuthorityKeyring binds a pinned Ed25519 authority key to the
// signing-key identity carried by Foundation signed artifacts. Consumers must
// not reconstruct that identity convention from raw strings.
func NewPinnedAuthorityKeyring(publicKey Ed25519PublicKeyHex) (SigningKeyring, error) {
	if err := publicKey.Validate(); err != nil {
		return SigningKeyring{}, fmt.Errorf(ErrFmtSigningKeyring, err)
	}
	keyID, err := ParseSigningKeyID(publicKey.String())
	if err != nil {
		return SigningKeyring{}, fmt.Errorf(ErrFmtSigningKeyring, err)
	}
	keyring := SigningKeyring{Keys: []SigningPublicKey{{ID: keyID, PublicKey: publicKey}}}
	if err := keyring.Validate(); err != nil {
		return SigningKeyring{}, err
	}
	return keyring, nil
}

func (r SigningKeyring) Validate() error {
	if len(r.Keys) == 0 || len(r.Keys) > SigningKeyringMaxKeys {
		return fmt.Errorf(ErrFmtSigningKeyring, ErrFoundationContract)
	}
	for index, key := range r.Keys {
		if err := key.Validate(); err != nil {
			return err
		}
		for _, prior := range r.Keys[:index] {
			if prior.ID == key.ID {
				return fmt.Errorf(ErrFmtSigningKeyring, ErrFoundationContract)
			}
		}
	}
	return nil
}

func (r SigningKeyring) Lookup(id SigningKeyID) (ed25519.PublicKey, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r.lookupValidated(id)
}

func (r SigningKeyring) lookupValidated(id SigningKeyID) (ed25519.PublicKey, error) {
	for _, key := range r.Keys {
		if key.ID.String() == id.String() {
			return key.PublicKey.Bytes()
		}
	}
	return nil, fmt.Errorf(ErrFmtSigningKeyring, ErrFoundationContract)
}

type Signed[B CanonicalBody] struct {
	Body      B                   `json:"body"`
	KeyID     SigningKeyID        `json:"key_id"`
	Signature Ed25519SignatureHex `json:"signature"`
}

func (s Signed[B]) Validate() error {
	if err := s.KeyID.Validate(); err != nil {
		return err
	}
	if err := s.Signature.Validate(); err != nil {
		return fmt.Errorf(ErrFmtSignedSignature, err)
	}
	return s.Body.Validate()
}

func (s Signed[B]) Verify(keyring SigningKeyring) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if err := keyring.Validate(); err != nil {
		return err
	}
	message, err := appendSignedMessageValidated(nil, s.KeyID, s.Body)
	if err != nil {
		return err
	}
	key, err := keyring.lookupValidated(s.KeyID)
	if err != nil {
		return err
	}
	signature, err := s.Signature.Bytes()
	if err != nil {
		return fmt.Errorf(ErrFmtSignedSignature, err)
	}
	if !ed25519.Verify(key, message, signature) {
		return fmt.Errorf(ErrFmtSignedSignature, ErrFoundationContract)
	}
	return nil
}

func AppendSignedMessage[B CanonicalBody](dst []byte, keyID SigningKeyID, body B) ([]byte, error) {
	if err := keyID.Validate(); err != nil {
		return nil, err
	}
	if err := body.Validate(); err != nil {
		return nil, err
	}
	return appendSignedMessageValidated(dst, keyID, body)
}

func appendSignedMessageValidated[B CanonicalBody](dst []byte, keyID SigningKeyID, body B) ([]byte, error) {
	dst = append(dst, SignedMessageDomain...)
	dst = append(dst, SignedMessageSep)
	domain := body.SigningSchema().ResolveSigningDomain()
	if err := domain.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, domain.String()...)
	dst = append(dst, SignedMessageSep)
	dst = append(dst, keyID.String()...)
	dst = append(dst, SignedMessageSep)
	return body.Canonical(dst)
}
