package durability

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestAppendRequestHostileValidationTable(t *testing.T) {
	t.Parallel()

	rootText := t.TempDir()
	root, err := core.ParseAbsoluteDirectoryPath(rootText)
	if err != nil {
		t.Fatalf("ParseAbsoluteDirectoryPath() error = %v", err)
	}
	target, err := core.ParseAbsoluteFilePath(filepath.Join(rootText, "events.log"))
	if err != nil {
		t.Fatalf("ParseAbsoluteFilePath() error = %v", err)
	}
	valid := AppendRequest{Root: root, Target: target, DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(1)}
	tests := []struct {
		name    string
		request AppendRequest
		wantErr bool
	}{
		{name: "p01 minimum maximum", request: valid},
		{name: "p02 exact signed maximum", request: AppendRequest{Root: root, Target: target, DirectoryMode: 0o750, FileMode: 0o640, MaximumFileBytes: core.NewByteCount(math.MaxInt64)}},
		{name: "n01 zero request", request: AppendRequest{}, wantErr: true},
		{name: "n02 zero root", request: AppendRequest{Target: target, DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(1)}, wantErr: true},
		{name: "n03 relative target", request: AppendRequest{Root: root, Target: core.AbsoluteFilePath("relative.log"), DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(1)}, wantErr: true},
		{name: "n04 unclean target", request: AppendRequest{Root: root, Target: core.AbsoluteFilePath(rootText + string(filepath.Separator) + "a" + string(filepath.Separator) + ".." + string(filepath.Separator) + "events.log"), DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(1)}, wantErr: true},
		{name: "n05 target outside root", request: AppendRequest{Root: root, Target: core.AbsoluteFilePath(filepath.Join(t.TempDir(), "events.log")), DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(1)}, wantErr: true},
		{name: "n06 zero directory mode", request: AppendRequest{Root: root, Target: target, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(1)}, wantErr: true},
		{name: "n07 directory type mode", request: AppendRequest{Root: root, Target: target, DirectoryMode: fs.ModeDir | 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(1)}, wantErr: true},
		{name: "n08 zero file mode", request: AppendRequest{Root: root, Target: target, DirectoryMode: 0o700, MaximumFileBytes: core.NewByteCount(1)}, wantErr: true},
		{name: "n09 symlink file mode", request: AppendRequest{Root: root, Target: target, DirectoryMode: 0o700, FileMode: fs.ModeSymlink | 0o600, MaximumFileBytes: core.NewByteCount(1)}, wantErr: true},
		{name: "n10 zero maximum", request: AppendRequest{Root: root, Target: target, DirectoryMode: 0o700, FileMode: 0o600}, wantErr: true},
		{name: "n11 over signed maximum", request: AppendRequest{Root: root, Target: target, DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(math.MaxUint64)}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.request.Validate()
			if tc.wantErr && !errors.Is(err, core.ErrDurabilityContract) {
				t.Fatalf("AppendRequest.Validate() error = %v, want ErrDurabilityContract", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("AppendRequest.Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestOpenAppendRealFilesystemBoundaryTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantOpenError error
		wantWriteErr  error
		name          string
		initial       []byte
		write         []byte
		want          []byte
		maximum       uint64
	}{
		{name: "p01 create empty then zero write", maximum: 1, write: nil, want: nil},
		{name: "p02 create and fill exactly", maximum: 4, write: []byte("abcd"), want: []byte("abcd")},
		{name: "p03 preserve existing and fill exactly", initial: []byte("ab"), maximum: 4, write: []byte("cd"), want: []byte("abcd")},
		{name: "p04 existing exactly maximum accepts zero", initial: []byte("abcd"), maximum: 4, write: nil, want: []byte("abcd")},
		{name: "n01 one over remaining rejected without mutation", initial: []byte("ab"), maximum: 4, write: []byte("cde"), wantWriteErr: core.ErrDurableSizeLimit, want: []byte("ab")},
		{name: "n02 one byte at full rejected without mutation", initial: []byte("abcd"), maximum: 4, write: []byte("e"), wantWriteErr: core.ErrDurableSizeLimit, want: []byte("abcd")},
		{name: "n03 existing one over maximum rejected on open", initial: []byte("abcde"), maximum: 4, wantOpenError: core.ErrDurableSizeLimit, want: []byte("abcde")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			root, parseRootErr := core.ParseAbsoluteDirectoryPath(dir)
			if parseRootErr != nil {
				t.Fatalf("ParseAbsoluteDirectoryPath() error = %v", parseRootErr)
			}
			path := filepath.Join(dir, "events.log")
			if tc.initial != nil {
				if err := os.WriteFile(path, tc.initial, 0o600); err != nil {
					t.Fatalf("WriteFile(initial) error = %v", err)
				}
			}
			target, err := core.ParseAbsoluteFilePath(path)
			if err != nil {
				t.Fatalf("ParseAbsoluteFilePath() error = %v", err)
			}
			writer, err := OpenAppend(t.Context(), AppendRequest{Root: root, Target: target, DirectoryMode: 0o700, FileMode: 0o640, MaximumFileBytes: core.NewByteCount(tc.maximum)})
			if tc.wantOpenError != nil {
				if !errors.Is(err, tc.wantOpenError) {
					t.Fatalf("OpenAppend() error = %v, want %v", err, tc.wantOpenError)
				}
				got, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatalf("ReadFile(%s) error = %v", path, readErr)
				}
				if !bytes.Equal(got, tc.want) {
					t.Fatalf("ReadFile(%s) = %q, want %q", path, got, tc.want)
				}
				return
			}
			if err != nil {
				t.Fatalf("OpenAppend() error = %v, want nil", err)
			}
			n, writeErr := writer.Write(tc.write)
			if tc.wantWriteErr != nil {
				if n != 0 || !errors.Is(writeErr, tc.wantWriteErr) {
					t.Fatalf("AppendWriter.Write() = (%d,%v), want (0,%v)", n, writeErr, tc.wantWriteErr)
				}
			} else if n != len(tc.write) || writeErr != nil {
				t.Fatalf("AppendWriter.Write() = (%d,%v), want (%d,nil)", n, writeErr, len(tc.write))
			}
			if err := writer.Close(); err != nil {
				t.Fatalf("AppendWriter.Close() error = %v", err)
			}
			got, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("ReadFile(%s) error = %v", path, readErr)
			}
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("ReadFile(%s) = %q, want %q", path, got, tc.want)
			}
			if tc.initial == nil {
				info, statErr := os.Stat(path)
				if statErr != nil || info.Mode().Perm() != 0o640 {
					t.Fatalf("created mode = %v stat error = %v, want 0640", infoMode(info), statErr)
				}
			}
		})
	}
}

