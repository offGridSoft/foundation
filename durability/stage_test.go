package durability

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestStageCommitFailureMatrixPreservesExactCrashState(t *testing.T) {
	t.Parallel()

	errCreate := errors.New("create failure")
	errChmod := errors.New("chmod failure")
	errSync := errors.New("file sync failure")
	errClose := errors.New("close failure")
	errRename := errors.New("rename failure")
	errLink := errors.New("link failure")
	errRemove := errors.New("remove failure")
	errDirSync := errors.New("directory sync failure")
	cases := []struct {
		name            string
		install         InstallMode
		file            *fakeStageFile
		filesystem      *fakeStageFilesystem
		wantNewError    error
		wantCommitError error
		wantActivation  ActivationState
		wantTemporary   TemporaryState
		wantRename      int
		wantLink        int
		wantRemove      int
		wantDirSync     int
	}{
		{name: "p01_replace_durable", install: InstallReplace, file: &fakeStageFile{}, filesystem: &fakeStageFilesystem{}, wantActivation: ActivationDurable, wantTemporary: TemporaryRemoved, wantRename: 1, wantDirSync: 1},
		{name: "p02_create_durable", install: InstallCreate, file: &fakeStageFile{}, filesystem: &fakeStageFilesystem{}, wantActivation: ActivationDurable, wantTemporary: TemporaryRemoved, wantLink: 1, wantRemove: 1, wantDirSync: 1},
		{name: "n01_create_temp", install: InstallReplace, file: &fakeStageFile{}, filesystem: &fakeStageFilesystem{createErr: errCreate}, wantNewError: errCreate},
		{name: "n02_chmod", install: InstallReplace, file: &fakeStageFile{chmodErr: errChmod}, filesystem: &fakeStageFilesystem{}, wantNewError: errChmod, wantRemove: 1, wantDirSync: 1},
		{name: "n03_file_sync", install: InstallReplace, file: &fakeStageFile{syncErr: errSync}, filesystem: &fakeStageFilesystem{}, wantCommitError: errSync, wantActivation: ActivationNotActivated, wantTemporary: TemporaryRemoved, wantRemove: 1, wantDirSync: 1},
		{name: "n04_file_close", install: InstallReplace, file: &fakeStageFile{closeErr: errClose}, filesystem: &fakeStageFilesystem{}, wantCommitError: errClose, wantActivation: ActivationNotActivated, wantTemporary: TemporaryRemoved, wantRemove: 1, wantDirSync: 1},
		{name: "n05_rename", install: InstallReplace, file: &fakeStageFile{}, filesystem: &fakeStageFilesystem{renameErr: errRename}, wantCommitError: errRename, wantActivation: ActivationNotActivated, wantTemporary: TemporaryRemoved, wantRename: 1, wantRemove: 1, wantDirSync: 1},
		{name: "n06_link", install: InstallCreate, file: &fakeStageFile{}, filesystem: &fakeStageFilesystem{linkErr: errLink}, wantCommitError: errLink, wantActivation: ActivationNotActivated, wantTemporary: TemporaryRemoved, wantLink: 1, wantRemove: 1, wantDirSync: 1},
		{name: "n07_cleanup_after_link", install: InstallCreate, file: &fakeStageFile{}, filesystem: &fakeStageFilesystem{removeErr: errRemove}, wantCommitError: errRemove, wantActivation: ActivationDurable, wantTemporary: TemporaryRetained, wantLink: 1, wantRemove: 1, wantDirSync: 1},
		{name: "n08_parent_sync_after_replace", install: InstallReplace, file: &fakeStageFile{}, filesystem: &fakeStageFilesystem{syncDirectoryErrors: []error{errDirSync}}, wantCommitError: core.ErrDurableActivationIncomplete, wantActivation: ActivationDirectorySyncRequired, wantTemporary: TemporaryRemovalSyncRequired, wantRename: 1, wantDirSync: 1},
		{name: "n09_parent_sync_after_create", install: InstallCreate, file: &fakeStageFile{}, filesystem: &fakeStageFilesystem{syncDirectoryErrors: []error{errDirSync}}, wantCommitError: core.ErrDurableActivationIncomplete, wantActivation: ActivationDirectorySyncRequired, wantTemporary: TemporaryRemovalSyncRequired, wantLink: 1, wantRemove: 1, wantDirSync: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			target, err := core.ParseAbsoluteFilePath(filepath.Join(t.TempDir(), "target"))
			if err != nil {
				t.Fatalf("ParseAbsoluteFilePath() error = %v", err)
			}
			tc.file.name = filepath.Join(filepath.Dir(target.String()), ".stage")
			tc.filesystem.file = tc.file
			stage, err := newStage(t.Context(), WriteRequest{Target: target, Mode: 0o600, Install: tc.install, MaximumBytes: core.NewByteCount(1024)}, tc.filesystem)
			if tc.wantNewError != nil {
				if !errors.Is(err, tc.wantNewError) || tc.filesystem.removeCalls != tc.wantRemove || tc.filesystem.syncDirectoryCalls != tc.wantDirSync {
					t.Fatalf("newStage() = (%v,%v), remove/sync calls %d/%d; want error %v and calls %d/%d", stage, err, tc.filesystem.removeCalls, tc.filesystem.syncDirectoryCalls, tc.wantNewError, tc.wantRemove, tc.wantDirSync)
				}
				return
			}
			if err != nil {
				t.Fatalf("newStage() error = %v, want nil", err)
			}
			result, err := stage.Commit(t.Context())
			if tc.wantCommitError == nil && err != nil {
				t.Fatalf("Commit() error = %v, want nil", err)
			}
			if tc.wantCommitError != nil && !errors.Is(err, tc.wantCommitError) {
				t.Fatalf("Commit() error = %v, want errors.Is %v", err, tc.wantCommitError)
			}
			if result.Activation != tc.wantActivation || result.Temporary != tc.wantTemporary || result.Validate() != nil {
				t.Fatalf("Commit() result = %+v, want activation %d temporary %d and valid", result, tc.wantActivation, tc.wantTemporary)
			}
			if tc.filesystem.renameCalls != tc.wantRename || tc.filesystem.linkCalls != tc.wantLink || tc.filesystem.removeCalls != tc.wantRemove || tc.filesystem.syncDirectoryCalls != tc.wantDirSync {
				t.Fatalf("operation calls rename/link/remove/dirsync = %d/%d/%d/%d, want %d/%d/%d/%d", tc.filesystem.renameCalls, tc.filesystem.linkCalls, tc.filesystem.removeCalls, tc.filesystem.syncDirectoryCalls, tc.wantRename, tc.wantLink, tc.wantRemove, tc.wantDirSync)
			}
		})
	}
}

