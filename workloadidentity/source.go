package workloadidentity

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
	GoogleMetadataIdentityURL   = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity"
	GoogleMetadataFlavorHeader  = "Metadata-Flavor"
	GoogleMetadataFlavorValue   = "Google"
	GoogleMetadataAudienceQuery = "audience"
	GoogleMetadataFormatQuery   = "format"
	GoogleMetadataFullFormat    = "full"
	HTTPBudget                  = 5 * time.Second
	ErrFmtSource                = "workloadidentity.Source: %w"
)

type TokenSource interface {
	core.Validatable
	Token(context.Context) (Token, error)
}

type Source struct {
	HTTP     *http.Client
	Audience core.APIEndpoint
}

func (s Source) Validate() error {
	if s.HTTP == nil {
		return fmt.Errorf(ErrFmtSource, ErrContract)
	}
	if err := s.Audience.Validate(); err != nil {
		return wrap(ErrFmtSource, err)
	}
	return nil
}

func (s Source) Token(ctx context.Context) (Token, error) {
	if err := s.Validate(); err != nil {
		return Token{}, err
	}
	if ctx == nil {
		return Token{}, fmt.Errorf(ErrFmtSource, errors.Join(ErrContract, core.ErrNilContext))
	}
	requestContext, cancel := context.WithTimeout(ctx, HTTPBudget)
	defer cancel()
	request := exchange.BoundedRequest[metadataIdentityTarget]{
		Target: metadataIdentityTarget{},
		Headers: core.HTTPHeaders{Values: []core.HTTPHeader{{
			Name: GoogleMetadataFlavorHeader, Value: GoogleMetadataFlavorValue,
		}}},
		Query: core.HTTPQuery{Parameters: []core.HTTPQueryParameter{
			{Name: GoogleMetadataAudienceQuery, Value: s.Audience.String()},
			{Name: GoogleMetadataFormatQuery, Value: GoogleMetadataFullFormat},
		}},
		Semantics:      core.HTTPRequestSemantics{Method: core.HTTPMethodGet, Replay: core.HTTPReplaySingleAttempt},
		ExpectedStatus: core.HTTPStatusOK,
	}
	policy := exchange.BoundedPolicy{
		AttemptTimeout:    core.NewNanosecondsDuration(HTTPBudget),
		RequestBodyLimit:  core.NewByteCount(1),
		ResponseBodyLimit: core.NewByteCount(TokenMaxBytes),
		Redirect:          core.HTTPRedirectReject,
	}
	exchangeClient, err := exchange.NewClient(s.HTTP)
	if err != nil {
		return Token{}, wrap(ErrFmtSource, err)
	}
	response, err := exchange.SendBounded(requestContext, exchangeClient, request, policy)
	if err != nil {
		return Token{}, wrap(ErrFmtSource, err)
	}
	token, err := ParseToken(string(response.Body))
	if err != nil {
		return Token{}, wrap(ErrFmtSource, err)
	}
	return token, nil
}

type metadataIdentityTarget struct{}

func (metadataIdentityTarget) Validate() error { return nil }

func (metadataIdentityTarget) String() string { return GoogleMetadataIdentityURL }

var _ core.Validatable = Source{}
var _ TokenSource = Source{}
