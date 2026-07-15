package custody

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	RFC3161DERMaximumBytes = 64 << 10
	RFC3161ImprintDomain   = "foundation-witness-custody-rfc3161-" + core.ContractYear
)

type RFC3161Token struct {
	encoded string
}

type RFC3161Response struct {
	encoded string
}

func NewRFC3161Token(raw []byte) (RFC3161Token, error) {
	encoded, err := encodeRFC3161DER(raw)
	if err != nil {
		return RFC3161Token{}, err
	}
	token := RFC3161Token{encoded: encoded}
	if err := token.Validate(); err != nil {
		return RFC3161Token{}, err
	}
	return token, nil
}

func ParseRFC3161Token(value string) (RFC3161Token, error) {
	token := RFC3161Token{encoded: value}
	if err := token.Validate(); err != nil {
		return RFC3161Token{}, err
	}
	return token, nil
}

func (t RFC3161Token) String() string { return t.encoded }

func (t RFC3161Token) Validate() error {
	raw, err := decodeRFC3161DER(t.encoded)
	if err != nil {
		return err
	}
	return validateRFC3161Token(raw)
}

func (t RFC3161Token) Bytes() ([]byte, error) { return decodeRFC3161DER(t.encoded) }

func (t RFC3161Token) MarshalJSON() ([]byte, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(t.encoded)
}

func (t *RFC3161Token) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	parsed, err := ParseRFC3161Token(value)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

func NewRFC3161Response(raw []byte) (RFC3161Response, error) {
	encoded, err := encodeRFC3161DER(raw)
	if err != nil {
		return RFC3161Response{}, err
	}
	response := RFC3161Response{encoded: encoded}
	if err := response.Validate(); err != nil {
		return RFC3161Response{}, err
	}
	return response, nil
}

func ParseRFC3161Response(value string) (RFC3161Response, error) {
	response := RFC3161Response{encoded: value}
	if err := response.Validate(); err != nil {
		return RFC3161Response{}, err
	}
	return response, nil
}

func (r RFC3161Response) String() string { return r.encoded }

func (r RFC3161Response) Validate() error {
	raw, err := decodeRFC3161DER(r.encoded)
	if err != nil {
		return err
	}
	_, err = embeddedRFC3161Token(raw)
	return err
}

func (r RFC3161Response) Bytes() ([]byte, error) { return decodeRFC3161DER(r.encoded) }

func (r RFC3161Response) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(r.encoded)
}

func (r *RFC3161Response) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	parsed, err := ParseRFC3161Response(value)
	if err != nil {
		return err
	}
	*r = parsed
	return nil
}

type TimestampProof struct {
	BundleRoot    core.BLAKE3Hex     `json:"bundle_root"`
	ImprintSHA256 core.SHA256Hex     `json:"imprint_sha256"`
	Token         RFC3161Token       `json:"token"`
	Response      RFC3161Response    `json:"response"`
	TimestampedAt core.UnixNanoTime  `json:"timestamped_at"`
	Authority     TimestampAuthority `json:"authority"`
}

type TimestampProofInput struct {
	BundleRoot    core.BLAKE3Hex
	Token         RFC3161Token
	Response      RFC3161Response
	TimestampedAt core.UnixNanoTime
	Authority     TimestampAuthority
}

func (p TimestampProof) Canonical(dst []byte) ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return appendTimestampProofJSON(dst, p)
}

func (p TimestampProof) MarshalJSON() ([]byte, error) {
	return p.Canonical(nil)
}

func appendTimestampProofJSON(dst []byte, proof TimestampProof) ([]byte, error) {
	dst = append(dst, '{')
	var err error
	dst, err = core.AppendJSONField(dst, core.JSONFieldAuthority, proof.Authority)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldBundleRoot, proof.BundleRoot)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldImprintSHA256, proof.ImprintSHA256)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldToken, proof.Token)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldResponse, proof.Response)
	dst, err = core.AppendJSONFieldAfterComma(dst, err, core.JSONFieldTimestampedAt, proof.TimestampedAt)
	if err != nil {
		return nil, err
	}
	return append(dst, '}'), nil
}

func BuildTimestampProof(input TimestampProofInput) (TimestampProof, error) {
	imprint, err := DeriveTimestampImprint(input.BundleRoot)
	if err != nil {
		return TimestampProof{}, err
	}
	proof := TimestampProof{
		Authority: input.Authority, BundleRoot: input.BundleRoot, ImprintSHA256: imprint,
		Token: input.Token, Response: input.Response, TimestampedAt: input.TimestampedAt,
	}
	if err := proof.Validate(); err != nil {
		return TimestampProof{}, err
	}
	return proof, nil
}