func TestOpenAppendRejectsContextAndNonRegularTargets(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	root, _ := core.ParseAbsoluteDirectoryPath(dir)
	missing, _ := core.ParseAbsoluteFilePath(filepath.Join(dir, "missing.log"))
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	contextTests := []struct {
		ctx     context.Context
		wantErr error
		name    string
	}{
		{name: "nil", ctx: nil, wantErr: core.ErrNilContext},
		{name: "cancelled", ctx: cancelled, wantErr: context.Canceled},
	}
	for _, tc := range contextTests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			writer, err := OpenAppend(tc.ctx, AppendRequest{Root: root, Target: missing, DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(1)})
			if writer != nil || !errors.Is(err, tc.wantErr) {
				t.Fatalf("OpenAppend(%s) = (%v,%v), want (nil,%v)", tc.name, writer, err, tc.wantErr)
			}
			if _, statErr := os.Stat(missing.String()); !errors.Is(statErr, fs.ErrNotExist) {
				t.Fatalf("Stat(missing after rejected context) error = %v, want fs.ErrNotExist", statErr)
			}
		})
	}

	directoryTarget, _ := core.ParseAbsoluteFilePath(dir)
	regular := filepath.Join(dir, "regular.log")
	if err := os.WriteFile(regular, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile(regular) error = %v", err)
	}
	symlinkPath := filepath.Join(dir, "link.log")
	if err := os.Symlink(regular, symlinkPath); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	symlinkTarget, _ := core.ParseAbsoluteFilePath(symlinkPath)
	for _, tc := range []struct {
		name   string
		target core.AbsoluteFilePath
	}{
		{name: "directory", target: directoryTarget},
		{name: "symlink", target: symlinkTarget},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			writer, err := OpenAppend(t.Context(), AppendRequest{Root: root, Target: tc.target, DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(8)})
			if writer != nil || !errors.Is(err, core.ErrDurabilityContract) {
				t.Fatalf("OpenAppend(%s) = (%v,%v), want (nil,ErrDurabilityContract)", tc.name, writer, err)
			}
		})
	}

	escapeRootText := t.TempDir()
	escapeOutside := t.TempDir()
	if err := os.Symlink(escapeOutside, filepath.Join(escapeRootText, "logs")); err != nil {
		t.Fatalf("Symlink(parent escape) error = %v", err)
	}
	escapeRoot, _ := core.ParseAbsoluteDirectoryPath(escapeRootText)
	escapeTarget, _ := core.ParseAbsoluteFilePath(filepath.Join(escapeRootText, "logs", "events.log"))
	writer, err := OpenAppend(t.Context(), AppendRequest{Root: escapeRoot, Target: escapeTarget, DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(8)})
	if writer != nil || !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("OpenAppend(symlink parent) = (%v,%v), want (nil,ErrDurabilityContract)", writer, err)
	}
	if _, statErr := os.Stat(filepath.Join(escapeOutside, "events.log")); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("Stat(outside escaped target) error = %v, want fs.ErrNotExist", statErr)
	}
}

