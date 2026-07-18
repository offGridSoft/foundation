package custody

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

// DownloadGrantClockSkew tolerates clock drift when a consumer checks a
// grant's validity window with ValidateAt.
const DownloadGrantClockSkew = 30 * time.Second

const (
	jsonFieldTargets   = "targets"
	jsonFieldExpiresAt = "expires_at"
)

var _ core.CanonicalBody = DownloadGrantBody{}

// DownloadURL is a signed, expiring HTTPS retrieval URL issued by the server
// for one custody object. It is custody-owned: signed upload URLs remain
// core.SignedUploadURL and the two must not be conflated.
type DownloadURL struct {
	value string
}

func ParseDownloadURL(value string) (DownloadURL, error) {
	if err := core.ValidateHTTPSURL(value, core.HTTPSURLPolicy{
		MaxRunes:    core.HTTPSURLDefaultMaxRunes,
		RequirePath: true,
		AllowQuery:  true,
	}); err != nil {
		return DownloadURL{}, fmt.Errorf(ErrFmtDownloadURL, core.ErrCustodyContract)
	}
	return DownloadURL{value: value}, nil
}

func (u DownloadURL) String() string {
	return u.value
}

func (u DownloadURL) Validate() error {
	_, err := ParseDownloadURL(u.value)
	return err
}

func (u DownloadURL) MarshalJSON() ([]byte, error) {
	if err := u.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(u.value)
}

func (u *DownloadURL) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtDownloadURL, core.ErrCustodyContract)
	}
	parsed, err := ParseDownloadURL(value)
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

type DownloadMethod uint8

const (
	downloadMethodInvalid DownloadMethod = iota
	DownloadMethodSignedGET
)

const DownloadMethodTokenSignedGET = "signed_get"

func downloadMethodNames() [DownloadMethodSignedGET + 1]string {
	return [...]string{DownloadMethodSignedGET: DownloadMethodTokenSignedGET}
}

func (m DownloadMethod) String() string {
	if m.IsValid() {
		return downloadMethodNames()[m]
	}
	return ""
}

func (m DownloadMethod) IsValid() bool {
	return m > downloadMethodInvalid && int(m) < len(downloadMethodNames()) && downloadMethodNames()[m] != ""
}

func (m DownloadMethod) Validate() error {
	if !m.IsValid() {
		return fmt.Errorf(ErrFmtDownloadMethod, core.ErrCustodyContract)
	}
	return nil
}

func ParseDownloadMethod(token string) (DownloadMethod, error) {
	for method := DownloadMethodSignedGET; int(method) < len(downloadMethodNames()); method++ {
		if downloadMethodNames()[method] == token {
			return method, nil
		}
	}
	return downloadMethodInvalid, fmt.Errorf(ErrFmtDownloadMethod, core.ErrCustodyContract)
}

func (m DownloadMethod) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m.String())
}

func (m *DownloadMethod) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtDownloadMethod, core.ErrCustodyContract)
	}
	parsed, err := ParseDownloadMethod(token)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// DownloadRequest asks the server for a signed download grant over one
// finalized custody bundle. Like SessionOpenRequest it authenticates through
// the account's device lease; the server resolves custody standing from the
// lease before issuing any grant.
type DownloadRequest struct {
	Lease      OpenLeaseRef   `json:"lease"`
	Customer   CustomerID     `json:"customer_id"`
	BundleRoot core.BLAKE3Hex `json:"bundle_root"`
	Schema     core.SchemaID  `json:"schema"`
}

func (r DownloadRequest) Validate() error {
	if r.Schema != core.SchemaCustodyDownloadRequest {
		return fmt.Errorf(ErrFmtDownloadRequest, core.ErrCustodyContract)
	}
	if err := r.Customer.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadRequest, err)
	}
	if err := r.BundleRoot.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadRequest, err)
	}
	if err := r.Lease.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadRequest, err)
	}
	return nil
}

// DownloadTarget binds one signed retrieval URL to the content-addressed
// identity the receipt sealed for that artifact. DownloadArtifact verifies
// the streamed bytes against SHA256 and Size before reporting success.
type DownloadTarget struct {
	Artifact ArtifactName         `json:"artifact"`
	Object   ObjectPath           `json:"object"`
	URL      DownloadURL          `json:"url"`
	SHA256   core.SHA256Hex       `json:"sha256"`
	BLAKE3   core.BLAKE3Hex       `json:"blake3"`
	Size     core.ByteCount       `json:"size_bytes"`
	Provider core.StorageProvider `json:"provider"`
	Method   DownloadMethod       `json:"method"`
}

func (t DownloadTarget) Validate() error {
	if err := t.Artifact.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadTarget, err)
	}
	if err := t.Object.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadTarget, err)
	}
	if err := t.URL.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadTarget, err)
	}
	if err := t.Method.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadTarget, err)
	}
	if err := t.Provider.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadTarget, err)
	}
	return t.validateContentAddress()
}

