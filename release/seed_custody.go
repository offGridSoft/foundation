package release

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	GarbleCustodySeedBytes     = 64
	GarbleCustodySeedTextBytes = 88
	GarbleSeedDerivationDomain = "foundation-garble-seed-" + core.ContractYear
)

type GarbleCustodySeed struct {
	value [GarbleCustodySeedBytes]byte
}

func ParseGarbleCustodySeed(value string) (GarbleCustodySeed, error) {
	if len(value) != GarbleCustodySeedTextBytes {
		return GarbleCustodySeed{}, fmt.Errorf(ErrFmtCustodySeed, core.ErrReleaseContract)
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(raw) != GarbleCustodySeedBytes || base64.StdEncoding.EncodeToString(raw) != value || allZeroSeed(raw) {
		return GarbleCustodySeed{}, fmt.Errorf(ErrFmtCustodySeed, core.ErrReleaseContract)
	}
	var seed GarbleCustodySeed
	copy(seed.value[:], raw)
	return seed, nil
}

func NewGarbleCustodySeed(value []byte) (GarbleCustodySeed, error) {
	if len(value) != GarbleCustodySeedBytes || allZeroSeed(value) {
		return GarbleCustodySeed{}, fmt.Errorf(ErrFmtCustodySeed, core.ErrReleaseContract)
	}
	var seed GarbleCustodySeed
	copy(seed.value[:], value)
	return seed, nil
}

func allZeroSeed(value []byte) bool {
	for _, part := range value {
		if part != 0 {
			return false
		}
	}
	return true
}

func (s GarbleCustodySeed) Validate() error {
	_, err := ParseGarbleCustodySeed(base64.StdEncoding.EncodeToString(s.value[:]))
	return err
}

func (s GarbleCustodySeed) Bytes() []byte {
	out := make([]byte, len(s.value))
	copy(out, s.value[:])
	return out
}

// MarshalText emits the one canonical Secret Manager representation accepted
// by ParseGarbleCustodySeed. It exists so persistence adapters never duplicate
// the seed's base64 protocol.
func (s GarbleCustodySeed) MarshalText() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return []byte(base64.StdEncoding.EncodeToString(s.value[:])), nil
}

func (s GarbleCustodySeed) SHA256() core.SHA256Hex {
	sum := sha256.Sum256(s.value[:])
	return core.NewSHA256Hex(sum)
}

func (s GarbleCustodySeed) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(base64.StdEncoding.EncodeToString(s.value[:]))
}

func (s GarbleCustodySeed) EffectiveSeed(releaseID ReleaseID) (GarbleSeed, error) {
	if err := s.Validate(); err != nil {
		return GarbleSeed{}, err
	}
	if err := releaseID.Validate(); err != nil {
		return GarbleSeed{}, err
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte(GarbleSeedDerivationDomain))
	_, _ = hash.Write([]byte{core.SignedMessageSep})
	_, _ = hash.Write([]byte(releaseID.String()))
	_, _ = hash.Write([]byte{core.SignedMessageSep})
	_, _ = hash.Write(s.value[:])
	effective := base64.RawStdEncoding.EncodeToString(hash.Sum(nil)[:GarbleSeedBytes])
	return ParseRequiredGarbleSeed(effective)
}

func (s *GarbleCustodySeed) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtCustodySeed, core.ErrReleaseContract)
	}
	parsed, err := ParseGarbleCustodySeed(value)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

type SeedRequest struct {
	Version   core.ProductVersion `json:"version"`
	ReleaseID ReleaseID           `json:"release_id"`
	Commit    core.BuildCommit    `json:"commit"`
	Product   core.Product        `json:"product"`
}

func (r SeedRequest) Validate() error {
	if err := r.Product.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtSeedRequest, err)
	}
	if err := r.Version.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtSeedRequest, err)
	}
	if err := r.ReleaseID.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtSeedRequest, err)
	}
	if err := r.Commit.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtSeedRequest, err)
	}
	if err := ValidateReleaseIDIdentity(r.ReleaseID, r.Product, r.Version, r.Commit); err != nil {
		return wrapReleaseContract(ErrFmtSeedRequest, err)
	}
	return nil
}