func TestAbortRetriesDirectorySyncWithoutRepeatingRemoval(t *testing.T) {
	t.Parallel()

	errSync := errors.New("abort directory sync failure")
	target, _ := core.ParseAbsoluteFilePath(filepath.Join(t.TempDir(), "target"))
	file := &fakeStageFile{name: filepath.Join(filepath.Dir(target.String()), ".stage")}
	filesystem := &fakeStageFilesystem{file: file, syncDirectoryErrors: []error{errSync, nil}}
	stage, err := newStage(t.Context(), WriteRequest{Target: target, Mode: 0o600, Install: InstallReplace, MaximumBytes: core.NewByteCount(10)}, filesystem)
	if err != nil {
		t.Fatalf("newStage() error = %v", err)
	}
	firstErr := stage.Abort()
	if !errors.Is(firstErr, core.ErrDurableCleanupIncomplete) || !errors.Is(firstErr, errSync) || stage.result.Temporary != TemporaryRemovalSyncRequired {
		t.Fatalf("first Abort() = %v state=%s, want cleanup-incomplete/sync identity and removal-sync-required", firstErr, stage.result.Temporary)
	}
	if err := stage.Abort(); err != nil || stage.result.Temporary != TemporaryRemoved {
		t.Fatalf("second Abort() = %v state=%s, want removed", err, stage.result.Temporary)
	}
	if err := stage.Abort(); err != nil {
		t.Fatalf("idempotent Abort() error = %v", err)
	}
	if filesystem.removeCalls != 1 || filesystem.syncDirectoryCalls != 2 || file.closeCalls != 1 {
		t.Fatalf("Abort() calls remove/sync/close = %d/%d/%d, want 1/2/1", filesystem.removeCalls, filesystem.syncDirectoryCalls, file.closeCalls)
	}
}

