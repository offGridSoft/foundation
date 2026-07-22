package custody

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"io"
	"math"
	"net/http"
	"slices"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
	"github.com/offGridSoft/foundation/v2026/exchange"
)

const (
	// CustodyAPIHTTPBudget bounds one custody API exchange (open, finalize,
	// download-grant). Bundle transfers use CustodyTransferHTTPBudget.
	CustodyAPIHTTPBudget = 15 * time.Second
	// CustodyTransferHTTPBudget bounds one streamed artifact transfer to or
	// from signed storage URLs.
	CustodyTransferHTTPBudget = 10 * time.Minute
	// CustodyTransferResponseMaxBytes bounds the storage provider's own
	// response body (success metadata or error document) on a transfer.
	// Artifact payload streams are bounded by the declared artifact size.
	CustodyTransferResponseMaxBytes = 1 << 20
	// GCSGenerationHeaderName carries the created object generation on a
	// signed-PUT response; finalize reports it for server-side verification.
	GCSGenerationHeaderName = "X-Goog-Generation"
)

type CustodyAPIError struct {
	RequestID  core.APIRequestID
	Body       core.APIErrorBody
	StatusCode int
}

func (e CustodyAPIError) Error() string {
	return fmt.Sprintf(ErrFmtCustodyAPI, e.StatusCode, e.Body.Code, e.Body.Message)
}

func (e CustodyAPIError) Unwrap() error { return core.ErrCustodyContract }

type CustodyHTTPError struct {
	Cause      error
	StatusCode int
}

type CustodyRetryExhaustedError struct {
	Cause      error
	RetryAfter core.NanosecondsDuration
	Attempts   uint64
}

func (e CustodyRetryExhaustedError) Error() string {
	return fmt.Sprintf("custody retry exhausted after %d attempts: %v", e.Attempts, e.Cause)
}

func (e CustodyRetryExhaustedError) Unwrap() error {
	return errors.Join(core.ErrCustodyContract, core.ErrExchangeRetryExhausted, e.Cause)
}

func (e CustodyHTTPError) Error() string {
	return fmt.Sprintf(ErrFmtCustodyHTTP, e.StatusCode, e.Cause)
}

func (e CustodyHTTPError) Unwrap() error {
	return errors.Join(core.ErrCustodyContract, e.Cause)
}

// Client drives the custody protocol end to end: session open, streamed
// artifact upload to signed storage targets, finalize, and typed bundle
// retrieval. Requests authenticate through the OpenLeaseRef they carry (the
// server binds the lease to the account's custody standing); every signed
// server payload is verified against ServerKeys before it is returned.
type Client struct {
	HTTP       *http.Client
	Endpoints  Endpoints
	ServerKeys core.SigningKeyring
}

func (c Client) Validate() error {
	if c.HTTP == nil {
		return fmt.Errorf(ErrFmtClient, core.ErrCustodyContract)
	}
	if err := c.Endpoints.Validate(); err != nil {
		return fmt.Errorf(ErrFmtClient, err)
	}
	if err := c.ServerKeys.Validate(); err != nil {
		return fmt.Errorf(ErrFmtClient, err)
	}
	return nil
}

// OpenSession submits the session-open request and returns the validated
// response. A reused receipt is accepted only after its server signature
// verifies; an upload grant is accepted only when its targets bind to the
// exact requested customer, bundle root, and artifact set.
func (c Client) OpenSession(ctx context.Context, request SessionOpenRequest) (SessionOpenResponse, error) {
	if err := c.Validate(); err != nil {
		return SessionOpenResponse{}, err
	}
	if err := request.Validate(); err != nil {
		return SessionOpenResponse{}, err
	}
	response, err := custodyPost[SessionOpenRequest, SessionOpenResponse](ctx, c.HTTP, c.Endpoints.Open, request)
	if err != nil {
		return SessionOpenResponse{}, err
	}
	if err := response.Verify(c.ServerKeys); err != nil {
		return SessionOpenResponse{}, err
	}
	if err := bindOpenResponse(request, response); err != nil {
		return SessionOpenResponse{}, err
	}
	return response, nil
}