func TestAppendWriterFailureLattice(t *testing.T) {
	t.Parallel()

	errStat := errors.New("stat failure")
	errWrite := errors.New("write failure")
	tests := []struct {
		wantError error
		file      *fakeAppendFile
		name      string
		data      []byte
		maximum   uint64
		wantN     int
	}{
		{name: "p01 empty", file: newFakeAppendFile(0), maximum: 1},
		{name: "p02 exact", file: newFakeAppendFile(1), data: []byte("x"), maximum: 2, wantN: 1},
		{name: "n01 stat", file: &fakeAppendFile{statErr: errStat}, data: []byte("x"), maximum: 2, wantError: errStat},
		{name: "n02 negative size", file: newFakeAppendFile(-1), data: []byte("x"), maximum: 2, wantError: core.ErrDurabilityContract},
		{name: "n03 non regular", file: &fakeAppendFile{info: fakeAppendFileInfo{mode: fs.ModeDir, size: 0}}, data: []byte("x"), maximum: 2, wantError: core.ErrDurabilityContract},
		{name: "n04 full", file: newFakeAppendFile(2), data: []byte("x"), maximum: 2, wantError: core.ErrDurableSizeLimit},
		{name: "n05 short nil", file: &fakeAppendFile{info: fakeAppendFileInfo{mode: 0o600}, writeN: 1}, data: []byte("xx"), maximum: 2, wantN: 1, wantError: core.ErrDurableShortWrite},
		{name: "n06 short error", file: &fakeAppendFile{info: fakeAppendFileInfo{mode: 0o600}, writeN: 1, writeErr: errWrite}, data: []byte("xx"), maximum: 2, wantN: 1, wantError: errWrite},
		{name: "n07 full error", file: &fakeAppendFile{info: fakeAppendFileInfo{mode: 0o600}, writeN: 2, writeErr: errWrite}, data: []byte("xx"), maximum: 2, wantN: 2, wantError: errWrite},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			target, _ := core.ParseAbsoluteFilePath(filepath.Join(t.TempDir(), "fake.log"))
			root, _ := core.ParseAbsoluteDirectoryPath(filepath.Dir(target.String()))
			writer := &AppendWriter{file: tc.file, request: AppendRequest{Root: root, Target: target, DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(tc.maximum)}, state: appendStateOpen}
			n, err := writer.Write(tc.data)
			if n != tc.wantN {
				t.Fatalf("AppendWriter.Write() count = %d, want %d", n, tc.wantN)
			}
			if tc.wantError != nil && !errors.Is(err, tc.wantError) {
				t.Fatalf("AppendWriter.Write() error = %v, want %v", err, tc.wantError)
			}
			if tc.wantError == nil && err != nil {
				t.Fatalf("AppendWriter.Write() error = %v, want nil", err)
			}
		})
	}
}

