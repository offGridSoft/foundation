package core

import (
	"fmt"
	"strings"

	"encoding/json"
)

const (
	APIHeaderXRequestID  = "X-Request-Id"
	APIRequestIDMissing  = "missing"
	APIRequestIDMaxRunes = 256

	APICodeTokenNotFound           = "not_found"
	APICodeTokenInvalidInput       = "invalid_input"
	APICodeTokenConflict           = "conflict"
	APICodeTokenUnauthorized       = "unauthorized"
	APICodeTokenForbidden          = "forbidden"
	APICodeTokenPayloadTooLarge    = "payload_too_large" // #nosec G101 -- public API error token, not a credential.
	APICodeTokenServiceUnavailable = "service_unavailable"
	APICodeTokenInternal           = "internal"

	ErrFmtAPIRequestID = "core.APIRequestID: %w"
	ErrFmtAPICode      = "core.APICode: %w"
	ErrFmtAPIErrorBody = "core.APIErrorBody: %w"
	ErrFmtAPIEnvelope  = "core.APIEnvelope: %w"
)

type APIRequestID struct {
	value string
}

func NewAPIRequestID(value string) APIRequestID {
	runes := []rune(value)
	if len(runes) > APIRequestIDMaxRunes {
		value = string(runes[:APIRequestIDMaxRunes])
	}
	if value == "" {
		value = APIRequestIDMissing
	}
	return APIRequestID{value: value}
}

func (id APIRequestID) String() string {
	return id.value
}

func (id APIRequestID) Validate() error {
	if err := ValidateOpaqueToken(id.value, APIRequestIDMaxRunes); err != nil {
		return fmt.Errorf(ErrFmtAPIRequestID, ErrFoundationContract)
	}
	return nil
}

func (id APIRequestID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

func (id *APIRequestID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtAPIRequestID, ErrFoundationContract)
	}
	*id = NewAPIRequestID(value)
	return id.Validate()
}

type APICode uint8

const (
	apiCodeInvalid APICode = iota
	APICodeNotFound
	APICodeInvalidInput
	APICodeConflict
	APICodeUnauthorized
	APICodeForbidden
	APICodePayloadTooLarge
	APICodeServiceUnavailable
	APICodeInternal
)

var apiCodeNames = [...]string{
	APICodeNotFound:           APICodeTokenNotFound,
	APICodeInvalidInput:       APICodeTokenInvalidInput,
	APICodeConflict:           APICodeTokenConflict,
	APICodeUnauthorized:       APICodeTokenUnauthorized,
	APICodeForbidden:          APICodeTokenForbidden,
	APICodePayloadTooLarge:    APICodeTokenPayloadTooLarge,
	APICodeServiceUnavailable: APICodeTokenServiceUnavailable,
	APICodeInternal:           APICodeTokenInternal,
}

func (c APICode) String() string {
	if c.IsValid() {
		return apiCodeNames[c]
	}
	return ""
}

func (c APICode) IsValid() bool {
	return c > apiCodeInvalid && int(c) < len(apiCodeNames) && apiCodeNames[c] != ""
}

func (c APICode) Validate() error {
	if !c.IsValid() {
		return fmt.Errorf(ErrFmtAPICode, ErrFoundationContract)
	}
	return nil
}

func ParseAPICode(token string) (APICode, error) {
	for code := APICodeNotFound; int(code) < len(apiCodeNames); code++ {
		if apiCodeNames[code] == token {
			return code, nil
		}
	}
	return apiCodeInvalid, fmt.Errorf(ErrFmtAPICode, ErrFoundationContract)
}

func (c APICode) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.String())
}

func (c *APICode) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtAPICode, ErrFoundationContract)
	}
	parsed, err := ParseAPICode(token)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

type APIErrorBody struct {
	Message string  `json:"message"`
	Tip     string  `json:"tip,omitempty"`
	Code    APICode `json:"code"`
}

func (b APIErrorBody) Validate() error {
	if err := b.Code.Validate(); err != nil {
		return fmt.Errorf(ErrFmtAPIErrorBody, err)
	}
	if b.Message == "" {
		return fmt.Errorf(ErrFmtAPIErrorBody, ErrFoundationContract)
	}
	if b.Tip != "" && strings.TrimSpace(b.Tip) == "" {
		return fmt.Errorf(ErrFmtAPIErrorBody, ErrFoundationContract)
	}
	return nil
}

type APIEnvelope[T any] struct {
	Data      *T            `json:"data"`
	Error     *APIErrorBody `json:"error"`
	RequestID APIRequestID  `json:"request_id"`
}

func (e APIEnvelope[T]) Validate() error {
	if err := e.RequestID.Validate(); err != nil {
		return fmt.Errorf(ErrFmtAPIEnvelope, err)
	}
	switch {
	case e.Data != nil && e.Error == nil:
		return nil
	case e.Data == nil && e.Error != nil:
		return validateAPIEnvelopeError(e.Error)
	default:
		return fmt.Errorf(ErrFmtAPIEnvelope, ErrFoundationContract)
	}
}

func (e APIEnvelope[T]) ValidateSuccess() error {
	if err := e.Validate(); err != nil {
		return err
	}
	if e.Data == nil {
		return fmt.Errorf(ErrFmtAPIEnvelope, ErrFoundationContract)
	}
	return nil
}

func (e APIEnvelope[T]) ValidateFailure() error {
	if err := e.Validate(); err != nil {
		return err
	}
	if e.Error == nil {
		return fmt.Errorf(ErrFmtAPIEnvelope, ErrFoundationContract)
	}
	return nil
}

func validateAPIEnvelopeError(body *APIErrorBody) error {
	if err := body.Validate(); err != nil {
		return fmt.Errorf(ErrFmtAPIEnvelope, err)
	}
	return nil
}
