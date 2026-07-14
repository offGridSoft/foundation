package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

type StorageProvider uint8

const (
	StorageProviderUnknown StorageProvider = iota
	StorageProviderGCS
	StorageProviderS3
)

func storageProviderNames() [StorageProviderS3 + 1]string {
	return [...]string{
		StorageProviderGCS: storageProviderTokenGCS,
		StorageProviderS3:  storageProviderTokenS3,
	}
}

const (
	storageProviderTokenGCS = "gcs"
	storageProviderTokenS3  = "s3"
)

func (p StorageProvider) String() string {
	if p.IsValid() {
		return storageProviderNames()[p]
	}
	return ""
}

func (p StorageProvider) IsValid() bool {
	return p > StorageProviderUnknown && int(p) < len(storageProviderNames()) && storageProviderNames()[p] != ""
}

func (p StorageProvider) Validate() error {
	if !p.IsValid() {
		return fmt.Errorf(ErrFmtStorageProvider, ErrFoundationContract)
	}
	return nil
}

func (p StorageProvider) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.String())
}

func ParseStorageProvider(token string) (StorageProvider, error) {
	for provider := StorageProviderGCS; int(provider) < len(storageProviderNames()); provider++ {
		if storageProviderNames()[provider] == token {
			return provider, nil
		}
	}
	return StorageProviderUnknown, fmt.Errorf(ErrFmtStorageProvider, ErrFoundationContract)
}

func (p *StorageProvider) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtStorageProvider, ErrFoundationContract)
	}
	parsed, err := ParseStorageProvider(token)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

type UploadMethod uint8

const (
	UploadMethodUnknown UploadMethod = iota
	UploadMethodSignedPUT
	UploadMethodResumableURI
)

func uploadMethodNames() [UploadMethodResumableURI + 1]string {
	return [...]string{
		UploadMethodSignedPUT:    uploadMethodTokenSignedPUT,
		UploadMethodResumableURI: uploadMethodTokenResumableURI,
	}
}

const (
	uploadMethodTokenSignedPUT    = "signed_put"
	uploadMethodTokenResumableURI = "resumable_uri" // #nosec G101 -- upload method protocol token, not a credential.
)

func (m UploadMethod) String() string {
	if m.IsValid() {
		return uploadMethodNames()[m]
	}
	return ""
}

func (m UploadMethod) IsValid() bool {
	return m > UploadMethodUnknown && int(m) < len(uploadMethodNames()) && uploadMethodNames()[m] != ""
}

func (m UploadMethod) Validate() error {
	if !m.IsValid() {
		return fmt.Errorf(ErrFmtUploadMethod, ErrFoundationContract)
	}
	return nil
}

func (m UploadMethod) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m.String())
}

func ParseUploadMethod(token string) (UploadMethod, error) {
	for method := UploadMethodSignedPUT; int(method) < len(uploadMethodNames()); method++ {
		if uploadMethodNames()[method] == token {
			return method, nil
		}
	}
	return UploadMethodUnknown, fmt.Errorf(ErrFmtUploadMethod, ErrFoundationContract)
}

func (m *UploadMethod) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtUploadMethod, ErrFoundationContract)
	}
	parsed, err := ParseUploadMethod(token)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

type SignedUploadURL struct {
	value string
}

func ParseSignedUploadURL(value string) (SignedUploadURL, error) {
	if err := ValidateHTTPSURL(value, HTTPSURLPolicy{
		MaxRunes:    HTTPSURLDefaultMaxRunes,
		RequirePath: true,
		AllowQuery:  true,
	}); err != nil {
		return SignedUploadURL{}, fmt.Errorf(ErrFmtSignedUploadURL, ErrFoundationContract)
	}
	return SignedUploadURL{value: value}, nil
}

func (u SignedUploadURL) String() string {
	return u.value
}

func (u SignedUploadURL) Validate() error {
	_, err := ParseSignedUploadURL(u.value)
	return err
}

func (u SignedUploadURL) MarshalJSON() ([]byte, error) {
	if err := u.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(u.value)
}

func (u *SignedUploadURL) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtSignedUploadURL, ErrFoundationContract)
	}
	parsed, err := ParseSignedUploadURL(value)
	if err != nil {
		return err
	}
	*u = parsed
	return u.Validate()
}

type UploadHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (h UploadHeader) Validate() error {
	if err := ValidateHTTPHeaderName(h.Name); err != nil {
		return fmt.Errorf(ErrFmtUploadHeader, ErrFoundationContract)
	}
	if err := ValidateHTTPHeaderValue(h.Value); err != nil {
		return fmt.Errorf(ErrFmtUploadHeader, ErrFoundationContract)
	}
	return nil
}

func ValidateUploadHeaders(headers []UploadHeader) error {
	if err := (CollectionCardinality{
		Length:  len(headers),
		Minimum: 0,
		Maximum: HTTPHeaderMaximumDefault,
	}).Validate(); err != nil {
		return fmt.Errorf(ErrFmtUploadHeader, ErrFoundationContract)
	}
	for index, header := range headers {
		if err := header.Validate(); err != nil {
			return err
		}
		for _, prior := range headers[:index] {
			if strings.EqualFold(prior.Name, header.Name) {
				return fmt.Errorf(ErrFmtUploadHeader, ErrFoundationContract)
			}
		}
	}
	return nil
}
