package core

import (
	"strings"
	"unicode"
)

const OpaqueTokenDefaultMaxRunes = 128

func ValidateOpaqueToken(value string, maxRunes int) error {
	if maxRunes < 1 {
		return ErrFoundationContract
	}
	if value == "" || strings.TrimSpace(value) != value {
		return ErrFoundationContract
	}
	if len([]rune(value)) > maxRunes {
		return ErrFoundationContract
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return ErrFoundationContract
		}
	}
	return nil
}
