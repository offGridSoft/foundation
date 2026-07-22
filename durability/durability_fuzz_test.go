package durability

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func FuzzDurabilityEnumJSONNeverMutatesOnRejection(f *testing.F) {
	for _, seed := range [][]byte{[]byte(`"replace"`), []byte(`"durable"`), []byte(`"removed"`), []byte(`null`), []byte{}, []byte(`{}`)} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		install := InstallCreate
		installErr := install.UnmarshalJSON(data)
		if installErr != nil && (install != InstallCreate || !errors.Is(installErr, core.ErrDurabilityContract)) {
			t.Fatalf("InstallMode.UnmarshalJSON(%q) = (%d,%v), want unchanged on typed rejection", data, install, installErr)
		}
		if installErr == nil {
			raw, err := json.Marshal(install)
			if err != nil || !json.Valid(raw) || install.Validate() != nil {
				t.Fatalf("accepted InstallMode = %d raw:%q errors:%v/%v", install, raw, err, install.Validate())
			}
		}

		activation := ActivationDurable
		activationErr := activation.UnmarshalJSON(data)
		if activationErr != nil && (activation != ActivationDurable || !errors.Is(activationErr, core.ErrDurabilityContract)) {
			t.Fatalf("ActivationState.UnmarshalJSON(%q) = (%d,%v), want unchanged on typed rejection", data, activation, activationErr)
		}
		if activationErr == nil && activation.Validate() != nil {
			t.Fatalf("accepted ActivationState(%d).Validate() error = %v", activation, activation.Validate())
		}

		temporary := TemporaryRemoved
		temporaryErr := temporary.UnmarshalJSON(data)
		if temporaryErr != nil && (temporary != TemporaryRemoved || !errors.Is(temporaryErr, core.ErrDurabilityContract)) {
			t.Fatalf("TemporaryState.UnmarshalJSON(%q) = (%d,%v), want unchanged on typed rejection", data, temporary, temporaryErr)
		}
		if temporaryErr == nil && temporary.Validate() != nil {
			t.Fatalf("accepted TemporaryState(%d).Validate() error = %v", temporary, temporary.Validate())
		}
	})
}

func FuzzStageWriteBoundaries(f *testing.F) {
	f.Add(uint16(1), []byte{})
	f.Add(uint16(1), []byte("x"))
	f.Add(uint16(2), []byte("abc"))
	f.Add(uint16(1024), []byte("evidence"))
	f.Fuzz(func(t *testing.T, rawMaximum uint16, data []byte) {
		maximum := uint64(rawMaximum) + 1
		target, err := core.ParseAbsoluteFilePath(filepath.Join(t.TempDir(), "target"))
		if err != nil {
			t.Fatalf("ParseAbsoluteFilePath(target) error = %v", err)
		}
		file := &fakeStageFile{name: filepath.Join(filepath.Dir(target.String()), ".stage")}
		stage, err := newStage(t.Context(), WriteRequest{Target: target, Mode: 0o600, Install: InstallReplace, MaximumBytes: core.NewByteCount(maximum)}, &fakeStageFilesystem{file: file})
		if err != nil {
			t.Fatalf("newStage(maximum=%d) error = %v", maximum, err)
		}
		n, writeErr := stage.Write(data)
		want := min(uint64(len(data)), maximum)
		if n < 0 || uint64(n) != want || stage.written != want {
			t.Fatalf("Write(maximum=%d,len=%d) = n:%d written:%d, want %d", maximum, len(data), n, stage.written, want)
		}
		if uint64(len(data)) > maximum && !errors.Is(writeErr, core.ErrDurableSizeLimit) {
			t.Fatalf("Write(maximum=%d,len=%d) error = %v, want ErrDurableSizeLimit", maximum, len(data), writeErr)
		}
		if uint64(len(data)) <= maximum && writeErr != nil {
			t.Fatalf("Write(maximum=%d,len=%d) error = %v, want nil", maximum, len(data), writeErr)
		}
	})
}