func TestAppendWriterClosePreservesEveryFailureIdentity(t *testing.T) {
	t.Parallel()

	errSync := errors.New("sync failure")
	errClose := errors.New("close failure")
	target, _ := core.ParseAbsoluteFilePath(filepath.Join(t.TempDir(), "fake.log"))
	root, _ := core.ParseAbsoluteDirectoryPath(filepath.Dir(target.String()))
	file := newFakeAppendFile(0)
	file.syncErr = errSync
	file.closeErr = errClose
	writer := &AppendWriter{file: file, request: AppendRequest{Root: root, Target: target, DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(1)}, state: appendStateOpen}
	if err := writer.Close(); !errors.Is(err, errSync) || !errors.Is(err, errClose) {
		t.Fatalf("AppendWriter.Close() error = %v, want both sync and close identities", err)
	}
	if _, err := writer.Write([]byte("x")); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("AppendWriter.Write(after close) error = %v, want ErrDurabilityContract", err)
	}
	if err := writer.Sync(t.Context()); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("AppendWriter.Sync(after close) error = %v, want ErrDurabilityContract", err)
	}
	if err := writer.Close(); !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("AppendWriter.Close(second) error = %v, want ErrDurabilityContract", err)
	}
}

func TestAppendWriterSerializesConcurrentRecords(t *testing.T) {
	t.Parallel()

	const records = 128
	line := []byte("record\n")
	path := filepath.Join(t.TempDir(), "events.log")
	root, _ := core.ParseAbsoluteDirectoryPath(filepath.Dir(path))
	target, _ := core.ParseAbsoluteFilePath(path)
	writer, err := OpenAppend(t.Context(), AppendRequest{Root: root, Target: target, DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(records * uint64(len(line)))})
	if err != nil {
		t.Fatalf("OpenAppend() error = %v", err)
	}
	var group sync.WaitGroup
	group.Add(records)
	for range records {
		go func() {
			defer group.Done()
			if n, writeErr := writer.Write(line); n != len(line) || writeErr != nil {
				t.Errorf("AppendWriter.Write() = (%d,%v), want (%d,nil)", n, writeErr, len(line))
			}
		}()
	}
	group.Wait()
	if err := writer.Close(); err != nil {
		t.Fatalf("AppendWriter.Close() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got := bytes.Count(data, line); got != records || len(data) != records*len(line) {
		t.Fatalf("concurrent records = %d bytes = %d, want %d records and %d bytes", got, len(data), records, records*len(line))
	}
}

type fakeAppendFile struct {
	statErr  error
	writeErr error
	syncErr  error
	closeErr error
	info     fakeAppendFileInfo
	writeN   int
}

func newFakeAppendFile(size int64) *fakeAppendFile {
	return &fakeAppendFile{info: fakeAppendFileInfo{mode: 0o600, size: size}, writeN: -1}
}

func (f *fakeAppendFile) Write(data []byte) (int, error) {
	if f.writeN >= 0 {
		return f.writeN, f.writeErr
	}
	return len(data), f.writeErr
}

func (f *fakeAppendFile) Stat() (fs.FileInfo, error) { return f.info, f.statErr }
func (f *fakeAppendFile) SyncStable() error          { return f.syncErr }
func (f *fakeAppendFile) Close() error               { return f.closeErr }

type fakeAppendFileInfo struct {
	mode fs.FileMode
	size int64
}

func (i fakeAppendFileInfo) Name() string       { return "append.log" }
func (i fakeAppendFileInfo) Size() int64        { return i.size }
func (i fakeAppendFileInfo) Mode() fs.FileMode  { return i.mode }
func (i fakeAppendFileInfo) ModTime() time.Time { return time.Time{} }
func (i fakeAppendFileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i fakeAppendFileInfo) Sys() any           { return nil }

func infoMode(info fs.FileInfo) fs.FileMode {
	if info == nil {
		return 0
	}
	return info.Mode().Perm()
}

var _ io.Writer = (*AppendWriter)(nil)