func bindOpenResponse(request SessionOpenRequest, response SessionOpenResponse) error {
	if response.Customer != request.Customer || response.BundleRoot != request.BundleRoot {
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
	if response.Disposition != SessionOpenDispositionUploadRequired {
		return nil
	}
	if len(response.Upload.Targets) != len(request.Artifacts) {
		return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
	}
	for _, artifact := range request.Artifacts {
		if !uploadTargetExistsFor(response.Upload.Targets, artifact.Name) {
			return fmt.Errorf(ErrFmtOpenResponse, core.ErrCustodyContract)
		}
	}
	return nil
}

func uploadTargetExistsFor(targets []UploadTarget, name ArtifactName) bool {
	for _, target := range targets {
		if target.Artifact == name {
			return true
		}
	}
	return false
}

// Finalize submits the finalize request and returns the signed custody
// receipt. The receipt is accepted only after its server signature verifies
// and its body binds to the exact session, customer, bundle root, and
// reported content-addressed object set.
func (c Client) Finalize(ctx context.Context, request FinalizeRequest) (core.Signed[ReceiptBody], error) {
	if err := c.Validate(); err != nil {
		return core.Signed[ReceiptBody]{}, err
	}
	if err := request.Validate(); err != nil {
		return core.Signed[ReceiptBody]{}, err
	}
	signed, err := custodyPost[FinalizeRequest, core.Signed[ReceiptBody]](ctx, c.HTTP, c.Endpoints.Finalize, request)
	if err != nil {
		return core.Signed[ReceiptBody]{}, err
	}
	if err := signed.Verify(c.ServerKeys); err != nil {
		return core.Signed[ReceiptBody]{}, err
	}
	if err := bindReceiptToFinalize(request, signed.Body); err != nil {
		return core.Signed[ReceiptBody]{}, err
	}
	return signed, nil
}

func bindReceiptToFinalize(request FinalizeRequest, receipt ReceiptBody) error {
	if receipt.Session != request.Session || receipt.Customer != request.Customer || receipt.BundleRoot != request.BundleRoot {
		return fmt.Errorf(ErrFmtReceipt, core.ErrCustodyContract)
	}
	if len(receipt.Objects) != len(request.Objects) {
		return fmt.Errorf(ErrFmtReceipt, core.ErrCustodyContract)
	}
	for _, reported := range request.Objects {
		if !uploadedObjectExists(receipt.Objects, reported) {
			return fmt.Errorf(ErrFmtReceipt, core.ErrCustodyContract)
		}
	}
	return nil
}

func uploadedObjectExists(objects []UploadedObject, want UploadedObject) bool {
	return slices.Contains(objects, want)
}

// UploadArtifactInput carries one artifact stream to its signed upload
// target. Body must yield exactly Artifact.Size bytes whose SHA-256 digest
// equals Artifact.SHA256; the client verifies both while streaming.
type UploadArtifactInput struct {
	Body     io.Reader
	Artifact ArtifactDescriptor
	Target   UploadTarget
}

func (i UploadArtifactInput) Validate() error {
	if i.Body == nil {
		return fmt.Errorf(ErrFmtUploadArtifact, core.ErrCustodyContract)
	}
	if err := i.Target.Validate(); err != nil {
		return fmt.Errorf(ErrFmtUploadArtifact, err)
	}
	if err := i.Artifact.Validate(); err != nil {
		return fmt.Errorf(ErrFmtUploadArtifact, err)
	}
	if i.Target.Artifact != i.Artifact.Name || i.Target.Method != core.UploadMethodSignedPUT || i.Target.Provider != core.StorageProviderGCS {
		return fmt.Errorf(ErrFmtUploadArtifact, core.ErrCustodyContract)
	}
	if i.Artifact.Size.Uint64() >= math.MaxInt64 {
		return fmt.Errorf(ErrFmtUploadArtifact, core.ErrCustodyContract)
	}
	return nil
}

// UploadArtifact streams the artifact to its signed target with O(1) memory:
// the body is never buffered, the SHA-256 digest and byte count accumulate
// on the wire path, and both must match the declared descriptor before the
// UploadedObject that finalize reports is produced.
func (c Client) UploadArtifact(ctx context.Context, input UploadArtifactInput) (UploadedObject, error) {
	if err := c.Validate(); err != nil {
		return UploadedObject{}, err
	}
	if ctx == nil {
		return UploadedObject{}, fmt.Errorf(ErrFmtUploadArtifact, errors.Join(core.ErrCustodyContract, core.ErrNilContext))
	}
	if err := input.Validate(); err != nil {
		return UploadedObject{}, err
	}
	requestContext, cancel := context.WithTimeout(ctx, CustodyTransferHTTPBudget)
	defer cancel()
	hasher := sha256.New()
	response, err := exchange.SendStream(requestContext, exchange.Client{HTTP: c.HTTP}, exchange.StreamUploadRequest[core.SignedUploadURL]{
		Target:         input.Target.URL,
		Body:           io.TeeReader(input.Body, hasher),
		ContentLength:  input.Artifact.Size,
		Headers:        uploadHTTPHeaders(input.Target.Headers),
		Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodPut, Replay: core.HTTPReplaySingleAttempt},
		ExpectedStatus: core.HTTPStatusOK,
		CaptureHeaders: exchange.HeaderSelection{Names: []string{GCSGenerationHeaderName}},
	}, custodyTransferPolicy())
	if err != nil {
		return UploadedObject{}, custodyStreamError(response.Status, err)
	}
	streamed, err := response.BytesWritten.Int64()
	if err != nil {
		return UploadedObject{}, fmt.Errorf(ErrFmtUploadArtifact, err)
	}
	generation, _ := response.Headers.Get(GCSGenerationHeaderName)
	return uploadedObjectFromResponse(input, generation, streamed, hasher)
}

