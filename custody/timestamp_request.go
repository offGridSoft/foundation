package custody

import (
	"crypto/sha256"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const RFC3161RequestVersion = 1

// rfc3161MessageImprint is the RFC 3161 section 2.4.1 MessageImprint carrying
// the SHA-256 custody imprint. The enclosing TimeStampReq SEQUENCE is encoded
// component-by-component in appendTimestampQueryContent so the load-bearing
// DER field order (version, messageImprint, certReq) is owned by the encoder,
// not by Go struct layout.
type rfc3161MessageImprint struct {
	HashAlgorithm pkix.AlgorithmIdentifier
	HashedMessage []byte
}

func rfc3161SHA256OID() asn1.ObjectIdentifier {
	return asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
}

// EncodeRFC3161TimestampQuery builds the DER TimeStampReq for a custody
// timestamp imprint: version 1, a SHA-256 message imprint, and certReq TRUE.
// reqPolicy, nonce, and extensions are intentionally absent — the custody
// contract owns no TSTInfo parser, so a nonce echo could never be verified
// and would be false assurance. The imprint must already be the
// domain-separated digest produced by DeriveTimestampImprint.
func EncodeRFC3161TimestampQuery(imprint core.SHA256Hex) ([]byte, error) {
	if err := imprint.Validate(); err != nil {
		return nil, fmt.Errorf(ErrFmtTimestampQuery, err)
	}
	hashed, err := hex.DecodeString(imprint.String())
	if err != nil || len(hashed) != sha256.Size {
		return nil, fmt.Errorf(ErrFmtTimestampQuery, core.ErrCustodyContract)
	}
	content, err := appendTimestampQueryContent(hashed)
	if err != nil {
		return nil, fmt.Errorf(ErrFmtTimestampQuery, core.ErrCustodyContract)
	}
	query, err := asn1.Marshal(asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSequence, IsCompound: true, Bytes: content})
	if err != nil || len(query) == 0 || len(query) > RFC3161DERMaximumBytes {
		return nil, fmt.Errorf(ErrFmtTimestampQuery, core.ErrCustodyContract)
	}
	return query, nil
}

// appendTimestampQueryContent emits the TimeStampReq SEQUENCE body in the
// exact RFC 3161 component order: version, messageImprint, certReq.
func appendTimestampQueryContent(hashed []byte) ([]byte, error) {
	content, err := asn1.Marshal(RFC3161RequestVersion)
	if err != nil {
		return nil, err
	}
	imprintDER, err := asn1.Marshal(rfc3161MessageImprint{
		HashAlgorithm: pkix.AlgorithmIdentifier{Algorithm: rfc3161SHA256OID(), Parameters: asn1.NullRawValue},
		HashedMessage: hashed,
	})
	if err != nil {
		return nil, err
	}
	certReqDER, err := asn1.Marshal(true)
	if err != nil {
		return nil, err
	}
	return append(append(content, imprintDER...), certReqDER...), nil
}
