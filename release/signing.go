package release

import (
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const ErrFmtReleaseSignature = "release.Signature: %w"

type ReleaseSigner interface {
	SignRelease(payload []byte) (core.Ed25519SignatureHex, error)
}

func SignReleasePayload(signer ReleaseSigner, payload []byte) (core.Ed25519SignatureHex, error) {
	if signer == nil || len(payload) == 0 {
		return core.Ed25519SignatureHex{}, fmt.Errorf(ErrFmtReleaseSignature, core.ErrReleaseContract)
	}
	signature, err := signer.SignRelease(payload)
	if err != nil {
		return core.Ed25519SignatureHex{}, fmt.Errorf(ErrFmtReleaseSignature, err)
	}
	if err := signature.Validate(); err != nil {
		return core.Ed25519SignatureHex{}, fmt.Errorf(ErrFmtReleaseSignature, err)
	}
	return signature, nil
}
