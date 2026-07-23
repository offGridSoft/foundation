package durability

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

const contentStageTestMaximum = 32

func TestContentStageRealFilesystemLifecycleHostileTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		exercise func(*testing.T, ContentStageRequest, core.AbsoluteFilePath)
		name     string
	}{
		{name: "exact ceiling across multiple writes commits byte-identical create-only file", exercise: exerciseContentStageExactCeiling},
		{name: "zero-byte content commits an honest empty object", exercise: exerciseContentStageZeroBytes},
		{name: "one byte beyond ceiling commits only the accepted prefix", exercise: exerciseContentStageOverflow},
		{name: "existing immutable target survives and stage residue is reclaimed", exercise: exerciseContentStageExistingTarget},
		{name: "cancelled commit leaves the open stage resumable", exercise: exerciseContentStageCancelledCommit},
		{name: "relative target rejection leaves the stage resumable", exercise: exerciseContentStageRelativeTarget},
		{name: "outside-root target rejection leaves the stage resumable", exercise: exerciseContentStageOutsideTarget},
		{name: "abort is idempotent and rejects every later operation", exercise: exerciseContentStageAbort},
		{name: "nested real target directory commits without leaking stage files", exercise: exerciseContentStageNestedTarget},
		{name: "zero-length writes preserve the exact byte budget", exercise: exerciseContentStageZeroWrites},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rootText := t.TempDir()
			stageText := filepath.Join(rootText, "stage")
			targetDirectory := filepath.Join(rootText, "objects")
			for _, directory := range []string{stageText, targetDirectory, filepath.Join(targetDirectory, "nested")} {
				if err := os.Mkdir(directory, 0o700); err != nil {
					t.Fatalf("Mkdir(%q) error = %v, want nil", directory, err)
				}
			}
			request := contentStageRequest(rootText, stageText)
			testCase.exercise(t, request, core.AbsoluteFilePath(filepath.Join(targetDirectory, "object")))
		})
	}
}

func exerciseContentStageExactCeiling(t *testing.T, request ContentStageRequest, target core.AbsoluteFilePath) {
	t.Helper()
	stage := mustContentStage(t, request)
	chunks := [][]byte{bytes.Repeat([]byte{'a'}, contentStageTestMaximum-1), {'b'}}
	for _, chunk := range chunks {
		count, err := stage.Write(chunk)
		if err != nil || count != len(chunk) {
			t.Fatalf("Write(%d) = (%d,%v), want (%d,nil)", len(chunk), count, err, len(chunk))
		}
	}
	result, err := stage.Commit(t.Context(), target)
	requireContentCommit(t, result, err)
	requireContentBytes(t, target.String(), append(chunks[0], chunks[1]...))
}

func exerciseContentStageZeroBytes(t *testing.T, request ContentStageRequest, target core.AbsoluteFilePath) {
	t.Helper()
	stage := mustContentStage(t, request)
	result, err := stage.Commit(t.Context(), target)
	requireContentCommit(t, result, err)
	requireContentBytes(t, target.String(), nil)
}

func exerciseContentStageOverflow(t *testing.T, request ContentStageRequest, target core.AbsoluteFilePath) {
	t.Helper()
	stage := mustContentStage(t, request)
	input := bytes.Repeat([]byte{'z'}, contentStageTestMaximum+1)
	count, err := stage.Write(input)
	if count != contentStageTestMaximum || !errors.Is(err, core.ErrDurableSizeLimit) {
		t.Fatalf("Write(one over) = (%d,%v), want (%d,ErrDurableSizeLimit)", count, err, contentStageTestMaximum)
	}
	if count, err := stage.Write([]byte{'x'}); count != 0 || !errors.Is(err, core.ErrDurableSizeLimit) {
		t.Fatalf("Write(after ceiling) = (%d,%v), want (0,ErrDurableSizeLimit)", count, err)
	}
	result, commitErr := stage.Commit(t.Context(), target)
	requireContentCommit(t, result, commitErr)
	requireContentBytes(t, target.String(), input[:contentStageTestMaximum])
}

