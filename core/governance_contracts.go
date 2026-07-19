package core

import (
	"encoding/json"
	"fmt"
)

const (
	ErrFmtGovernanceDocument       = "core.GovernanceDocument: %w"
	ErrFmtGovernanceDocumentSHA256 = "core.GovernanceDocument.SHA256: %w"
	GoModuleFileName               = "go.mod"
)

type GovernanceDocument uint8

const (
	GovernanceDocumentUnknown GovernanceDocument = iota
	GovernanceDocumentGoDoctrine
	GovernanceDocumentTestingProtocol
)

const (
	GovernanceDirectory                          = "_docs/governance"
	GovernanceDocumentTokenGoDoctrine            = "go_doctrine"
	GovernanceDocumentTokenTestingProtocol       = "testing_protocol"
	GoDoctrinePath                               = GovernanceDirectory + "/go_doctrine.md"
	TestingProtocolPath                          = GovernanceDirectory + "/testing_protocol.md"
	GoDoctrineSHA256Hex                          = "acba02627001f224f1f96b811f211ad66d879bf9c7483f8ec64f4ea7e3890380"
	TestingProtocolSHA256Hex                     = "24f321578e144e72a7720a58215d23d1491d81ee83d68dd6518c4fc4d8ca0ec0"
	GovernanceDocumentDefaultMaxBytes      int64 = 512 * 1024
)

func GovernanceDocuments() [2]GovernanceDocument {
	return [2]GovernanceDocument{GovernanceDocumentGoDoctrine, GovernanceDocumentTestingProtocol}
}

func (d GovernanceDocument) Validate() error {
	switch d {
	case GovernanceDocumentGoDoctrine, GovernanceDocumentTestingProtocol:
		return nil
	default:
		return fmt.Errorf(ErrFmtGovernanceDocument, ErrFoundationContract)
	}
}

func (d GovernanceDocument) IsValid() bool {
	return d == GovernanceDocumentGoDoctrine || d == GovernanceDocumentTestingProtocol
}

func (d GovernanceDocument) String() string {
	switch d {
	case GovernanceDocumentGoDoctrine:
		return GovernanceDocumentTokenGoDoctrine
	case GovernanceDocumentTestingProtocol:
		return GovernanceDocumentTokenTestingProtocol
	default:
		return ""
	}
}

func ParseGovernanceDocument(value string) (GovernanceDocument, error) {
	switch value {
	case GovernanceDocumentTokenGoDoctrine:
		return GovernanceDocumentGoDoctrine, nil
	case GovernanceDocumentTokenTestingProtocol:
		return GovernanceDocumentTestingProtocol, nil
	default:
		return GovernanceDocumentUnknown, fmt.Errorf(ErrFmtGovernanceDocument, ErrFoundationContract)
	}
}

func (d GovernanceDocument) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(d.String())
}

func (d *GovernanceDocument) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtGovernanceDocument, ErrFoundationContract)
	}
	parsed, err := ParseGovernanceDocument(value)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

func (d GovernanceDocument) Path() string {
	switch d {
	case GovernanceDocumentGoDoctrine:
		return GoDoctrinePath
	case GovernanceDocumentTestingProtocol:
		return TestingProtocolPath
	default:
		return ""
	}
}

func (d GovernanceDocument) ExpectedSHA256() (SHA256Hex, error) {
	var value string
	switch d {
	case GovernanceDocumentGoDoctrine:
		value = GoDoctrineSHA256Hex
	case GovernanceDocumentTestingProtocol:
		value = TestingProtocolSHA256Hex
	default:
		return SHA256Hex{}, fmt.Errorf(ErrFmtGovernanceDocumentSHA256, ErrFoundationContract)
	}
	return ParseSHA256Hex(value)
}
