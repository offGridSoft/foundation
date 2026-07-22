package shutdown

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/offGridSoft/foundation/v2026/core"
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

func newStepPanicError(recovered any) StepPanicError {
	return StepPanicError{Value: capturePanicValue(recovered), authentic: true}
}

func newForcePanicError(recovered any) ForcePanicError {
	return ForcePanicError{Value: capturePanicValue(recovered), authentic: true}
}

func capturePanicValue(recovered any) PanicValue {
	typeName := PanicTypeName(boundedPanicString(fmt.Sprintf("%T", recovered), core.ShutdownPanicTypeMaxRunes))
	diagnostic := PanicDiagnostic(strconv.QuoteToASCII(boundedPanicText(recovered)))
	return PanicValue{Type: typeName, Diagnostic: diagnostic}
}

func boundedPanicText(recovered any) (text string) {
	text = "panic diagnostic unavailable"
	defer func() {
		if recover() != nil {
			text = "panic diagnostic unavailable"
		}
	}()
	switch value := recovered.(type) {
	case string:
		return boundedPanicString(value, core.ShutdownPanicSourceMaxRunes)
	case []byte:
		return boundedPanicString(string(value[:min(len(value), core.ShutdownPanicSourceMaxRunes*utf8.UTFMax)]), core.ShutdownPanicSourceMaxRunes)
	case error:
		return boundedPanicString(value.Error(), core.ShutdownPanicSourceMaxRunes)
	case fmt.Stringer:
		return boundedPanicString(value.String(), core.ShutdownPanicSourceMaxRunes)
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64, complex64, complex128:
		return fmt.Sprintf("%v", value)
	default:
		return "panic value omitted; inspect type"
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
