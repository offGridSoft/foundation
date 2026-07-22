package durability

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestStageRejectsEveryUseAfterTerminalState(t *testing.T) {
	t.Parallel()

	t.Run("write and abort after durable commit leave the target intact", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		target, err := core.ParseAbsoluteFilePath(filepath.Join(dir, "target"))
		if err != nil {
			t.Fatalf("ParseAbsoluteFilePath() error = %v", err)
		}
		stage, err := NewStage(t.Context(), WriteRequest{Target: target, Mode: 0o600, Install: InstallReplace, MaximumBytes: core.NewByteCount(16)})
		if err != nil {
			t.Fatalf("NewStage() error = %v", err)
		}
		if _, err := stage.Write([]byte("evidence")); err != nil {
			t.Fatalf("Write() error = %v, want nil", err)
		}
		result, err := stage.Commit(t.Context())
		if err != nil || result.Activation != ActivationDurable || result.Temporary != TemporaryRemoved {
			t.Fatalf("Commit() = (%+v,%v), want durable/removed", result, err)
		}
		if _, err := stage.Write([]byte("late")); !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("Write(after commit) error = %v, want ErrDurabilityContract", err)
		}
		if _, err := stage.Write(nil); !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("Write(empty after commit) error = %v, want ErrDurabilityContract", err)
		}
		if err := stage.Abort(); err != nil {
			t.Fatalf("Abort(after commit) error = %v, want nil no-op", err)
		}
		data, err := os.ReadFile(target.String())
		if err != nil || string(data) != "evidence" {
			t.Fatalf("ReadFile(target after post-commit misuse) = (%q,%v), want evidence intact", data, err)
		}
	})

	t.Run("write and commit after abort are rejected typed", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		target, err := core.ParseAbsoluteFilePath(filepath.Join(dir, "target"))
		if err != nil {
			t.Fatalf("ParseAbsoluteFilePath() error = %v", err)
		}
		stage, err := NewStage(t.Context(), WriteRequest{Target: target, Mode: 0o600, Install: InstallCreate, MaximumBytes: core.NewByteCount(16)})
		if err != nil {
			t.Fatalf("NewStage() error = %v", err)
		}
		if _, err := stage.Write([]byte("partial")); err != nil {
			t.Fatalf("Write() error = %v, want nil", err)
		}
		if err := stage.Abort(); err != nil {
			t.Fatalf("Abort() error = %v, want nil", err)
		}
		if _, err := stage.Write([]byte("resurrect")); !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("Write(after abort) error = %v, want ErrDurabilityContract", err)
		}
		if _, err := stage.Commit(t.Context()); !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("Commit(after abort) error = %v, want ErrDurabilityContract", err)
		}
		if _, err := os.Stat(target.String()); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Stat(target after abort) error = %v, want fs.ErrNotExist", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("ReadDir(after abort) = (%d entries,%v), want empty directory", len(entries), err)
		}
	})

	t.Run("nil stage receiver is rejected typed and abort is a safe no-op", func(t *testing.T) {
		t.Parallel()
		var stage *Stage
		if _, err := stage.Write([]byte("x")); !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("nil Stage.Write() error = %v, want ErrDurabilityContract", err)
		}
		if _, err := stage.Commit(t.Context()); !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("nil Stage.Commit() error = %v, want ErrDurabilityContract", err)
		}
		if err := stage.Abort(); err != nil {
			t.Fatalf("nil Stage.Abort() error = %v, want nil", err)
		}
	})
}

