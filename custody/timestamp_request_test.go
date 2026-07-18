package custody

import (
	"bytes"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

const rfc3161GoldenQueryPrefixHex = "30390201013031300d060960864801650304020105000420"
const rfc3161GoldenQuerySuffixHex = "0101ff"

func TestEncodeRFC3161TimestampQueryGoldenDER(t *testing.T) {
	t.Parallel()

	imprint, err := DeriveTimestampImprint(mustBundleRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	query, err := EncodeRFC3161TimestampQuery(imprint)
	if err != nil {
		t.Fatalf("EncodeRFC3161TimestampQuery() error = %v", err)
	}
	want := rfc3161GoldenQueryPrefixHex + imprint.String() + rfc3161GoldenQuerySuffixHex
	if got := hex.EncodeToString(query); got != want {
		t.Fatalf("query DER = %s, want %s", got, want)
	}
	again, err := EncodeRFC3161TimestampQuery(imprint)
	if err != nil {
		t.Fatalf("EncodeRFC3161TimestampQuery() second call error = %v", err)
	}
	if !bytes.Equal(query, again) {
		t.Fatalf("query DER = %x, want deterministic %x", again, query)
	}
}

func TestEncodeRFC3161TimestampQueryRoundTripsTypedFields(t *testing.T) {
	t.Parallel()

	imprint, err := DeriveTimestampImprint(mustBundleRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	query, err := EncodeRFC3161TimestampQuery(imprint)
	if err != nil {
		t.Fatalf("EncodeRFC3161TimestampQuery() error = %v", err)
	}
	content := requireTimestampQuerySequence(t, query)
	rest := requireTimestampQueryVersion(t, content)
	rest = requireTimestampQueryImprint(t, rest, imprint)
	requireTimestampQueryCertReq(t, rest)
}

func requireTimestampQuerySequence(t *testing.T, query []byte) []byte {
	t.Helper()
	var outer asn1.RawValue
	trailing, err := asn1.Unmarshal(query, &outer)
	if err != nil {
		t.Fatalf("asn1.Unmarshal(query) error = %v", err)
	}
	if len(trailing) != 0 {
		t.Fatalf("query trailing bytes = %d, want 0", len(trailing))
	}
	if outer.Class != asn1.ClassUniversal || outer.Tag != asn1.TagSequence || !outer.IsCompound {
		t.Fatalf("outer = class %d tag %d compound %v, want universal SEQUENCE", outer.Class, outer.Tag, outer.IsCompound)
	}
	return outer.Bytes
}

func requireTimestampQueryVersion(t *testing.T, content []byte) []byte {
	t.Helper()
	var version int
	rest, err := asn1.Unmarshal(content, &version)
	if err != nil {
		t.Fatalf("version decode error = %v", err)
	}
	if version != RFC3161RequestVersion {
		t.Fatalf("version = %d, want %d", version, RFC3161RequestVersion)
	}
	return rest
}

func requireTimestampQueryImprint(t *testing.T, content []byte, imprint core.SHA256Hex) []byte {
	t.Helper()
	var imprintField rfc3161MessageImprint
	rest, err := asn1.Unmarshal(content, &imprintField)
	if err != nil {
		t.Fatalf("message imprint decode error = %v", err)
	}
	if !imprintField.HashAlgorithm.Algorithm.Equal(rfc3161SHA256OID()) {
		t.Fatalf("hash algorithm = %v, want %v", imprintField.HashAlgorithm.Algorithm, rfc3161SHA256OID())
	}
	wantHashed, err := hex.DecodeString(imprint.String())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(imprintField.HashedMessage, wantHashed) {
		t.Fatalf("hashed message = %x, want %x", imprintField.HashedMessage, wantHashed)
	}
	return rest
}

func requireTimestampQueryCertReq(t *testing.T, content []byte) {
	t.Helper()
	var certReq bool
	rest, err := asn1.Unmarshal(content, &certReq)
	if err != nil {
		t.Fatalf("certReq decode error = %v", err)
	}
	if !certReq {
		t.Fatal("certReq = false, want true")
	}
	if len(rest) != 0 {
		t.Fatalf("sequence trailing bytes = %d, want 0", len(rest))
	}
}

func TestEncodeRFC3161TimestampQueryDistinctImprintsDiverge(t *testing.T) {
	t.Parallel()

	first, err := DeriveTimestampImprint(mustBundleRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	second, err := DeriveTimestampImprint(mustBLAKE3(t, "a"))
	if err != nil {
		t.Fatal(err)
	}
	firstQuery, err := EncodeRFC3161TimestampQuery(first)
	if err != nil {
		t.Fatalf("EncodeRFC3161TimestampQuery(first) error = %v", err)
	}
	secondQuery, err := EncodeRFC3161TimestampQuery(second)
	if err != nil {
		t.Fatalf("EncodeRFC3161TimestampQuery(second) error = %v", err)
	}
	if bytes.Equal(firstQuery, secondQuery) {
		t.Fatalf("distinct imprints produced identical query DER %x", firstQuery)
	}
	if len(firstQuery) != len(secondQuery) {
		t.Fatalf("query lengths = %d and %d, want identical fixed framing", len(firstQuery), len(secondQuery))
	}
}

func TestEncodeRFC3161TimestampQueryRejectsZeroImprint(t *testing.T) {
	t.Parallel()

	query, err := EncodeRFC3161TimestampQuery(core.SHA256Hex{})
	if !errors.Is(err, core.ErrFoundationContract) {
		t.Fatalf("EncodeRFC3161TimestampQuery(zero) error = %v, want %v", err, core.ErrFoundationContract)
	}
	if query != nil {
		t.Fatalf("query = %x, want nil on rejection", query)
	}
}