func exerciseContentStageExistingTarget(t *testing.T, request ContentStageRequest, target core.AbsoluteFilePath) {
	t.Helper()
	want := []byte("immutable")
	if err := os.WriteFile(target.String(), want, 0o600); err != nil {
		t.Fatalf("WriteFile(existing) error = %v, want nil", err)
	}
	stage := mustContentStage(t, request)
	if _, err := stage.Write([]byte("replacement")); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}
	result, err := stage.Commit(t.Context(), target)
	if !errors.Is(err, fs.ErrExist) || result.Activation != ActivationNotActivated || result.Temporary != TemporaryRemoved {
		t.Fatalf("Commit(existing) = (%+v,%v), want not-activated/removed/fs.ErrExist", result, err)
	}
	requireContentBytes(t, target.String(), want)
	requireDirectoryEmpty(t, request.Directory.String())
}

func exerciseContentStageCancelledCommit(t *testing.T, request ContentStageRequest, target core.AbsoluteFilePath) {
	t.Helper()
	stage := mustContentStage(t, request)
	if _, err := stage.Write([]byte("resumable")); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	result, err := stage.Commit(cancelled, target)
	if !errors.Is(err, context.Canceled) || result.Activation != ActivationNotActivated || result.Temporary != TemporaryRetained {
		t.Fatalf("Commit(cancelled) = (%+v,%v), want retained/context.Canceled", result, err)
	}
	result, err = stage.Commit(t.Context(), target)
	requireContentCommit(t, result, err)
	requireContentBytes(t, target.String(), []byte("resumable"))
}

func exerciseContentStageRelativeTarget(t *testing.T, request ContentStageRequest, target core.AbsoluteFilePath) {
	t.Helper()
	exerciseContentStageRejectedTargetThenResume(t, request, "relative", target)
}

func exerciseContentStageOutsideTarget(t *testing.T, request ContentStageRequest, target core.AbsoluteFilePath) {
	t.Helper()
	exerciseContentStageRejectedTargetThenResume(t, request, core.AbsoluteFilePath(filepath.Join(filepath.Dir(request.Root.String()), "outside")), target)
}

func exerciseContentStageRejectedTargetThenResume(t *testing.T, request ContentStageRequest, rejected, target core.AbsoluteFilePath) {
	t.Helper()
	stage := mustContentStage(t, request)
	if _, err := stage.Write([]byte("resumable")); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}
	result, err := stage.Commit(t.Context(), rejected)
	if !errors.Is(err, core.ErrDurabilityContract) || result.Activation != ActivationNotActivated || result.Temporary != TemporaryRetained {
		t.Fatalf("Commit(rejected target) = (%+v,%v), want retained/ErrDurabilityContract", result, err)
	}
	result, err = stage.Commit(t.Context(), target)
	requireContentCommit(t, result, err)
}

func exerciseContentStageAbort(t *testing.T, request ContentStageRequest, target core.AbsoluteFilePath) {
	t.Helper()
	stage := mustContentStage(t, request)
	if _, err := stage.Write([]byte("partial")); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}
	if err := stage.Abort(); err != nil {
		t.Fatalf("Abort() error = %v, want nil", err)
	}
	if err := stage.Abort(); err != nil {
		t.Fatalf("Abort(second) error = %v, want nil", err)
	}
	if count, err := stage.Write([]byte{'x'}); count != 0 || !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("Write(after abort) = (%d,%v), want typed rejection", count, err)
	}
	if result, err := stage.Commit(t.Context(), target); result.Activation != ActivationNotActivated || result.Temporary != TemporaryRemoved || !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("Commit(after abort) = (%+v,%v), want not-activated/removed/typed rejection", result, err)
	}
	requireDirectoryEmpty(t, request.Directory.String())
}

