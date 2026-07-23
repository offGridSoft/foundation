//go:build !windows

package durability

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestOpenAppendRejectsFIFOWithoutWaitingForReader(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "events.pipe")
	root, _ := core.ParseAbsoluteDirectoryPath(filepath.Dir(path))
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("Mkfifo() error = %v", err)
	}
	target, err := core.ParseAbsoluteFilePath(path)
	if err != nil {
		t.Fatalf("ParseAbsoluteFilePath() error = %v", err)
	}
	writer, err := OpenAppend(t.Context(), AppendRequest{Root: root, Target: target, DirectoryMode: 0o700, FileMode: 0o600, MaximumFileBytes: core.NewByteCount(8)})
	if writer != nil || !errors.Is(err, core.ErrDurabilityContract) {
		t.Fatalf("OpenAppend(FIFO) = (%v,%v), want (nil,ErrDurabilityContract) without blocking", writer, err)
	}
	info, statErr := os.Lstat(path)
	if statErr != nil || info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("Lstat(FIFO) = (%v,%v), want original named pipe preserved", info, statErr)
	}
}
