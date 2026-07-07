package custody

import (
	"fmt"
	"net/url"
	"strings"

	json "github.com/goccy/go-json"
	"github.com/offGridSoft/foundation/core"
)

const (
	SchemaSessionOpenRequest   = "witness-custody-session-open-v1"
	SchemaSessionOpenResponse  = "witness-custody-session-targets-v1"
	SchemaFinalizeRequest      = "witness-custody-finalize-v1"
	SchemaReceipt              = "witness-custody-receipt-v1"
	ULIDTextLen                = 26
	ULIDUpperCrockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
)

type CustomerID struct {
	value string
}

func ParseCustomerID(value string) (CustomerID, error) {
	if !validULIDText(value) {
		return CustomerID{}, fmt.Errorf(ErrFmtULID, core.ErrCustodyContract)
	}
	return CustomerID{value: value}, nil
}

func (id CustomerID) String() string {
	return id.value
}

func (id CustomerID) Validate() error {
	_, err := ParseCustomerID(id.value)
	return err
}

func (id CustomerID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

func (id *CustomerID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtULID, core.ErrCustodyContract)
	}
	parsed, err := ParseCustomerID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

type SessionID struct {
	value string
}

func ParseSessionID(value string) (SessionID, error) {
	if !validULIDText(value) {
		return SessionID{}, fmt.Errorf(ErrFmtULID, core.ErrCustodyContract)
	}
	return SessionID{value: value}, nil
}

func (id SessionID) String() string {
	return id.value
}

func (id SessionID) Validate() error {
	_, err := ParseSessionID(id.value)
	return err
}

func (id SessionID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

func (id *SessionID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtULID, core.ErrCustodyContract)
	}
	parsed, err := ParseSessionID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

func validULIDText(value string) bool {
	if len(value) != ULIDTextLen {
		return false
	}
	for _, r := range value {
		if !validULIDRune(r) {
			return false
		}
	}
	return true
}

func validULIDRune(r rune) bool {
	return strings.ContainsRune(ULIDUpperCrockfordAlphabet, r)
}

type ArtifactName struct {
	value string
}

func ParseArtifactName(value string) (ArtifactName, error) {
	if strings.TrimSpace(value) == "" || strings.ContainsAny(value, `/\`) {
		return ArtifactName{}, fmt.Errorf(ErrFmtArtifactName, core.ErrCustodyContract)
	}
	return ArtifactName{value: value}, nil
}

func (n ArtifactName) String() string {
	return n.value
}

func (n ArtifactName) Validate() error {
	_, err := ParseArtifactName(n.value)
	return err
}

func (n ArtifactName) MarshalJSON() ([]byte, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(n.value)
}

func (n *ArtifactName) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtArtifactName, core.ErrCustodyContract)
	}
	parsed, err := ParseArtifactName(value)
	if err != nil {
		return err
	}
	*n = parsed
	return nil
}

type ObjectPath struct {
	value string
}

func ParseObjectPath(value string) (ObjectPath, error) {
	if strings.TrimSpace(value) == "" || strings.HasPrefix(value, "/") {
		return ObjectPath{}, fmt.Errorf(ErrFmtObjectPath, core.ErrCustodyContract)
	}
	if strings.Contains(value, "../") || strings.Contains(value, "//") {
		return ObjectPath{}, fmt.Errorf(ErrFmtObjectPath, core.ErrCustodyContract)
	}
	return ObjectPath{value: value}, nil
}

func (p ObjectPath) String() string {
	return p.value
}

func (p ObjectPath) Validate() error {
	_, err := ParseObjectPath(p.value)
	return err
}

func (p ObjectPath) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p.value)
}

func (p *ObjectPath) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtObjectPath, core.ErrCustodyContract)
	}
	parsed, err := ParseObjectPath(value)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

type SignedUploadURL struct {
	value string
}

func ParseSignedUploadURL(value string) (SignedUploadURL, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return SignedUploadURL{}, fmt.Errorf(ErrFmtSignedURL, err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return SignedUploadURL{}, fmt.Errorf(ErrFmtSignedURL, core.ErrCustodyContract)
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
		return fmt.Errorf(ErrFmtSignedURL, core.ErrCustodyContract)
	}
	parsed, err := ParseSignedUploadURL(value)
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

type ReleaseIdentity struct {
	Version     core.ProductVersion `json:"version"`
	Commit      BuildCommit         `json:"commit"`
	ManifestSHA core.SHA256Hex      `json:"tool_manifest_sha"`
}

func (r ReleaseIdentity) Validate() error {
	if err := r.Version.Validate(); err != nil {
		return fmt.Errorf(ErrFmtRelease, err)
	}
	if err := r.Commit.Validate(); err != nil {
		return fmt.Errorf(ErrFmtRelease, err)
	}
	if err := r.ManifestSHA.Validate(); err != nil {
		return fmt.Errorf(ErrFmtRelease, err)
	}
	return nil
}

type BuildCommit struct {
	value string
}

func ParseBuildCommit(value string) (BuildCommit, error) {
	if !validBuildCommitHex(value) {
		return BuildCommit{}, fmt.Errorf(ErrFmtRelease, core.ErrCustodyContract)
	}
	return BuildCommit{value: value}, nil
}

func (c BuildCommit) String() string {
	return c.value
}

func (c BuildCommit) Validate() error {
	_, err := ParseBuildCommit(c.value)
	return err
}

func (c BuildCommit) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.value)
}

func (c *BuildCommit) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtRelease, core.ErrCustodyContract)
	}
	parsed, err := ParseBuildCommit(value)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

func validBuildCommitHex(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	return core.IsLowerHex(value)
}

type OpenLeaseRef struct {
	LeaseID           core.LeaseID           `json:"lease_id"`
	DeviceFingerprint core.DeviceFingerprint `json:"device_fingerprint"`
}

func (r OpenLeaseRef) Validate() error {
	if err := r.LeaseID.Validate(); err != nil {
		return err
	}
	return r.DeviceFingerprint.Validate()
}

type Generation struct {
	value string
}

func ParseGeneration(value string) (Generation, error) {
	if strings.TrimSpace(value) == "" {
		return Generation{}, fmt.Errorf(ErrFmtGeneration, core.ErrCustodyContract)
	}
	return Generation{value: value}, nil
}

func (g Generation) String() string {
	return g.value
}

func (g Generation) Validate() error {
	_, err := ParseGeneration(g.value)
	return err
}

func (g Generation) MarshalJSON() ([]byte, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(g.value)
}

func (g *Generation) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtGeneration, core.ErrCustodyContract)
	}
	parsed, err := ParseGeneration(value)
	if err != nil {
		return err
	}
	*g = parsed
	return nil
}