func exerciseContentStageNestedTarget(t *testing.T, request ContentStageRequest, target core.AbsoluteFilePath) {
	t.Helper()
	nested := core.AbsoluteFilePath(filepath.Join(filepath.Dir(target.String()), "nested", "object"))
	stage := mustContentStage(t, request)
	if _, err := stage.Write([]byte("nested")); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}
	result, err := stage.Commit(t.Context(), nested)
	requireContentCommit(t, result, err)
	requireContentBytes(t, nested.String(), []byte("nested"))
}

func exerciseContentStageZeroWrites(t *testing.T, request ContentStageRequest, target core.AbsoluteFilePath) {
	t.Helper()
	stage := mustContentStage(t, request)
	for range 100 {
		if count, err := stage.Write(nil); count != 0 || err != nil {
			t.Fatalf("Write(nil) = (%d,%v), want (0,nil)", count, err)
		}
	}
	input := bytes.Repeat([]byte{'q'}, contentStageTestMaximum)
	if count, err := stage.Write(input); count != len(input) || err != nil {
		t.Fatalf("Write(exact after zero writes) = (%d,%v), want (%d,nil)", count, err, len(input))
	}
	result, err := stage.Commit(t.Context(), target)
	requireContentCommit(t, result, err)
}

func TestContentStageIngressAndCreationFailureHostileTable(t *testing.T) {
	t.Parallel()

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	tokenFailure := errors.New("content stage token test failure")
	tests := []struct {
		wantError error
		build     func(*testing.T) (context.Context, ContentStageRequest, contentStageTokenSource)
		name      string
	}{
		{name: "nil context is rejected", wantError: core.ErrNilContext, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			return nil, contentStageRequest(t.TempDir(), t.TempDir()), cryptoContentStageTokenSource{}
		}},
		{name: "cancelled context is rejected", wantError: context.Canceled, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			root := t.TempDir()
			return cancelled, contentStageRequest(root, root), cryptoContentStageTokenSource{}
		}},
		{name: "zero root is rejected", wantError: core.ErrDurabilityContract, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			return t.Context(), ContentStageRequest{Directory: core.AbsoluteDirectoryPath(t.TempDir()), Mode: 0o600, MaximumBytes: core.NewByteCount(1)}, cryptoContentStageTokenSource{}
		}},
		{name: "relative root is rejected", wantError: core.ErrDurabilityContract, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			return t.Context(), ContentStageRequest{Root: "relative", Directory: core.AbsoluteDirectoryPath(t.TempDir()), Mode: 0o600, MaximumBytes: core.NewByteCount(1)}, cryptoContentStageTokenSource{}
		}},
		{name: "zero directory is rejected", wantError: core.ErrDurabilityContract, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			return t.Context(), ContentStageRequest{Root: core.AbsoluteDirectoryPath(t.TempDir()), Mode: 0o600, MaximumBytes: core.NewByteCount(1)}, cryptoContentStageTokenSource{}
		}},
		{name: "directory outside root is rejected", wantError: core.ErrDurabilityContract, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			return t.Context(), contentStageRequest(t.TempDir(), t.TempDir()), cryptoContentStageTokenSource{}
		}},
		{name: "missing root is rejected", wantError: fs.ErrNotExist, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			root := filepath.Join(t.TempDir(), "missing")
			return t.Context(), contentStageRequest(root, root), cryptoContentStageTokenSource{}
		}},
		{name: "missing staging directory is rejected", wantError: fs.ErrNotExist, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			root := t.TempDir()
			return t.Context(), contentStageRequest(root, filepath.Join(root, "missing")), cryptoContentStageTokenSource{}
		}},
		{name: "zero mode is rejected", wantError: core.ErrDurabilityContract, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			root := t.TempDir()
			request := contentStageRequest(root, root)
			request.Mode = 0
			return t.Context(), request, cryptoContentStageTokenSource{}
		}},
		{name: "mode type bits are rejected", wantError: core.ErrDurabilityContract, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			root := t.TempDir()
			request := contentStageRequest(root, root)
			request.Mode = fs.ModeDir | 0o700
			return t.Context(), request, cryptoContentStageTokenSource{}
		}},
		{name: "zero maximum is rejected", wantError: core.ErrDurabilityContract, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			root := t.TempDir()
			request := contentStageRequest(root, root)
			request.MaximumBytes = core.ByteCount{}
			return t.Context(), request, cryptoContentStageTokenSource{}
		}},
		{name: "maximum above signed file range is rejected", wantError: core.ErrDurabilityContract, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			root := t.TempDir()
			request := contentStageRequest(root, root)
			request.MaximumBytes = core.NewByteCount(uint64(math.MaxInt64) + 1)
			return t.Context(), request, cryptoContentStageTokenSource{}
		}},
		{name: "nil token source is rejected", wantError: core.ErrDurabilityContract, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			root := t.TempDir()
			return t.Context(), contentStageRequest(root, root), nil
		}},
		{name: "token entropy failure is retained", wantError: tokenFailure, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			root := t.TempDir()
			return t.Context(), contentStageRequest(root, root), failingContentStageTokens{err: tokenFailure}
		}},
		{name: "all bounded create attempts collide loudly", wantError: fs.ErrExist, build: func(t *testing.T) (context.Context, ContentStageRequest, contentStageTokenSource) {
			root := t.TempDir()
			name := filepath.Join(root, core.DurableStagePrefix+string(bytes.Repeat([]byte{'0'}, core.DurableStageTokenBytes*2)))
			if err := os.WriteFile(name, nil, 0o600); err != nil {
				t.Fatalf("WriteFile(collision) error = %v, want nil", err)
			}
			return t.Context(), contentStageRequest(root, root), zeroContentStageTokens{}
		}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx, request, tokens := testCase.build(t)
			stage, err := newContentStage(ctx, request, tokens)
			if stage != nil || !errors.Is(err, testCase.wantError) {
				t.Fatalf("newContentStage() = (%v,%v), want nil/errors.Is %v", stage, err, testCase.wantError)
			}
		})
	}
}

