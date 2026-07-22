package shutdown

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	panicDiagnosticUnavailable  = "panic diagnostic unavailable"
	panicValueOmittedDiagnostic = "panic value omitted; inspect type"
)

type PanicTypeName string

func ParsePanicTypeName(value string) (PanicTypeName, error) {
	parsed := PanicTypeName(value)
	if err := parsed.Validate(); err != nil {
		return "", err
	}
	return parsed, nil
}

func (n PanicTypeName) Validate() error {
	if n == "" || utf8.RuneCountInString(string(n)) > core.ShutdownPanicTypeMaxRunes {
		return core.ErrShutdownContract
	}
	if err := core.ValidateControlFreeUTF8(string(n)); err != nil {
		return errors.Join(core.ErrShutdownContract, err)
	}
	return nil
}

func (n PanicTypeName) String() string { return string(n) }

type PanicDiagnostic string

func ParsePanicDiagnostic(value string) (PanicDiagnostic, error) {
	parsed := PanicDiagnostic(value)
	if err := parsed.Validate(); err != nil {
		return "", err
	}
	return parsed, nil
}

func (d PanicDiagnostic) Validate() error {
	if d == "" || utf8.RuneCountInString(string(d)) > core.ShutdownPanicDiagnosticMaxRunes {
		return core.ErrShutdownContract
	}
	if err := core.ValidateControlFreeUTF8(string(d)); err != nil {
		return errors.Join(core.ErrShutdownContract, err)
	}
	decoded, err := strconv.Unquote(string(d))
	if err != nil || utf8.RuneCountInString(decoded) > core.ShutdownPanicSourceMaxRunes || strconv.QuoteToASCII(decoded) != string(d) {
		return core.ErrShutdownContract
	}
	return nil
}

func (d PanicDiagnostic) String() string { return string(d) }

type PanicValue struct {
	Type       PanicTypeName
	Diagnostic PanicDiagnostic
}

func (v PanicValue) Validate() error {
	if err := v.Type.Validate(); err != nil {
		return err
	}
	return v.Diagnostic.Validate()
}

type StepPanicError struct {
	Value     PanicValue
	authentic bool
}

func (e StepPanicError) Validate() error {
	if !e.authentic {
		return core.ErrShutdownContract
	}
	return e.Value.Validate()
}

func (e StepPanicError) Error() string {
	return fmt.Sprintf("shutdown step recovered panic type %s: %s", e.Value.Type, e.Value.Diagnostic)
}

func (e StepPanicError) Unwrap() error { return core.ErrShutdownStepPanic }

type ForcePanicError struct {
	Value     PanicValue
	authentic bool
}

func (e ForcePanicError) Validate() error {
	if !e.authentic {
		return core.ErrShutdownContract
	}
	return e.Value.Validate()
}

func (e ForcePanicError) Error() string {
	return fmt.Sprintf("shutdown force action recovered panic type %s: %s", e.Value.Type, e.Value.Diagnostic)
}

func (e ForcePanicError) Unwrap() error { return core.ErrShutdownForcePanic }

func newStepPanicError(value PanicValue) StepPanicError {
	return StepPanicError{Value: value, authentic: true}
}

func newForcePanicError(value PanicValue) ForcePanicError {
	return ForcePanicError{Value: value, authentic: true}
}

type panicAction func() error
type panicFailure func(PanicValue) error

func runCapturingPanic(action panicAction, result chan<- error, failure panicFailure) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}
		typeName := fmt.Sprintf("%T", recovered)
		text := panicDiagnosticUnavailable
		func() {
			defer func() { _ = recover() }()
			switch value := recovered.(type) {
			case string:
				text = value
			case []byte:
				text = string(value[:min(len(value), core.ShutdownPanicSourceMaxRunes*utf8.UTFMax)])
			case error:
				text = value.Error()
			case fmt.Stringer:
				text = value.String()
			case bool, int, int8, int16, int32, int64,
				uint, uint8, uint16, uint32, uint64, uintptr,
				float32, float64, complex64, complex128:
				text = fmt.Sprintf("%v", value)
			default:
				text = panicValueOmittedDiagnostic
			}
		}()
		result <- failure(newPanicValue(typeName, text))
	}()
	result <- action()
}

func newPanicValue(typeName, text string) PanicValue {
	parsedType := PanicTypeName(boundedPanicString(typeName, core.ShutdownPanicTypeMaxRunes))
	return PanicValue{Type: parsedType, Diagnostic: boundedPanicDiagnostic(text)}
}

// boundedPanicDiagnostic bounds the quoted form, not just the source: ASCII
// quoting expands hostile runes up to ten to one, so a source-only bound can
// exceed the diagnostic ceiling and invalidate an authentic capture.
func boundedPanicDiagnostic(text string) PanicDiagnostic {
	source := boundedPanicString(text, core.ShutdownPanicSourceMaxRunes)
	for {
		quoted := strconv.QuoteToASCII(source)
		if utf8.RuneCountInString(quoted) <= core.ShutdownPanicDiagnosticMaxRunes {
			return PanicDiagnostic(quoted)
		}
		_, size := utf8.DecodeLastRuneInString(source)
		source = source[:len(source)-size]
	}
}

func boundedPanicString(value string, maximum int) string {
	var output strings.Builder
	output.Grow(min(len(value), maximum*utf8.UTFMax))
	count := 0
	for _, character := range value {
		if count == maximum {
			break
		}
		output.WriteRune(character)
		count++
	}
	return output.String()
}