func (p TimestampProof) Validate() error {
	if err := validateTimestampProofIdentity(p); err != nil {
		return err
	}
	return validateTimestampProofToken(p)
}

func validateTimestampProofIdentity(proof TimestampProof) error {
	if err := proof.Authority.Validate(); err != nil {
		return err
	}
	if err := proof.BundleRoot.Validate(); err != nil {
		return fmt.Errorf(ErrFmtTimestamp, err)
	}
	wantImprint, err := DeriveTimestampImprint(proof.BundleRoot)
	if err != nil || proof.ImprintSHA256 != wantImprint {
		return fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	if err := core.ValidateRequiredUnixNanoTime(proof.TimestampedAt); err != nil {
		return fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	return nil
}

func validateTimestampProofToken(proof TimestampProof) error {
	token, err := proof.Token.Bytes()
	if err != nil {
		return err
	}
	response, err := proof.Response.Bytes()
	if err != nil {
		return err
	}
	embedded, err := embeddedRFC3161Token(response)
	if err != nil || validateRFC3161Token(token) != nil || subtle.ConstantTimeCompare(token, embedded) != 1 {
		return fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	return nil
}

func validateRFC3161Token(token []byte) error {
	outer, err := decodeRFC3161Sequence(token)
	if err != nil {
		return err
	}
	var contentType asn1.ObjectIdentifier
	content, err := asn1.Unmarshal(outer.Bytes, &contentType)
	if err != nil || !contentType.Equal(rfc3161SignedDataOID()) {
		return fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	var signedData asn1.RawValue
	trailing, err := asn1.Unmarshal(content, &signedData)
	if err != nil || !validRFC3161SignedData(signedData, trailing) {
		return fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	return nil
}

func rfc3161SignedDataOID() asn1.ObjectIdentifier {
	return asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
}

func validRFC3161SignedData(value asn1.RawValue, trailing []byte) bool {
	return len(trailing) == 0 && value.Class == 2 && value.Tag == 0 && value.IsCompound
}

func DeriveTimestampImprint(bundleRoot core.BLAKE3Hex) (core.SHA256Hex, error) {
	if err := bundleRoot.Validate(); err != nil {
		return core.SHA256Hex{}, fmt.Errorf(ErrFmtTimestamp, err)
	}
	raw, err := hex.DecodeString(bundleRoot.String())
	if err != nil {
		return core.SHA256Hex{}, fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte(RFC3161ImprintDomain))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(raw)
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	return core.NewSHA256Hex(digest), nil
}

func encodeRFC3161DER(raw []byte) (string, error) {
	if len(raw) == 0 || len(raw) > RFC3161DERMaximumBytes {
		return "", fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func decodeRFC3161DER(value string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(raw) == 0 || len(raw) > RFC3161DERMaximumBytes || base64.StdEncoding.EncodeToString(raw) != value {
		return nil, fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	return raw, nil
}

func embeddedRFC3161Token(response []byte) ([]byte, error) {
	outer, err := decodeRFC3161Sequence(response)
	if err != nil {
		return nil, err
	}
	var status asn1.RawValue
	content, err := asn1.Unmarshal(outer.Bytes, &status)
	if err != nil || !validRFC3161Status(status) {
		return nil, fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	var token asn1.RawValue
	trailing, err := asn1.Unmarshal(content, &token)
	if err != nil || !validRFC3161Sequence(token, trailing) {
		return nil, fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	return append([]byte(nil), token.FullBytes...), nil
}

func decodeRFC3161Sequence(raw []byte) (asn1.RawValue, error) {
	var value asn1.RawValue
	trailing, err := asn1.Unmarshal(raw, &value)
	if err != nil || !validRFC3161Sequence(value, trailing) {
		return asn1.RawValue{}, fmt.Errorf(ErrFmtTimestamp, core.ErrCustodyContract)
	}
	return value, nil
}

func validRFC3161Sequence(value asn1.RawValue, trailing []byte) bool {
	return len(trailing) == 0 && value.Class == 0 && value.Tag == asn1.TagSequence && value.IsCompound
}

func validRFC3161Status(value asn1.RawValue) bool {
	return validRFC3161Sequence(value, nil) && successfulRFC3161Status(value.Bytes)
}

func successfulRFC3161Status(data []byte) bool {
	var status int
	_, err := asn1.Unmarshal(data, &status)
	return err == nil && (status == 0 || status == 1)
}
