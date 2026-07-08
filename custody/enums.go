package custody

import (
	"fmt"

	"encoding/json"
	"github.com/offGridSoft/foundation/core"
)

type StorageProvider uint8

const (
	storageProviderInvalid StorageProvider = iota
	StorageProviderGCS
	StorageProviderS3
)

var storageProviderNames = [...]string{
	StorageProviderGCS: "gcs",
	StorageProviderS3:  "s3",
}

func (p StorageProvider) String() string {
	if p.IsValid() {
		return storageProviderNames[p]
	}
	return ""
}

func (p StorageProvider) IsValid() bool {
	return p > storageProviderInvalid && int(p) < len(storageProviderNames) && storageProviderNames[p] != ""
}

func (p StorageProvider) Validate() error {
	if !p.IsValid() {
		return fmt.Errorf(ErrFmtStorage, core.ErrCustodyContract)
	}
	return nil
}

func (p StorageProvider) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.String())
}

func (p *StorageProvider) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtStorage, core.ErrCustodyContract)
	}
	for provider := StorageProviderGCS; int(provider) < len(storageProviderNames); provider++ {
		if storageProviderNames[provider] == token {
			*p = provider
			return nil
		}
	}
	return fmt.Errorf(ErrFmtStorage, core.ErrCustodyContract)
}

type UploadMethod uint8

const (
	uploadMethodInvalid UploadMethod = iota
	UploadMethodSignedPUT
	UploadMethodResumableURI
)

var uploadMethodNames = [...]string{
	UploadMethodSignedPUT:    "signed_put",
	UploadMethodResumableURI: "resumable_uri",
}

func (m UploadMethod) String() string {
	if m.IsValid() {
		return uploadMethodNames[m]
	}
	return ""
}

func (m UploadMethod) IsValid() bool {
	return m > uploadMethodInvalid && int(m) < len(uploadMethodNames) && uploadMethodNames[m] != ""
}

func (m UploadMethod) Validate() error {
	if !m.IsValid() {
		return fmt.Errorf(ErrFmtStorage, core.ErrCustodyContract)
	}
	return nil
}

func (m UploadMethod) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m.String())
}

func (m *UploadMethod) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtStorage, core.ErrCustodyContract)
	}
	for method := UploadMethodSignedPUT; int(method) < len(uploadMethodNames); method++ {
		if uploadMethodNames[method] == token {
			*m = method
			return nil
		}
	}
	return fmt.Errorf(ErrFmtStorage, core.ErrCustodyContract)
}

type RetentionClass uint8

const (
	retentionClassInvalid RetentionClass = iota
	RetentionClassConditional
	RetentionClassPrepaid
)

var retentionClassNames = [...]string{
	RetentionClassConditional: "conditional",
	RetentionClassPrepaid:     "prepaid",
}

func (c RetentionClass) String() string {
	if c.IsValid() {
		return retentionClassNames[c]
	}
	return ""
}

func (c RetentionClass) IsValid() bool {
	return c > retentionClassInvalid && int(c) < len(retentionClassNames) && retentionClassNames[c] != ""
}

func (c RetentionClass) Validate() error {
	if !c.IsValid() {
		return fmt.Errorf(ErrFmtRetention, core.ErrCustodyContract)
	}
	return nil
}

func (c RetentionClass) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.String())
}

func (c *RetentionClass) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtRetention, core.ErrCustodyContract)
	}
	for class := RetentionClassConditional; int(class) < len(retentionClassNames); class++ {
		if retentionClassNames[class] == token {
			*c = class
			return nil
		}
	}
	return fmt.Errorf(ErrFmtRetention, core.ErrCustodyContract)
}