func (r SeedRequest) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	dst := []byte{'{'}
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldProduct, r.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, r.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReleaseID, r.ReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCommit, r.Commit)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

type SeedGrantBody struct {
	Version    core.ProductVersion `json:"version"`
	ReleaseID  ReleaseID           `json:"release_id"`
	Commit     core.BuildCommit    `json:"commit"`
	SeedSHA256 core.SHA256Hex      `json:"seed_sha256"`
	IssuedAt   core.UnixNanoTime   `json:"issued_at"`
	Schema     core.SchemaID       `json:"schema"`
	Seed       GarbleCustodySeed   `json:"seed"`
	Product    core.Product        `json:"product"`
}

func BuildSeedGrantBody(request SeedRequest, seed GarbleCustodySeed, issuedAt core.UnixNanoTime) (SeedGrantBody, error) {
	if err := request.Validate(); err != nil {
		return SeedGrantBody{}, err
	}
	if err := seed.Validate(); err != nil {
		return SeedGrantBody{}, err
	}
	body := SeedGrantBody{
		Version: request.Version, ReleaseID: request.ReleaseID, Commit: request.Commit,
		SeedSHA256: seed.SHA256(), IssuedAt: issuedAt, Schema: core.SchemaReleaseSeedGrant,
		Seed: seed, Product: request.Product,
	}
	if err := body.Validate(); err != nil {
		return SeedGrantBody{}, err
	}
	return body, nil
}

func VerifySeedGrant(grant core.Signed[SeedGrantBody], request SeedRequest, keyring core.SigningKeyring) error {
	if err := request.Validate(); err != nil {
		return err
	}
	if err := grant.Verify(keyring); err != nil {
		return err
	}
	if grant.Body.Request() != request {
		return fmt.Errorf(ErrFmtSeedGrant, core.ErrReleaseContract)
	}
	return nil
}

func (b SeedGrantBody) Validate() error {
	if b.Schema != core.SchemaReleaseSeedGrant {
		return fmt.Errorf(ErrFmtSeedGrant, core.ErrReleaseContract)
	}
	if err := (SeedRequest{Product: b.Product, Version: b.Version, ReleaseID: b.ReleaseID, Commit: b.Commit}).Validate(); err != nil {
		return wrapReleaseContract(ErrFmtSeedGrant, err)
	}
	if err := b.Seed.Validate(); err != nil {
		return wrapReleaseContract(ErrFmtSeedGrant, err)
	}
	if err := b.SeedSHA256.Validate(); err != nil || b.SeedSHA256 != b.Seed.SHA256() {
		return fmt.Errorf(ErrFmtSeedGrant, core.ErrReleaseContract)
	}
	if err := core.ValidateRequiredUnixNanoTime(b.IssuedAt); err != nil {
		return wrapReleaseContract(ErrFmtSeedGrant, err)
	}
	return nil
}

func (b SeedGrantBody) Canonical(dst []byte) ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldSchema, b.Schema)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldProduct, b.Product)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldVersion, b.Version)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldReleaseID, b.ReleaseID)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldCommit, b.Commit)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldSeedSHA256, b.SeedSHA256)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldSeed, b.Seed)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, jsonFieldIssuedAt, b.IssuedAt)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func (b SeedGrantBody) MarshalJSON() ([]byte, error) {
	return b.Canonical(nil)
}

func (b SeedGrantBody) SigningSchema() core.SchemaID {
	return b.Schema
}

func (b SeedGrantBody) Request() SeedRequest {
	return SeedRequest{Product: b.Product, Version: b.Version, ReleaseID: b.ReleaseID, Commit: b.Commit}
}

func (b SeedGrantBody) EffectiveSeed() (GarbleSeed, error) {
	if err := b.Validate(); err != nil {
		return GarbleSeed{}, err
	}
	return b.Seed.EffectiveSeed(b.ReleaseID)
}

var _ core.CanonicalBody = SeedGrantBody{}