func TestStageRetriesRetainedTemporaryRemovalAndItsDirectorySync(t *testing.T) {
	t.Parallel()

	errRemove := errors.New("first removal failed")
	errSync := errors.New("cleanup directory sync failed")
	target, _ := core.ParseAbsoluteFilePath(filepath.Join(t.TempDir(), "target"))
	file := &fakeStageFile{name: filepath.Join(filepath.Dir(target.String()), ".stage")}
	filesystem := &fakeStageFilesystem{
		file:                file,
		removeErrors:        []error{errRemove, nil},
		syncDirectoryErrors: []error{nil, errSync, nil},
	}
	stage, err := newStage(t.Context(), WriteRequest{Target: target, Mode: 0o600, Install: InstallCreate, MaximumBytes: core.NewByteCount(10)}, filesystem)
	if err != nil {
		t.Fatalf("newStage() error = %v", err)
	}
	first, err := stage.Commit(t.Context())
	if !errors.Is(err, errRemove) || first.Activation != ActivationDurable || first.Temporary != TemporaryRetained {
		t.Fatalf("first Commit() = (%+v,%v), want durable target with retained temporary and removal identity", first, err)
	}
	second, err := stage.Commit(t.Context())
	if !errors.Is(err, core.ErrDurableCleanupIncomplete) || second.Activation != ActivationDurable || second.Temporary != TemporaryRemovalSyncRequired {
		t.Fatalf("second Commit() = (%+v,%v), want durable target with cleanup sync required", second, err)
	}
	third, err := stage.Commit(t.Context())
	if err != nil || third.Activation != ActivationDurable || third.Temporary != TemporaryRemoved {
		t.Fatalf("third Commit() = (%+v,%v), want fully durable and cleaned", third, err)
	}
	if filesystem.linkCalls != 1 || filesystem.removeCalls != 2 || filesystem.syncDirectoryCalls != 3 {
		t.Fatalf("calls link/remove/sync = %d/%d/%d, want 1/2/3 with no activation replay", filesystem.linkCalls, filesystem.removeCalls, filesystem.syncDirectoryCalls)
	}
}

func TestStageRetriesOnlyDirectorySyncAfterActivation(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("first directory sync failed")
	target, _ := core.ParseAbsoluteFilePath(filepath.Join(t.TempDir(), "target"))
	file := &fakeStageFile{name: filepath.Join(filepath.Dir(target.String()), ".stage")}
	filesystem := &fakeStageFilesystem{file: file, syncDirectoryErrors: []error{sentinel, nil}}
	stage, err := newStage(t.Context(), WriteRequest{Target: target, Mode: 0o600, Install: InstallReplace, MaximumBytes: core.NewByteCount(10)}, filesystem)
	if err != nil {
		t.Fatalf("newStage() error = %v", err)
	}
	first, err := stage.Commit(t.Context())
	if !errors.Is(err, core.ErrDurableActivationIncomplete) || first.Activation != ActivationDirectorySyncRequired {
		t.Fatalf("first Commit() = (%+v,%v), want directory-sync-required typed failure", first, err)
	}
	second, err := stage.Commit(t.Context())
	if err != nil || second.Activation != ActivationDurable || filesystem.renameCalls != 1 || file.syncCalls != 1 || file.closeCalls != 1 || filesystem.syncDirectoryCalls != 2 {
		t.Fatalf("second Commit() = (%+v,%v), calls rename/file-sync/file-close/dir-sync=%d/%d/%d/%d; want durable and 1/1/1/2", second, err, filesystem.renameCalls, file.syncCalls, file.closeCalls, filesystem.syncDirectoryCalls)
	}
	third, err := stage.Commit(t.Context())
	if err != nil || third != second || filesystem.syncDirectoryCalls != 2 {
		t.Fatalf("idempotent Commit() = (%+v,%v), dir-sync calls %d; want unchanged durable result and no syscall", third, err, filesystem.syncDirectoryCalls)
	}
	if err := stage.Abort(); err != nil || filesystem.removeCalls != 0 {
		t.Fatalf("Abort(after activation) = %v, remove calls %d; want nil/0", err, filesystem.removeCalls)
	}
}

func TestStageWriteHostileCountAndLimitTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		file        *fakeStageFile
		maximum     uint64
		data        []byte
		wantN       int
		wantWritten uint64
		wantError   error
	}{
		{name: "p01_empty_at_zero_usage", file: &fakeStageFile{}, maximum: 1, data: []byte{}, wantN: 0},
		{name: "p02_below_limit", file: &fakeStageFile{}, maximum: 10, data: []byte("abc"), wantN: 3, wantWritten: 3},
		{name: "p03_exact_limit", file: &fakeStageFile{}, maximum: 3, data: []byte("abc"), wantN: 3, wantWritten: 3},
		{name: "n01_cross_limit", file: &fakeStageFile{}, maximum: 2, data: []byte("abc"), wantN: 2, wantWritten: 2, wantError: core.ErrDurableSizeLimit},
		{name: "n02_underlying_error", file: &fakeStageFile{writeConfigured: true, writeN: 1, writeErr: fs.ErrInvalid}, maximum: 10, data: []byte("abc"), wantN: 1, wantWritten: 1, wantError: fs.ErrInvalid},
		{name: "n03_short_nil", file: &fakeStageFile{writeConfigured: true, writeN: 1}, maximum: 10, data: []byte("abc"), wantN: 1, wantWritten: 1, wantError: core.ErrDurableShortWrite},
		{name: "n04_negative_count", file: &fakeStageFile{writeConfigured: true, writeN: -1}, maximum: 10, data: []byte("abc"), wantError: core.ErrDurableShortWrite},
		{name: "n05_oversized_count", file: &fakeStageFile{writeConfigured: true, writeN: 99}, maximum: 10, data: []byte("abc"), wantError: core.ErrDurableShortWrite},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			target, _ := core.ParseAbsoluteFilePath(filepath.Join(t.TempDir(), "target"))
			tc.file.name = filepath.Join(filepath.Dir(target.String()), ".stage")
			filesystem := &fakeStageFilesystem{file: tc.file}
			stage, err := newStage(t.Context(), WriteRequest{Target: target, Mode: 0o600, Install: InstallReplace, MaximumBytes: core.NewByteCount(tc.maximum)}, filesystem)
			if err != nil {
				t.Fatalf("newStage() error = %v", err)
			}
			n, err := stage.Write(tc.data)
			if n != tc.wantN || stage.written != tc.wantWritten || (tc.wantError == nil && err != nil) || (tc.wantError != nil && !errors.Is(err, tc.wantError)) {
				t.Fatalf("Write() = (%d,%v), written=%d; want (%d,errors.Is %v), written=%d", n, err, stage.written, tc.wantN, tc.wantError, tc.wantWritten)
			}
		})
	}
}

func TestWriteRealFilesystemReplaceCreateLimitAndCancellation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target, _ := core.ParseAbsoluteFilePath(filepath.Join(dir, "evidence"))
	request := WriteRequest{Target: target, Mode: 0o640, Install: InstallReplace, MaximumBytes: core.NewByteCount(8)}
	result, err := Write(t.Context(), request, bytes.NewBufferString("new-data"))
	if err != nil || result.Activation != ActivationDurable || result.Temporary != TemporaryRemoved {
		t.Fatalf("Write(real replace) = (%+v,%v), want durable/removed", result, err)
	}
	data, err := os.ReadFile(target.String())
	if err != nil || string(data) != "new-data" {
		t.Fatalf("ReadFile(target) = (%q,%v), want new-data", data, err)
	}
	info, err := os.Stat(target.String())
	if err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("Stat(target) = (%v,%v), want mode 0640", info, err)
	}

	create := request
	create.Install = InstallCreate
	if _, err := Write(t.Context(), create, bytes.NewBufferString("replace?")); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("Write(create existing) error = %v, want fs.ErrExist", err)
	}
	data, _ = os.ReadFile(target.String())
	if string(data) != "new-data" {
		t.Fatalf("create-only changed existing bytes to %q", data)
	}

	over := request
	over.Target, _ = core.ParseAbsoluteFilePath(filepath.Join(dir, "over"))
	over.MaximumBytes = core.NewByteCount(3)
	if _, err := Write(t.Context(), over, bytes.NewBufferString("four")); !errors.Is(err, core.ErrDurableSizeLimit) {
		t.Fatalf("Write(over limit) error = %v, want ErrDurableSizeLimit", err)
	}
	if _, err := os.Stat(over.Target.String()); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("over-limit target stat error = %v, want fs.ErrNotExist", err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Write(cancelled, request, bytes.NewBufferString("x")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Write(cancelled) error = %v, want context.Canceled", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	for _, entry := range entries {
		if len(entry.Name()) >= len(".foundation-stage-") && entry.Name()[:len(".foundation-stage-")] == ".foundation-stage-" {
			t.Fatalf("orphan stage remained after failure: %s", entry.Name())
		}
	}
}

