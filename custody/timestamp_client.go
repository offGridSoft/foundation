package custody

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	RFC3161HTTPBudget            = 15 * time.Second
	RFC3161ContentTypeQuery      = "application/timestamp-query"
	RFC3161ContentTypeReply      = "application/timestamp-reply"
	TimestampAuthorityFreeTSAURL = "https://freetsa.org/tsr"
)

type TimestampHTTPError struct {
	Cause      error
	StatusCode int
}

func (e TimestampHTTPError) Error() string {
	return fmt.Sprintf(ErrFmtTimestampHTTP, e.StatusCode, e.Cause)
}

func (e TimestampHTTPError) Unwrap() error {
	return errors.Join(core.ErrCustodyContract, e.Cause)
}

// EndpointURL resolves the authority's fixed RFC 3161 submission endpoint.
func (a TimestampAuthority) EndpointURL() (core.APIEndpoint, error) {
	switch a {
	case TimestampAuthorityFreeTSA:
		return core.ParseAPIEndpoint(TimestampAuthorityFreeTSAURL)
	default:
		return core.APIEndpoint{}, fmt.Errorf(ErrFmtTimestampClient, core.ErrCustodyContract)
	}
}

// TimestampClient submits RFC 3161 timestamp queries over HTTP and returns
// the typed custody proof. It transports bytes only: request DER comes from
// EncodeRFC3161TimestampQuery and every reply byte is accepted exclusively
// through the existing RFC3161Response/RFC3161Token constructors.
type TimestampClient struct {
	HTTP      *http.Client
	Now       func() time.Time
	Endpoint  core.APIEndpoint
	Authority TimestampAuthority
}

func NewFreeTSATimestampClient(httpClient *http.Client, now func() time.Time) (TimestampClient, error) {
	endpoint, err := TimestampAuthorityFreeTSA.EndpointURL()
	if err != nil {
		return TimestampClient{}, err
	}
	client := TimestampClient{HTTP: httpClient, Now: now, Endpoint: endpoint, Authority: TimestampAuthorityFreeTSA}
	if err := client.Validate(); err != nil {
		return TimestampClient{}, err
	}
	return client, nil
}

func (c TimestampClient) Validate() error {
	if c.HTTP == nil || c.Now == nil {
		return fmt.Errorf(ErrFmtTimestampClient, core.ErrCustodyContract)
	}
	if err := c.Endpoint.Validate(); err != nil {
		return fmt.Errorf(ErrFmtTimestampClient, err)
	}
	if err := c.Authority.Validate(); err != nil {
		return fmt.Errorf(ErrFmtTimestampClient, err)
	}
	return nil
}

// TimestampWitnessCustody derives the domain-separated imprint for
// bundleRoot, submits the TimeStampReq to the authority endpoint, and folds
// the granted reply into a validated TimestampProof.
func (c TimestampClient) TimestampWitnessCustody(ctx context.Context, bundleRoot core.BLAKE3Hex) (TimestampProof, error) {
	if err := c.Validate(); err != nil {
		return TimestampProof{}, err
	}
	if ctx == nil {
		return TimestampProof{}, fmt.Errorf(ErrFmtTimestampClient, errors.Join(core.ErrCustodyContract, core.ErrNilContext))
	}
	imprint, err := DeriveTimestampImprint(bundleRoot)
	if err != nil {
		return TimestampProof{}, err
	}
	query, err := EncodeRFC3161TimestampQuery(imprint)
	if err != nil {
		return TimestampProof{}, err
	}
	replyDER, err := c.postTimestampQuery(ctx, query)
	if err != nil {
		return TimestampProof{}, err
	}
	return c.buildProofFromReply(bundleRoot, replyDER)
}

func (c TimestampClient) postTimestampQuery(ctx context.Context, query []byte) ([]byte, error) {
	requestContext, cancel := context.WithTimeout(ctx, RFC3161HTTPBudget)
	defer cancel()
	httpRequest, err := http.NewRequestWithContext(requestContext, http.MethodPost, c.Endpoint.String(), bytes.NewReader(query))
	if err != nil {
		return nil, TimestampHTTPError{Cause: err}
	}
	httpRequest.Header.Set(core.HTTPHeaderContentType, RFC3161ContentTypeQuery)
	httpResponse, err := c.HTTP.Do(httpRequest)
	if err != nil {
		return nil, TimestampHTTPError{Cause: err}
	}
	if httpResponse == nil || httpResponse.Body == nil {
		return nil, TimestampHTTPError{Cause: core.ErrCustodyContract}
	}
	defer func() { _ = httpResponse.Body.Close() }()
	return readTimestampReply(httpResponse)
}

func readTimestampReply(httpResponse *http.Response) ([]byte, error) {
	replyDER, err := io.ReadAll(io.LimitReader(httpResponse.Body, RFC3161DERMaximumBytes+1))
	if err != nil || len(replyDER) == 0 || len(replyDER) > RFC3161DERMaximumBytes {
		return nil, TimestampHTTPError{StatusCode: httpResponse.StatusCode, Cause: core.ErrCustodyContract}
	}
	contentType := httpResponse.Header.Get(core.HTTPHeaderContentType)
	if httpResponse.StatusCode != core.HTTPStatusOK || !strings.HasPrefix(contentType, RFC3161ContentTypeReply) {
		return nil, TimestampHTTPError{StatusCode: httpResponse.StatusCode, Cause: core.ErrCustodyContract}
	}
	return replyDER, nil
}

func (c TimestampClient) buildProofFromReply(bundleRoot core.BLAKE3Hex, replyDER []byte) (TimestampProof, error) {
	response, err := NewRFC3161Response(replyDER)
	if err != nil {
		return TimestampProof{}, err
	}
	tokenDER, err := embeddedRFC3161Token(replyDER)
	if err != nil {
		return TimestampProof{}, err
	}
	token, err := NewRFC3161Token(tokenDER)
	if err != nil {
		return TimestampProof{}, err
	}
	return BuildTimestampProof(TimestampProofInput{
		Authority: c.Authority, BundleRoot: bundleRoot, Token: token, Response: response,
		TimestampedAt: core.NewUnixNanoTime(c.Now()),
	})
}