func uploadHTTPHeaders(headers []core.UploadHeader) core.HTTPHeaders {
	values := make([]core.HTTPHeader, len(headers))
	for index, header := range headers {
		values[index] = core.HTTPHeader(header)
	}
	return core.HTTPHeaders{Values: values}
}

func custodyTransferPolicy() exchange.StreamPolicy {
	return exchange.StreamPolicy{
		AttemptTimeout: core.NewNanosecondsDuration(CustodyTransferHTTPBudget),
		ErrorBodyLimit: core.NewByteCount(CustodyTransferResponseMaxBytes),
		Redirect:       core.HTTPRedirectReject,
	}
}

func custodyStreamError(status core.HTTPStatusCode, cause error) error {
	return CustodyHTTPError{StatusCode: status.Int(), Cause: cause}
}

func uploadedObjectFromResponse(input UploadArtifactInput, generationValue string, streamed int64, hasher hash.Hash) (UploadedObject, error) {
	expected, err := input.Artifact.Size.Int64()
	if err != nil {
		return UploadedObject{}, fmt.Errorf(ErrFmtUploadArtifact, err)
	}
	if streamed != expected {
		return UploadedObject{}, fmt.Errorf(ErrFmtUploadArtifact, core.ErrStorageVerification)
	}
	var sum [sha256.Size]byte
	hasher.Sum(sum[:0])
	if core.NewSHA256Hex(sum) != input.Artifact.SHA256 {
		return UploadedObject{}, fmt.Errorf(ErrFmtUploadArtifact, core.ErrStorageVerification)
	}
	generation, err := ParseGeneration(generationValue)
	if err != nil {
		return UploadedObject{}, fmt.Errorf(ErrFmtUploadArtifact, err)
	}
	object := UploadedObject{
		Artifact:   input.Artifact.Name,
		Object:     input.Target.Object,
		Generation: generation,
		SHA256:     input.Artifact.SHA256,
		BLAKE3:     input.Artifact.BLAKE3,
		Size:       input.Artifact.Size,
		Provider:   core.StorageProviderGCS,
	}
	if err := object.Validate(); err != nil {
		return UploadedObject{}, fmt.Errorf(ErrFmtUploadArtifact, err)
	}
	return object, nil
}

func custodyPost[Request core.HTTPIdempotentBody, Response core.Validatable](
	ctx context.Context,
	httpClient *http.Client,
	endpoint core.APIEndpoint,
	request Request,
) (Response, error) {
	var zero Response
	if ctx == nil {
		return zero, fmt.Errorf(ErrFmtClient, errors.Join(core.ErrCustodyContract, core.ErrNilContext))
	}
	key, err := request.HTTPIdempotencyKey()
	if err != nil {
		return zero, CustodyHTTPError{Cause: err}
	}
	requestContext, cancel := context.WithTimeout(ctx, CustodyAPIHTTPBudget)
	defer cancel()
	exchangeRequest := exchange.Request[Request]{
		Body: &request, Endpoint: endpoint,
		Semantics: core.HTTPRequestSemantics{
			Method: core.HTTPMethodPost, Replay: core.HTTPReplayIdempotent, IdempotencyKey: key,
		},
		ExpectedStatus: core.HTTPStatusOK,
	}
	exchangeResponse, err := exchange.SendJSON[Request, Response](
		requestContext, exchange.Client{HTTP: httpClient}, exchangeRequest, custodyAPIClientPolicy(),
	)
	if err != nil {
		return zero, custodyExchangeError(exchangeResponse, err)
	}
	return *exchangeResponse.Envelope.Data, nil
}

func custodyAPIClientPolicy() exchange.ClientPolicy {
	return exchange.ClientPolicy{
		AttemptTimeout:    core.NewNanosecondsDuration(CustodyAPIHTTPBudget),
		RequestBodyLimit:  core.NewByteCount(core.StrictJSONMaxBytes),
		ResponseBodyLimit: core.NewByteCount(core.StrictJSONMaxBytes),
		Retry:             core.DefaultHTTPRetryPolicy(),
		Redirect:          core.HTTPRedirectReject,
	}
}

func custodyExchangeError[Response core.Validatable](response exchange.Response[Response], cause error) error {
	if exhausted, ok := errors.AsType[exchange.RetryExhaustedError](cause); ok {
		return CustodyRetryExhaustedError{
			Cause:      custodyExchangeCause(response.Status, exhausted.Cause),
			RetryAfter: exhausted.RetryAfter,
			Attempts:   exhausted.Attempts,
		}
	}
	return custodyExchangeCause(response.Status, cause)
}

func custodyExchangeCause(status core.HTTPStatusCode, cause error) error {
	if apiError, ok := errors.AsType[exchange.ResponseError](cause); ok {
		return CustodyAPIError{StatusCode: apiError.Status.Int(), RequestID: apiError.RequestID, Body: apiError.Body}
	}
	return CustodyHTTPError{StatusCode: status.Int(), Cause: cause}
}
