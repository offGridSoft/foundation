package custody

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
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
	for _, object := range objects {
		if object == want {
			return true
		}
	}
	return false
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
	size, err := input.Artifact.Size.Int64()
	if err != nil {
		return UploadedObject{}, fmt.Errorf(ErrFmtUploadArtifact, err)
	}
	stream := &countingReader{reader: io.TeeReader(io.LimitReader(input.Body, size+1), hasher)}
	httpResponse, err := c.doUploadPut(requestContext, input, stream, size)
	if err != nil {
		return UploadedObject{}, err
	}
	defer func() { _ = httpResponse.Body.Close() }()
	if err := drainTransferResponse(httpResponse); err != nil {
		return UploadedObject{}, err
	}
	return uploadedObjectFromResponse(input, httpResponse, stream.count, hasher)
}

func (c Client) doUploadPut(ctx context.Context, input UploadArtifactInput, body io.Reader, size int64) (*http.Response, error) {
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPut, input.Target.URL.String(), body)
	if err != nil {
		return nil, CustodyHTTPError{Cause: err}
	}
	httpRequest.ContentLength = size
	for _, header := range input.Target.Headers {
		httpRequest.Header.Set(header.Name, header.Value)
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

func drainTransferResponse(httpResponse *http.Response) error {
	_, err := io.Copy(io.Discard, io.LimitReader(httpResponse.Body, CustodyTransferResponseMaxBytes))
	if err != nil || httpResponse.StatusCode != core.HTTPStatusOK {
		return CustodyHTTPError{StatusCode: httpResponse.StatusCode, Cause: core.ErrCustodyContract}
	}
	return nil
}

func uploadedObjectFromResponse(input UploadArtifactInput, httpResponse *http.Response, streamed int64, hasher hash.Hash) (UploadedObject, error) {
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
	generation, err := ParseGeneration(httpResponse.Header.Get(GCSGenerationHeaderName))
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

type countingReader struct {
	reader io.Reader
	count  int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.count += int64(n)
	return n, err
}

func custodyPost[Request core.Validatable, Response core.Validatable](
	ctx context.Context,
	httpClient *http.Client,
	endpoint core.APIEndpoint,
	request Request,
) (Response, error) {
	var zero Response
	if ctx == nil {
		return zero, fmt.Errorf(ErrFmtClient, errors.Join(core.ErrCustodyContract, core.ErrNilContext))
	}
	body, err := json.Marshal(request)
	if err != nil {
		return zero, CustodyHTTPError{Cause: err}
	}
	requestContext, cancel := context.WithTimeout(ctx, CustodyAPIHTTPBudget)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(requestContext, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return zero, CustodyHTTPError{Cause: err}
	}
	httpRequest.Header.Set(core.HTTPHeaderContentType, core.HTTPContentTypeJSON)
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return zero, CustodyHTTPError{Cause: err}
	}
	if httpResponse == nil || httpResponse.Body == nil {
		return zero, CustodyHTTPError{Cause: core.ErrCustodyContract}
	}
	defer func() { _ = httpResponse.Body.Close() }()
	contentType := httpResponse.Header.Get(core.HTTPHeaderContentType)
	responseBody, err := io.ReadAll(io.LimitReader(httpResponse.Body, core.StrictJSONMaxBytes+1))
	if err != nil || len(responseBody) == 0 || len(responseBody) > core.StrictJSONMaxBytes {
		return zero, CustodyHTTPError{StatusCode: httpResponse.StatusCode, Cause: core.ErrCustodyContract}
	}
	return decodeCustodyHTTPResponse[Response](httpResponse.StatusCode, contentType, responseBody)
}

func decodeCustodyHTTPResponse[Response core.Validatable](statusCode int, contentType string, responseBody []byte) (Response, error) {
	var zero Response
	if !strings.HasPrefix(contentType, core.HTTPContentTypeJSON) {
		return zero, CustodyHTTPError{StatusCode: statusCode, Cause: core.ErrCustodyContract}
	}
	envelope, err := core.DecodeStrictJSON[core.APIEnvelope[Response]](responseBody)
	if err != nil {
		return zero, CustodyHTTPError{StatusCode: statusCode, Cause: err}
	}
	if statusCode != core.HTTPStatusOK {
		if err := envelope.ValidateFailure(); err != nil {
			return zero, CustodyHTTPError{StatusCode: statusCode, Cause: err}
		}
		return zero, CustodyAPIError{StatusCode: statusCode, RequestID: envelope.RequestID, Body: *envelope.Error}
	}
	if err := envelope.ValidateSuccess(); err != nil {
		return zero, CustodyHTTPError{StatusCode: statusCode, Cause: err}
	}
	return *envelope.Data, nil
}
