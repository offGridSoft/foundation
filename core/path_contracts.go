package core

import (
	"fmt"
	"strings"
)

const (
	FileNameTokenMaxRunes = 256
	PathTokenMaxRunes     = 1024
)

func ValidateFileNameToken(value string, maxRunes int) error {
	if err := ValidateOpaqueToken(value, maxRunes); err != nil {
		return fmt.Errorf(ErrFmtFileNameToken, ErrFoundationContract)
	}
	if strings.ContainsAny(value, `/\`) || value == "." || value == ".." {
		return fmt.Errorf(ErrFmtFileNameToken, ErrFoundationContract)
	}
	return nil
}

func ValidatePathToken(value string, maxRunes int) error {
	if err := ValidateOpaqueToken(value, maxRunes); err != nil {
		return fmt.Errorf(ErrFmtPathToken, ErrFoundationContract)
	}
	if strings.HasPrefix(value, "/") || strings.Contains(value, `\`) {
		return fmt.Errorf(ErrFmtPathToken, ErrFoundationContract)
	}
	for segment := range strings.SplitSeq(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf(ErrFmtPathToken, ErrFoundationContract)
		}
	}
	return nil
}