func TestWriteContractsRejectUnknownEnumsModesPathsAndSizes(t *testing.T) {
	t.Parallel()

	validPath, _ := core.ParseAbsoluteFilePath(filepath.Join(t.TempDir(), "target"))
	cases := []WriteRequest{
		{},
		{Target: "relative", Mode: 0o600, Install: InstallReplace, MaximumBytes: core.NewByteCount(1)},
		{Target: validPath, Install: InstallReplace, MaximumBytes: core.NewByteCount(1)},
		{Target: validPath, Mode: fs.ModeDir | 0o700, Install: InstallReplace, MaximumBytes: core.NewByteCount(1)},
		{Target: validPath, Mode: 0o600, Install: InstallUnknown, MaximumBytes: core.NewByteCount(1)},
		{Target: validPath, Mode: 0o600, Install: 3, MaximumBytes: core.NewByteCount(1)},
		{Target: validPath, Mode: 0o600, Install: InstallReplace},
		{Target: validPath, Mode: 0o600, Install: InstallReplace, MaximumBytes: core.NewByteCount(^uint64(0))},
	}
	for index, request := range cases {
		if err := request.Validate(); !errors.Is(err, core.ErrDurabilityContract) {
			t.Fatalf("case %d WriteRequest.Validate() error = %v, want ErrDurabilityContract", index, err)
		}
	}
	for _, state := range []ActivationState{ActivationUnknown, 4, 127, 128, 255} {
		if !errors.Is(state.Validate(), core.ErrDurabilityContract) {
			t.Fatalf("ActivationState(%d).Validate() error = %v, want ErrDurabilityContract", state, state.Validate())
		}
	}
	for _, state := range []TemporaryState{TemporaryUnknown, 4, 127, 128, 255} {
		if !errors.Is(state.Validate(), core.ErrDurabilityContract) {
			t.Fatalf("TemporaryState(%d).Validate() error = %v, want ErrDurabilityContract", state, state.Validate())
		}
	}
	if _, err := Write(t.Context(), WriteRequest{}, nil); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("Write(nil source) error = %v, want ErrDurabilityContract", err)
	}
}

type fakeStageFile struct {
	name            string
	chmodErr        error
	syncErr         error
	closeErr        error
	writeErr        error
	writeN          int
	writeConfigured bool
	syncCalls       int
	closeCalls      int
}

func (f *fakeStageFile) Write(data []byte) (int, error) {
	if f.writeConfigured {
		return f.writeN, f.writeErr
	}
	return len(data), f.writeErr
}
func (f *fakeStageFile) Chmod(fs.FileMode) error { return f.chmodErr }
func (f *fakeStageFile) Close() error {
	f.closeCalls++
	return f.closeErr
}
func (f *fakeStageFile) SyncStable() error {
	f.syncCalls++
	return f.syncErr
}
func (f *fakeStageFile) Name() string { return f.name }

type fakeStageFilesystem struct {
	file                stageFile
	createErr           error
	renameErr           error
	linkErr             error
	removeErr           error
	removeErrors        []error
	syncDirectoryErrors []error
	renameCalls         int
	linkCalls           int
	removeCalls         int
	syncDirectoryCalls  int
}

func (f *fakeStageFilesystem) CreateTemp(string, string) (stageFile, error) {
	return f.file, f.createErr
}
func (f *fakeStageFilesystem) Link(string, string) error {
	f.linkCalls++
	return f.linkErr
}
func (f *fakeStageFilesystem) Remove(string) error {
	index := f.removeCalls
	f.removeCalls++
	if index < len(f.removeErrors) {
		return f.removeErrors[index]
	}
	return f.removeErr
}
func (f *fakeStageFilesystem) Rename(string, string) error {
	f.renameCalls++
	return f.renameErr
}
func (f *fakeStageFilesystem) SyncParent(core.AbsoluteDirectoryPath) error {
	index := f.syncDirectoryCalls
	f.syncDirectoryCalls++
	if index < len(f.syncDirectoryErrors) {
		return f.syncDirectoryErrors[index]
	}
	return nil
}
