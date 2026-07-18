package workloadidentity

import (
	"errors"
	"fmt"
	"strings"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	TokenMaxBytes                   = 16 << 10
	PrincipalMaxBytes               = 254
	GoogleServiceAccountEmailSuffix = ".iam.gserviceaccount.com"
	ErrFmtToken                     = "workloadidentity.Token: %w"
	ErrFmtPrincipal                 = "workloadidentity.Principal: %w"
)

var ErrContract = fmt.Errorf("workload identity contract violation: %w", core.ErrFoundationContract)

type Token struct{ value string }
type Principal struct{ value string }

func ParsePrincipal(value string) (Principal, error) {
	local, domain, ok := strings.Cut(value, "@")
	if !ok || local == "" || domain == "" || len(value) > PrincipalMaxBytes || !validAccountPart(local) || !validDomain(domain) {
		return Principal{}, fmt.Errorf(ErrFmtPrincipal, ErrContract)
	}
	return Principal{value: value}, nil
}

func (p Principal) String() string  { return p.value }
func (p Principal) IsZero() bool    { return p.value == "" }
func (p Principal) Validate() error { _, err := ParsePrincipal(p.value); return err }

func ParseToken(value string) (Token, error) {
	if len(value) == 0 || len(value) > TokenMaxBytes || !validJWT(value) {
		return Token{}, fmt.Errorf(ErrFmtToken, ErrContract)
	}
	return Token{value: value}, nil
}

func ParseAuthorization(value string) (Token, error) {
	assertion, ok := strings.CutPrefix(value, core.HTTPAuthorizationBearerPrefix)
	if !ok || strings.Contains(assertion, " ") {
		return Token{}, fmt.Errorf(ErrFmtToken, ErrContract)
	}
	return ParseToken(assertion)
}

func (t Token) Validate() error { _, err := ParseToken(t.value); return err }

func (t Token) BearerValue() (string, error) {
	if err := t.Validate(); err != nil {
		return "", err
	}
	return core.HTTPAuthorizationBearerPrefix + t.value, nil
}

func (t Token) Assertion() (string, error) {
	if err := t.Validate(); err != nil {
		return "", err
	}
	return t.value, nil
}

func validAccountPart(value string) bool {
	for index := range len(value) {
		if !lowerAlphaNumericHyphen(value[index]) {
			return false
		}
	}
	return true
}

func validDomain(value string) bool {
	project, ok := strings.CutSuffix(value, GoogleServiceAccountEmailSuffix)
	return ok && project != "" && validAccountPart(project)
}

func lowerAlphaNumericHyphen(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= '0' && value <= '9' || value == '-'
}

func validJWT(value string) bool {
	segments, segmentBytes := 0, 0
	for index := range len(value) {
		if value[index] == '.' {
			if segmentBytes == 0 || segments == 2 {
				return false
			}
			segments++
			segmentBytes = 0
			continue
		}
		if !jwtByte(value[index]) {
			return false
		}
		segmentBytes++
	}
	return segments == 2 && segmentBytes > 0
}

func jwtByte(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '-' || value == '_'
}

func wrap(format string, err error) error { return fmt.Errorf(format, errors.Join(ErrContract, err)) }

var _ core.Validatable = Principal{}
var _ core.Validatable = Token{}
