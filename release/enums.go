package release

import (
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

type Kind uint8

const (
	KindUnknown Kind = iota
	KindProductBinary
	KindToolBundle
	KindManifest
	KindDownloadIndex
	KindCustodyRecord
)

func kindNames() [KindCustodyRecord + 1]string {
	return [...]string{
		KindProductBinary: kindTokenProductBinary,
		KindToolBundle:    kindTokenToolBundle,
		KindManifest:      kindTokenManifest,
		KindDownloadIndex: kindTokenDownloadIndex,
		KindCustodyRecord: kindTokenCustodyRecord,
	}
}

const (
	kindTokenProductBinary = "product_binary"
	kindTokenToolBundle    = "tool_bundle"
	kindTokenManifest      = "manifest"
	kindTokenDownloadIndex = "download_index"
	kindTokenCustodyRecord = "custody_record"
)

func (k Kind) String() string {
	if k.IsValid() {
		return kindNames()[k]
	}
	return ""
}

func (k Kind) IsValid() bool {
	return k > KindUnknown && int(k) < len(kindNames()) && kindNames()[k] != ""
}

func (k Kind) Validate() error {
	if !k.IsValid() {
		return fmt.Errorf(ErrFmtKind, core.ErrReleaseContract)
	}
	return nil
}

func (k Kind) RequiresPlatform() bool {
	return k == KindProductBinary || k == KindToolBundle
}

func (k Kind) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(k.String())
}

func ParseKind(token string) (Kind, error) {
	for kind := KindProductBinary; int(kind) < len(kindNames()); kind++ {
		if kindNames()[kind] == token {
			return kind, nil
		}
	}
	return KindUnknown, fmt.Errorf(ErrFmtKind, core.ErrReleaseContract)
}

func (k *Kind) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtKind, core.ErrReleaseContract)
	}
	parsed, err := ParseKind(token)
	if err != nil {
		return err
	}
	*k = parsed
	return nil
}

type Visibility uint8

const (
	VisibilityUnknown Visibility = iota
	VisibilityPublic
	VisibilityPrivate
)

func visibilityNames() [VisibilityPrivate + 1]string {
	return [...]string{
		VisibilityPublic:  ObjectSegmentPublic,
		VisibilityPrivate: ObjectSegmentPrivate,
	}
}

func (v Visibility) String() string {
	if v.IsValid() {
		return visibilityNames()[v]
	}
	return ""
}

func (v Visibility) IsValid() bool {
	return v > VisibilityUnknown && int(v) < len(visibilityNames()) && visibilityNames()[v] != ""
}

func (v Visibility) Validate() error {
	if !v.IsValid() {
		return fmt.Errorf(ErrFmtVisibility, core.ErrReleaseContract)
	}
	return nil
}

func (v Visibility) MarshalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(v.String())
}

func ParseVisibility(token string) (Visibility, error) {
	for visibility := VisibilityPublic; int(visibility) < len(visibilityNames()); visibility++ {
		if visibilityNames()[visibility] == token {
			return visibility, nil
		}
	}
	return VisibilityUnknown, fmt.Errorf(ErrFmtVisibility, core.ErrReleaseContract)
}

func (v *Visibility) UnmarshalJSON(data []byte) error {
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf(ErrFmtVisibility, core.ErrReleaseContract)
	}
	parsed, err := ParseVisibility(token)
	if err != nil {
		return err
	}
	*v = parsed
	return nil
}
