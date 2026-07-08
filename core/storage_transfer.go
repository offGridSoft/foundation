package core

import (
	"encoding/json"
	"fmt"
)

type StorageProvider uint8

const (
	StorageProviderUnknown StorageProvider = iota
	StorageProviderGCS
	StorageProviderS3
)

var storageProviderNames = [...]string{
	StorageProviderGCS: storageProviderTokenGCS,
	StorageProviderS3:  storageProviderTokenS3,
}

const (
	storageProviderTokenGCS = "gcs"
	storageProviderTokenS3  = "s3"
)

func (p StorageProvider) String() string {
	if p.IsValid() {
		return storageProviderNames[p]
	}
	return ""
}

func (p StorageProvider) IsValid() bool {
	return p > StorageProviderUnknown && int(p) < len(storageProviderNames) && storageProviderNames[p] != ""
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
	for provider := StorageProviderGCS; int(provider) < len(storageProviderNames); provider++ {
		if storageProviderNames[provider] == token {
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

var uploadMethodNames = [...]string{
	UploadMethodSignedPUT:    uploadMethodTokenSignedPUT,
	UploadMethodResumableURI: uploadMethodTokenResumableURI,
}

const (
	uploadMethodTokenSignedPUT    = "signed_put"
	uploadMethodTokenResumableURI = "resumable_uri" // #nosec G101 -- upload method protocol token, not a credential.
)

func (m UploadMethod) String() string {
	if m.IsValid() {
		return uploadMethodNames[m]
	}
	return ""
}

func (m UploadMethod) IsValid() bool {
	return m > UploadMethodUnknown && int(m) < len(uploadMethodNames) && uploadMethodNames[m] != ""
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
	for method := UploadMethodSignedPUT; int(method) < len(uploadMethodNames); method++ {
		if uploadMethodNames[method] == token {
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