func TestStageWriteAccumulatesAcrossCallsToExactLimitAndCommits(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target, err := core.ParseAbsoluteFilePath(filepath.Join(dir, "target"))
	if err != nil {
		t.Fatalf("ParseAbsoluteFilePath() error = %v", err)
	}
	stage, err := NewStage(t.Context(), WriteRequest{Target: target, Mode: 0o600, Install: InstallReplace, MaximumBytes: core.NewByteCount(10)})
	if err != nil {
		t.Fatalf("NewStage() error = %v", err)
	}
	for index, chunk := range []string{"abcd", "efgh"} {
		n, err := stage.Write([]byte(chunk))
		if n != len(chunk) || err != nil {
			t.Fatalf("Write(chunk %d) = (%d,%v), want (%d,nil)", index, n, err, len(chunk))
		}
	}
	n, err := stage.Write([]byte("ijkl"))
	if n != 2 || !errors.Is(err, core.ErrDurableSizeLimit) {
		t.Fatalf("Write(crossing chunk) = (%d,%v), want (2, ErrDurableSizeLimit)", n, err)
	}
	if n, err := stage.Write([]byte("m")); n != 0 || !errors.Is(err, core.ErrDurableSizeLimit) {
		t.Fatalf("Write(at exhausted limit) = (%d,%v), want (0, ErrDurableSizeLimit)", n, err)
	}
	result, err := stage.Commit(t.Context())
	if err != nil || result.Activation != ActivationDurable || result.Temporary != TemporaryRemoved {
		t.Fatalf("Commit(after limit) = (%+v,%v), want durable/removed", result, err)
	}
	data, err := os.ReadFile(target.String())
	if err != nil || string(data) != "abcdefghij" {
		t.Fatalf("ReadFile(target) = (%q,%v), want exactly the 10 allowed bytes", data, err)
	}
}

func TestStageCommitWithDeadContextStaysResumable(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target, err := core.ParseAbsoluteFilePath(filepath.Join(dir, "target"))
	if err != nil {
		t.Fatalf("ParseAbsoluteFilePath() error = %v", err)
	}
	stage, err := NewStage(t.Context(), WriteRequest{Target: target, Mode: 0o600, Install: InstallReplace, MaximumBytes: core.NewByteCount(8)})
	if err != nil {
		t.Fatalf("NewStage() error = %v", err)
	}
	if _, err := stage.Write([]byte("x")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := stage.Commit(cancelled)
	if !errors.Is(err, context.Canceled) || result.Activation != ActivationNotActivated || result.Temporary != TemporaryRetained {
		t.Fatalf("Commit(cancelled) = (%+v,%v), want untouched not-activated state and context.Canceled", result, err)
	}
	var nilContext context.Context
	if _, err := stage.Commit(nilContext); !errors.Is(err, core.ErrNilContext) {
		t.Fatalf("Commit(nil context) error = %v, want ErrNilContext", err)
	}
	resumed, err := stage.Commit(t.Context())
	if err != nil || resumed.Activation != ActivationDurable || resumed.Temporary != TemporaryRemoved {
		t.Fatalf("Commit(resumed) = (%+v,%v), want durable/removed after dead-context attempts", resumed, err)
	}
	data, err := os.ReadFile(target.String())
	if err != nil || string(data) != "x" {
		t.Fatalf("ReadFile(target) = (%q,%v), want staged byte durable", data, err)
	}
}

func TestWriteOutcomeRetainsExactRecoveryCapabilityAfterActivation(t *testing.T) {
	t.Parallel()

	errSync := errors.New("activation directory sync failed")
	target, _ := core.ParseAbsoluteFilePath(filepath.Join(t.TempDir(), "target"))
	file := &fakeStageFile{name: filepath.Join(filepath.Dir(target.String()), ".stage")}
	filesystem := &fakeStageFilesystem{file: file, syncDirectoryErrors: []error{errSync, nil}}
	request := WriteRequest{Target: target, Mode: 0o600, Install: InstallReplace, MaximumBytes: core.NewByteCount(16)}
	outcome, err := write(t.Context(), request, bytes.NewBufferString("evidence"), filesystem)
	if !errors.Is(err, core.ErrDurableActivationIncomplete) || !errors.Is(err, errSync) || outcome.Validate() != nil {
		t.Fatalf("write(directory sync failure) = (%+v,%v) validate=%v, want typed recoverable activation", outcome, err, outcome.Validate())
	}
	if outcome.Result.Activation != ActivationDirectorySyncRequired || outcome.Result.Temporary != TemporaryRemovalSyncRequired || outcome.Recovery == nil {
		t.Fatalf("write recovery state = %+v, want directory-sync/removal-sync with capability", outcome)
	}
	if err := outcome.Recover(t.Context()); err != nil || outcome.Validate() != nil {
		t.Fatalf("WriteOutcome.Recover() = %v state=%+v validate=%v, want durable recovery", err, outcome, outcome.Validate())
	}
	if outcome.Result.Activation != ActivationDurable || outcome.Result.Temporary != TemporaryRemoved || outcome.Recovery != nil {
		t.Fatalf("recovered outcome = %+v, want durable/removed without retained capability", outcome)
	}
	if filesystem.renameCalls != 1 || filesystem.syncDirectoryCalls != 2 || file.syncCalls != 1 || file.closeCalls != 1 {
		t.Fatalf("write/recover calls rename/dir-sync/file-sync/file-close = %d/%d/%d/%d, want 1/2/1/1", filesystem.renameCalls, filesystem.syncDirectoryCalls, file.syncCalls, file.closeCalls)
	}
	if err := outcome.Recover(t.Context()); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("WriteOutcome.Recover(after completion) error = %v, want ErrDurabilityContract", err)
	}
}

