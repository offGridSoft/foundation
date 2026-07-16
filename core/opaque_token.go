package core

import (
	"strings"
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
	if err := ValidateControlFreeUTF8(value); err != nil {
		return ErrFoundationContract
	}
	if value == "" || strings.TrimSpace(value) != value {
		return ErrFoundationContract
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return ErrFoundationContract
	}
	return nil
}