func (t DownloadTarget) validateContentAddress() error {
	if err := t.SHA256.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadTarget, err)
	}
	if err := t.BLAKE3.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadTarget, err)
	}
	if err := t.Size.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadTarget, err)
	}
	if t.Size.Uint64() >= math.MaxInt64 {
		return fmt.Errorf(ErrFmtDownloadTarget, core.ErrCustodyContract)
	}
	return nil
}

// DownloadGrantBody is the server-signed authorization to retrieve one
// finalized custody bundle. It is the single typed download path: CLI fetch
// and portal download both consume Signed[DownloadGrantBody] and stream
// through DownloadArtifact against these content-addressed targets.
type DownloadGrantBody struct {
	Receipt    ReceiptID         `json:"receipt_id"`
	Customer   CustomerID        `json:"customer_id"`
	BundleRoot core.BLAKE3Hex    `json:"bundle_root"`
	Targets    []DownloadTarget  `json:"targets"`
	IssuedAt   core.UnixNanoTime `json:"issued_at"`
	ExpiresAt  core.UnixNanoTime `json:"expires_at"`
	Schema     core.SchemaID     `json:"schema"`
}

func (b DownloadGrantBody) Validate() error {
	if b.Schema != core.SchemaCustodyDownloadGrant {
		return fmt.Errorf(ErrFmtDownloadGrant, core.ErrCustodyContract)
	}
	if err := b.Receipt.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadGrant, err)
	}
	if err := b.Customer.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadGrant, err)
	}
	if err := b.BundleRoot.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadGrant, err)
	}
	if err := validateDownloadTargetSet(b.Targets, b.Customer, b.BundleRoot); err != nil {
		return err
	}
	return b.validateWindow()
}

func (b DownloadGrantBody) validateWindow() error {
	if err := core.ValidateRequiredUnixNanoTime(b.IssuedAt); err != nil {
		return fmt.Errorf(ErrFmtDownloadGrant, core.ErrCustodyContract)
	}
	if err := core.ValidateRequiredUnixNanoTime(b.ExpiresAt); err != nil || !b.IssuedAt.Before(b.ExpiresAt) {
		return fmt.Errorf(ErrFmtDownloadGrant, core.ErrCustodyContract)
	}
	return nil
}

// ValidateAt refuses grants outside their validity window; consumers with a
// clock must call it before streaming any target.
func (b DownloadGrantBody) ValidateAt(now core.UnixNanoTime) error {
	if err := b.Validate(); err != nil {
		return err
	}
	if err := core.ValidateRequiredUnixNanoTime(now); err != nil {
		return fmt.Errorf(ErrFmtDownloadGrant, core.ErrCustodyContract)
	}
	if now.Before(b.IssuedAt.Add(-DownloadGrantClockSkew)) || now.After(b.ExpiresAt) {
		return fmt.Errorf(ErrFmtDownloadGrant, core.ErrCustodyContract)
	}
	return nil
}

// BindReceipt proves the grant's target set is exactly the receipt's sealed
// content-addressed object set: same receipt identity and one target per
// object with identical artifact, object path, hashes, and size.
func (b DownloadGrantBody) BindReceipt(receipt ReceiptBody) error {
	if err := b.Validate(); err != nil {
		return err
	}
	if err := receipt.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadGrant, err)
	}
	if b.Receipt != receipt.ReceiptID || b.Customer != receipt.Customer || b.BundleRoot != receipt.BundleRoot {
		return fmt.Errorf(ErrFmtDownloadGrant, core.ErrCustodyContract)
	}
	if len(b.Targets) != len(receipt.Objects) {
		return fmt.Errorf(ErrFmtDownloadGrant, core.ErrCustodyContract)
	}
	for _, object := range receipt.Objects {
		if !downloadTargetMatchesObject(b.Targets, object) {
			return fmt.Errorf(ErrFmtDownloadGrant, core.ErrCustodyContract)
		}
	}
	return nil
}

func downloadTargetMatchesObject(targets []DownloadTarget, object UploadedObject) bool {
	for _, target := range targets {
		if target.Artifact == object.Artifact {
			return target.Object == object.Object && target.SHA256 == object.SHA256 &&
				target.BLAKE3 == object.BLAKE3 && target.Size == object.Size
		}
	}
	return false
}

func validateDownloadTargetSet(targets []DownloadTarget, customer CustomerID, bundleRoot core.BLAKE3Hex) error {
	if err := (core.CollectionCardinality{
		Length:  len(targets),
		Minimum: 1,
		Maximum: core.CollectionMaximumDefault,
	}).Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadGrant, core.ErrCustodyContract)
	}
	for index, target := range targets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf(ErrFmtDownloadGrant, err)
		}
		if target.Provider != core.StorageProviderGCS {
			return fmt.Errorf(ErrFmtDownloadGrant, core.ErrCustodyContract)
		}
		if err := target.Object.ValidateWitnessIdentity(customer, bundleRoot, target.Artifact); err != nil {
			return fmt.Errorf(ErrFmtDownloadGrant, err)
		}
		for _, prior := range targets[:index] {
			if prior.Artifact == target.Artifact || prior.Object == target.Object {
				return fmt.Errorf(ErrFmtDownloadGrant, core.ErrCustodyContract)
			}
		}
	}
	return nil
}