func TestWriteOutcomeRetainsCleanupCapabilityWhenStageInitializationPartiallyFails(t *testing.T) {
	t.Parallel()

	errMode := errors.New("chmod failed")
	errSync := errors.New("abort directory sync failed")
	target, _ := core.ParseAbsoluteFilePath(filepath.Join(t.TempDir(), "target"))
	file := &fakeStageFile{name: filepath.Join(filepath.Dir(target.String()), ".stage"), chmodErr: errMode}
	filesystem := &fakeStageFilesystem{file: file, syncDirectoryErrors: []error{errSync, nil}}
	request := WriteRequest{Target: target, Mode: 0o600, Install: InstallReplace, MaximumBytes: core.NewByteCount(16)}
	outcome, err := write(t.Context(), request, bytes.NewBufferString("unreachable"), filesystem)
	if !errors.Is(err, errMode) || !errors.Is(err, core.ErrDurableCleanupIncomplete) || !errors.Is(err, errSync) || outcome.Validate() != nil {
		t.Fatalf("write(partial initialization) = (%+v,%v) validate=%v, want mode and recoverable cleanup identities", outcome, err, outcome.Validate())
	}
	if outcome.Result.Activation != ActivationNotActivated || outcome.Result.Temporary != TemporaryRemovalSyncRequired || outcome.Recovery == nil {
		t.Fatalf("partial initialization outcome = %+v, want not-activated/removal-sync with capability", outcome)
	}
	if err := outcome.Recover(t.Context()); err != nil || outcome.Validate() != nil || outcome.Result.Temporary != TemporaryRemoved || outcome.Recovery != nil {
		t.Fatalf("WriteOutcome.Recover(cleanup) = %v state=%+v validate=%v, want removed terminal cleanup", err, outcome, outcome.Validate())
	}
	if filesystem.removeCalls != 1 || filesystem.syncDirectoryCalls != 2 || file.closeCalls != 1 {
		t.Fatalf("cleanup recovery calls remove/sync/close = %d/%d/%d, want 1/2/1", filesystem.removeCalls, filesystem.syncDirectoryCalls, file.closeCalls)
	}
}

func TestWriteOutcomeRejectsImpossibleRecoveryStateLattice(t *testing.T) {
	t.Parallel()

	stage := &Stage{result: CommitResult{Activation: ActivationDirectorySyncRequired, Temporary: TemporaryRemovalSyncRequired}}
	cases := []WriteOutcome{
		{},
		{Result: CommitResult{Activation: ActivationDurable, Temporary: TemporaryRemoved}, Recovery: stage},
		{Result: CommitResult{Activation: ActivationDirectorySyncRequired, Temporary: TemporaryRemovalSyncRequired}},
		{Result: CommitResult{Activation: ActivationDirectorySyncRequired, Temporary: TemporaryRemovalSyncRequired}, Recovery: &Stage{result: CommitResult{Activation: ActivationDurable, Temporary: TemporaryRemoved}}},
		{Result: CommitResult{Activation: ActivationNotActivated, Temporary: TemporaryRetained}},
		{Result: CommitResult{Activation: ActivationDurable, Temporary: TemporaryRetained}},
	}
	for index, outcome := range cases {
		if err := outcome.Validate(); !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("invalid WriteOutcome %d Validate() error = %v, want ErrDurabilityContract", index, err)
		}
	}
	var nilOutcome *WriteOutcome
	if err := nilOutcome.Recover(t.Context()); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("nil WriteOutcome.Recover() error = %v, want ErrDurabilityContract", err)
	}
	recoverable := WriteOutcome{Result: stage.result, Recovery: stage}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := recoverable.Recover(cancelled); !errors.Is(err, context.Canceled) || recoverable.Result != stage.result || recoverable.Recovery != stage {
		t.Fatalf("WriteOutcome.Recover(cancelled) = %v state=%+v, want unchanged resumable context cancellation", err, recoverable)
	}
}
