package core

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	OpaqueTokenDefaultMaxRunes = 128
	RandomIdentityEntropyBytes = 32
)

func ValidateOpaqueToken(value string, maxRunes int) error {
	if maxRunes < 1 {
		return ErrFoundationContract
	}
	if !utf8.ValidString(value) {
		return ErrFoundationContract
	}
	if value == "" || strings.TrimSpace(value) != value {
		return ErrFoundationContract
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return ErrFoundationContract
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return ErrFoundationContract
		}
	}
	return nil
}