func TestContentStageNilReceiverContract(t *testing.T) {
	t.Parallel()

	var stage *ContentStage
	if count, err := stage.Write([]byte{'x'}); count != 0 || !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("nil Write() = (%d,%v), want typed rejection", count, err)
	}
	if err := stage.Abort(); err != nil {
		t.Fatalf("nil Abort() error = %v, want nil", err)
	}
	if result, err := stage.Commit(t.Context(), core.AbsoluteFilePath(filepath.Join(t.TempDir(), "object"))); result != (CommitResult{}) || !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("nil Commit() = (%+v,%v), want zero/typed rejection", result, err)
	}
}

type zeroContentStageTokens struct{}

func (zeroContentStageTokens) Fill(destination []byte) error {
	clear(destination)
	return nil
}

type failingContentStageTokens struct {
	err error
}

func (s failingContentStageTokens) Fill([]byte) error {
	return s.err
}

func contentStageRequest(root, directory string) ContentStageRequest {
	return ContentStageRequest{
		Root: core.AbsoluteDirectoryPath(root), Directory: core.AbsoluteDirectoryPath(directory),
		Mode: 0o600, MaximumBytes: core.NewByteCount(contentStageTestMaximum),
	}
}

func mustContentStage(t testing.TB, request ContentStageRequest) *ContentStage {
	t.Helper()
	stage, err := NewContentStage(t.Context(), request)
	if err != nil {
		t.Fatalf("NewContentStage() error = %v, want nil", err)
	}
	return stage
}

func requireContentCommit(t testing.TB, result CommitResult, err error) {
	t.Helper()
	if err != nil || result.Validate() != nil || result.Activation != ActivationDurable || result.Temporary != TemporaryRemoved {
		t.Fatalf("ContentStage.Commit() = (%+v,%v), validate=%v; want durable/removed/nil", result, err, result.Validate())
	}
}

func requireContentBytes(t testing.TB, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("ReadFile(%q) = (%q,%v), want (%q,nil)", path, got, err, want)
	}
}

func requireDirectoryEmpty(t testing.TB, path string) {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) != 0 {
		t.Fatalf("ReadDir(%q) = (%v,%v), want empty/nil", path, entries, err)
	}
}
