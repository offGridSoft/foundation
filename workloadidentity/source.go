package workloadidentity

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	GoogleMetadataIdentityURL     = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity"
	GoogleMetadataFlavorHeader    = "Metadata-Flavor"
	GoogleMetadataFlavorValue     = "Google"
	GoogleMetadataAudienceQuery   = "audience="
	GoogleMetadataFullFormatQuery = "&format=full"
	HTTPBudget                    = 5 * time.Second
	ErrFmtSource                  = "workloadidentity.Source: %w"
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
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, s.metadataURL(), nil)
	if err != nil {
		return Token{}, wrap(ErrFmtSource, err)
	}
	request.Header.Set(GoogleMetadataFlavorHeader, GoogleMetadataFlavorValue)
	client := *s.HTTP
	client.CheckRedirect = refuseRedirect
	response, err := client.Do(request)
	if err != nil {
		return Token{}, wrap(ErrFmtSource, err)
	}
	return parseResponse(response)
}

func parseResponse(response *http.Response) (Token, error) {
	if response == nil || response.Body == nil {
		return Token{}, fmt.Errorf(ErrFmtSource, ErrContract)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != core.HTTPStatusOK {
		return Token{}, fmt.Errorf(ErrFmtSource, ErrContract)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, TokenMaxBytes+1))
	if err != nil || len(body) == 0 || len(body) > TokenMaxBytes {
		return Token{}, fmt.Errorf(ErrFmtSource, ErrContract)
	}
	token, err := ParseToken(string(body))
	if err != nil {
		return Token{}, wrap(ErrFmtSource, err)
	}
	return token, nil
}

func (s Source) metadataURL() string {
	return GoogleMetadataIdentityURL + "?" + GoogleMetadataAudienceQuery + url.QueryEscape(s.Audience.String()) + GoogleMetadataFullFormatQuery
}
func refuseRedirect(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }

var _ core.Validatable = Source{}
var _ TokenSource = Source{}
