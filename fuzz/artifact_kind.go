package fuzz

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

// ArtifactKind is the closed identity of a retained fuzz corpus or crasher.
type ArtifactKind uint8

const (
	ArtifactKindUnknown ArtifactKind = iota
	ArtifactKindCorpus
	ArtifactKindCrasher
)

const (
	ArtifactKindNameUnknown = "unknown"
	ArtifactKindNameCorpus  = "fuzz-corpus"
	ArtifactKindNameCrasher = "fuzz-crasher"
	ErrFmtArtifactKind      = "fuzz.ArtifactKind: %w"
)

func (k ArtifactKind) String() string {
	switch k {
	case ArtifactKindCorpus:
		return ArtifactKindNameCorpus
	case ArtifactKindCrasher:
		return ArtifactKindNameCrasher
	default:
		return ArtifactKindNameUnknown
	}
}

func (k ArtifactKind) IsValid() bool {
	return k == ArtifactKindCorpus || k == ArtifactKindCrasher
}

func (k ArtifactKind) Validate() error {
	if !k.IsValid() {
		return fmt.Errorf(ErrFmtArtifactKind, core.ErrFuzzContract)
	}
	return nil
}

func (k ArtifactKind) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(k.String())
}

func (k *ArtifactKind) UnmarshalJSON(data []byte) error {
	if k == nil {
		return fmt.Errorf(ErrFmtArtifactKind, core.ErrFuzzContract)
	}
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtArtifactKind, errors.Join(core.ErrFuzzContract, err))
	}
	parsed, err := ParseArtifactKind(token)
	if err != nil {
		return err
	}
	*k = parsed
	return nil
}

func ParseArtifactKind(token string) (ArtifactKind, error) {
	switch token {
	case ArtifactKindNameCorpus:
		return ArtifactKindCorpus, nil
	case ArtifactKindNameCrasher:
		return ArtifactKindCrasher, nil
	default:
		return ArtifactKindUnknown, fmt.Errorf(ErrFmtArtifactKind, core.ErrFuzzContract)
	}
}
