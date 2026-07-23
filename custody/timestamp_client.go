package custody

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
	"github.com/offGridSoft/foundation/v2026/exchange"
)

const (
	RFC3161HTTPBudget            = 15 * time.Second
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
	request := exchange.BoundedRequest[core.APIEndpoint]{
		Target:                      c.Endpoint,
		Body:                        query,
		Semantics:                   core.HTTPRequestSemantics{Method: core.HTTPMethodPost, Replay: core.HTTPReplaySingleAttempt},
		ExpectedStatus:              core.HTTPStatusOK,
		RequestContentType:          core.HTTPMediaTypeTimestampQuery,
		ExpectedResponseContentType: core.HTTPMediaTypeTimestampReply,
	}
	policy := exchange.BoundedPolicy{
		AttemptTimeout:    core.NewNanosecondsDuration(RFC3161HTTPBudget),
		RequestBodyLimit:  core.NewByteCount(RFC3161DERMaximumBytes),
		ResponseBodyLimit: core.NewByteCount(RFC3161DERMaximumBytes),
		Redirect:          core.HTTPRedirectReject,
	}
	exchangeClient, err := exchange.NewClient(c.HTTP)
	if err != nil {
		return nil, TimestampHTTPError{Cause: err}
	}
	response, err := exchange.SendBounded(ctx, exchangeClient, request, policy)
	if err != nil {
		return nil, TimestampHTTPError{StatusCode: response.Status.Int(), Cause: err}
	}
	if len(response.Body) == 0 {
		return nil, TimestampHTTPError{StatusCode: response.Status.Int(), Cause: core.ErrCustodyContract}
	}
	return response.Body, nil
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
