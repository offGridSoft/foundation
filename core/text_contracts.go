package core

import (
	"unicode"
	"unicode/utf8"
)

// ValidateControlFreeUTF8 validates the shared text-safety floor used before
// text enters URLs, headers, terminals, templates, or external output. It does
// not impose ownership-specific rules such as non-empty, trimming, or length;
// the type that owns those rules layers them on top.
func ValidateControlFreeUTF8(value string) error {
	if !utf8.ValidString(value) {
		return ErrFoundationContract
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return ErrFoundationContract
		}
	}
	return nil
}