func (b DownloadGrantBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldReceiptID, b.Receipt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldCustomerID, b.Customer)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldBundleRoot, b.BundleRoot)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldTargets, b.Targets)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldIssuedAt, b.IssuedAt)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldExpiresAt, b.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (b DownloadGrantBody) MarshalJSON() ([]byte, error) {
	return b.Canonical(nil)
}

func (b DownloadGrantBody) SigningSchema() core.SchemaID {
	return b.Schema
}

// RequestDownload exchanges the lease-authenticated request for the signed
// download grant. The grant is accepted only after its server signature
// verifies and its body binds to the requested customer and bundle root.
func (c Client) RequestDownload(ctx context.Context, request DownloadRequest) (core.Signed[DownloadGrantBody], error) {
	if err := c.Validate(); err != nil {
		return core.Signed[DownloadGrantBody]{}, err
	}
	if err := request.Validate(); err != nil {
		return core.Signed[DownloadGrantBody]{}, err
	}
	grant, err := custodyPost[DownloadRequest, core.Signed[DownloadGrantBody]](ctx, c.HTTP, c.Endpoints.Download, request)
	if err != nil {
		return core.Signed[DownloadGrantBody]{}, err
	}
	if err := grant.Verify(c.ServerKeys); err != nil {
		return core.Signed[DownloadGrantBody]{}, err
	}
	if grant.Body.Customer != request.Customer || grant.Body.BundleRoot != request.BundleRoot {
		return core.Signed[DownloadGrantBody]{}, fmt.Errorf(ErrFmtDownloadGrant, core.ErrCustodyContract)
	}
	return grant, nil
}

// DownloadArtifact streams one grant target into dst with O(1) memory: bytes
// flow straight from the storage response through the SHA-256 accumulator
// into dst, bounded by the target's declared size. dst content is unverified
// until DownloadArtifact returns nil; a non-nil error means the stream was
// truncated, oversized, or failed content-address verification.
func (c Client) DownloadArtifact(ctx context.Context, target DownloadTarget, dst io.Writer) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if ctx == nil {
		return fmt.Errorf(ErrFmtDownloadArtifact, errors.Join(core.ErrCustodyContract, core.ErrNilContext))
	}
	if dst == nil {
		return fmt.Errorf(ErrFmtDownloadArtifact, core.ErrCustodyContract)
	}
	if err := target.Validate(); err != nil {
		return fmt.Errorf(ErrFmtDownloadArtifact, err)
	}
	requestContext, cancel := context.WithTimeout(ctx, CustodyTransferHTTPBudget)
	defer cancel()
	httpResponse, err := c.doDownloadGet(requestContext, target)
	if err != nil {
		return err
	}
	defer func() { _ = httpResponse.Body.Close() }()
	return verifyDownloadStream(httpResponse, target, dst)
}

func (c Client) doDownloadGet(ctx context.Context, target DownloadTarget) (*http.Response, error) {
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL.String(), nil)
	if err != nil {
		return nil, CustodyHTTPError{Cause: err}
	}
	httpResponse, err := c.HTTP.Do(httpRequest)
	if err != nil {
		return nil, CustodyHTTPError{Cause: err}
	}
	if httpResponse == nil || httpResponse.Body == nil {
		return nil, CustodyHTTPError{Cause: core.ErrCustodyContract}
	}
	return httpResponse, nil
}

func verifyDownloadStream(httpResponse *http.Response, target DownloadTarget, dst io.Writer) error {
	if httpResponse.StatusCode != core.HTTPStatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(httpResponse.Body, CustodyTransferResponseMaxBytes))
		return CustodyHTTPError{StatusCode: httpResponse.StatusCode, Cause: core.ErrCustodyContract}
	}
	size, err := target.Size.Int64()
	if err != nil {
		return fmt.Errorf(ErrFmtDownloadArtifact, err)
	}
	hasher := sha256.New()
	copied, err := io.Copy(io.MultiWriter(dst, hasher), io.LimitReader(httpResponse.Body, size+1))
	if err != nil {
		return CustodyHTTPError{Cause: err}
	}
	if copied != size {
		return fmt.Errorf(ErrFmtDownloadArtifact, core.ErrStorageVerification)
	}
	var sum [sha256.Size]byte
	hasher.Sum(sum[:0])
	if core.NewSHA256Hex(sum) != target.SHA256 {
		return fmt.Errorf(ErrFmtDownloadArtifact, core.ErrStorageVerification)
	}
	return nil
}
