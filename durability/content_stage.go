package durability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/offGridSoft/foundation/v2026/contextcheck"
	"github.com/offGridSoft/foundation/v2026/core"
)

// ContentStageRequest owns a bounded temporary file beneath Root. Directory
// may equal Root; every other path must remain beneath Root and contain no
// symlink component.
type ContentStageRequest struct {
	Root         core.AbsoluteDirectoryPath
	Directory    core.AbsoluteDirectoryPath
	Mode         fs.FileMode
	MaximumBytes core.ByteCount
}

func (r ContentStageRequest) Validate() error {
	if err := r.Root.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	if err := r.Directory.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	if err := validateCreationMode(r.Mode); err != nil {
		return err
	}
	if err := r.MaximumBytes.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	if _, err := r.MaximumBytes.Int64(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	_, err := rootedRelativePath(r.Root.String(), r.Directory.String(), true)
	return err
}

type contentStageState uint8

const (
	contentStageStateUnknown contentStageState = iota
	contentStageStateOpen
	contentStageStateSealed
	contentStageStateActivated
	contentStageStateDurable
	contentStageStateComplete
	contentStageStateAborted
)

type ContentStage struct {
	root         *os.Root
	file         *os.File
	temporary    string
	target       string
	request      ContentStageRequest
	writtenBytes uint64
	result       CommitResult
	state        contentStageState
	rootClosed   bool
}

type contentStageTokenSource interface {
	Fill([]byte) error
}

type cryptoContentStageTokenSource struct{}

func (cryptoContentStageTokenSource) Fill(destination []byte) error {
	_, err := io.ReadFull(rand.Reader, destination)
	return err
}

func NewContentStage(ctx context.Context, request ContentStageRequest) (*ContentStage, error) {
	return newContentStage(ctx, request, cryptoContentStageTokenSource{})
}

func newContentStage(ctx context.Context, request ContentStageRequest, tokens contentStageTokenSource) (*ContentStage, error) {
	if err := contextcheck.Validate(ctx); err != nil {
		return nil, err
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if tokens == nil {
		return nil, core.ErrDurabilityContract
	}
	root, err := openVerifiedDirectoryRoot(request.Root.String())
	if err != nil {
		return nil, err
	}
	directory, _ := rootedRelativePath(request.Root.String(), request.Directory.String(), true)
	if err := requireRootedRealDirectory(ctx, root, directory); err != nil {
		return nil, errors.Join(err, root.Close())
	}
	file, temporary, err := createContentStageFile(root, directory, request.Mode, tokens)
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	return &ContentStage{
		root: root, file: file, request: request, temporary: temporary,
		result: CommitResult{Activation: ActivationNotActivated, Temporary: TemporaryRetained},
		state:  contentStageStateOpen,
	}, nil
}

func createContentStageFile(root *os.Root, directory string, mode fs.FileMode, tokens contentStageTokenSource) (*os.File, string, error) {
	token := make([]byte, core.DurableStageTokenBytes)
	for range core.DurableStageCreateAttempts {
		if err := tokens.Fill(token); err != nil {
			return nil, "", fmt.Errorf("generate durable content stage name: %w", err)
		}
		name := filepath.Join(directory, core.DurableStagePrefix+hex.EncodeToString(token))
		file, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, mode)
		if err == nil {
			return file, name, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, "", fmt.Errorf("create durable content stage: %w", err)
		}
	}
	return nil, "", errors.Join(core.ErrDurabilityContract, fs.ErrExist)
}

func (s *ContentStage) Write(data []byte) (int, error) {
	if s == nil || s.file == nil || s.state != contentStageStateOpen {
		return 0, core.ErrDurabilityContract
	}
	if len(data) == 0 {
		return 0, nil
	}
	maximum := s.request.MaximumBytes.Uint64()
	if s.writtenBytes >= maximum {
		return 0, core.ErrDurableSizeLimit
	}
	allowed := min(uint64(len(data)), maximum-s.writtenBytes)
	count, writeErr := s.file.Write(data[:allowed])
	written, countErr := validatedWriteCount(count, allowed)
	if countErr != nil {
		return 0, countErr
	}
	s.writtenBytes += written
	if writeErr != nil {
		return count, writeErr
	}
	if written != allowed {
		return count, core.ErrDurableShortWrite
	}
	if allowed != uint64(len(data)) {
		return count, core.ErrDurableSizeLimit
	}
	return count, nil
}

// Commit activates the staged bytes at target with create-only semantics.
// A failure after activation returns a resumable result; retrying Commit with
// the same target completes the outstanding directory durability barrier.
func (s *ContentStage) Commit(ctx context.Context, target core.AbsoluteFilePath) (CommitResult, error) {
	if s == nil || s.root == nil || s.file == nil {
		return CommitResult{}, core.ErrDurabilityContract
	}
	if err := contextcheck.Validate(ctx); err != nil {
		return s.result, err
	}
	relative, err := s.validateTarget(ctx, target)
	if err != nil {
		return s.result, err
	}
	if err := s.advanceCommit(ctx, relative); err != nil {
		return s.result, err
	}
	if s.state != contentStageStateComplete {
		return s.result, core.ErrDurabilityContract
	}
	return s.result, errors.Join(s.result.Validate(), s.closeRoot())
}

func (s *ContentStage) advanceCommit(ctx context.Context, relative string) error {
	if err := s.sealForCommit(); err != nil {
		return err
	}
	if err := s.activateForCommit(ctx, relative); err != nil {
		return err
	}
	if err := s.syncActivationForCommit(ctx, relative); err != nil {
		return err
	}
	if s.state == contentStageStateDurable {
		return s.removeTemporary()
	}
	return nil
}

func (s *ContentStage) sealForCommit() error {
	if s.state == contentStageStateOpen {
		return s.seal()
	}
	return nil
}

func (s *ContentStage) activateForCommit(ctx context.Context, relative string) error {
	if s.state == contentStageStateSealed {
		if err := contextcheck.Validate(ctx); err != nil {
			return err
		}
		return s.activate(relative)
	}
	return nil
}

func (s *ContentStage) syncActivationForCommit(ctx context.Context, relative string) error {
	if s.state == contentStageStateActivated {
		if err := contextcheck.Validate(ctx); err != nil {
			return err
		}
		if err := syncRootDirectory(s.root, filepath.Dir(relative)); err != nil {
			return errors.Join(core.ErrDurableActivationIncomplete, err)
		}
		s.result.Activation = ActivationDurable
		s.state = contentStageStateDurable
	}
	return nil
}

func (s *ContentStage) validateTarget(ctx context.Context, target core.AbsoluteFilePath) (string, error) {
	if err := target.Validate(); err != nil {
		return "", errors.Join(core.ErrDurabilityContract, err)
	}
	relative, err := rootedRelativePath(s.request.Root.String(), target.String(), false)
	if err != nil {
		return "", err
	}
	if s.target != "" && s.target != relative {
		return "", core.ErrDurabilityContract
	}
	if s.state == contentStageStateComplete || s.state == contentStageStateAborted {
		return "", core.ErrDurabilityContract
	}
	if err := requireRootedRealDirectory(ctx, s.root, filepath.Dir(relative)); err != nil {
		return "", err
	}
	return relative, nil
}

func (s *ContentStage) seal() error {
	if err := CommitFile(s.file); err != nil {
		return errors.Join(fmt.Errorf("sync durable content stage: %w", err), s.Abort())
	}
	if err := s.file.Close(); err != nil {
		s.state = contentStageStateSealed
		return errors.Join(fmt.Errorf("close durable content stage: %w", err), s.abortSealed())
	}
	s.state = contentStageStateSealed
	return nil
}

func (s *ContentStage) activate(target string) error {
	if err := s.root.Link(s.temporary, target); err != nil {
		return errors.Join(fmt.Errorf("activate durable content stage: %w", err), s.abortSealed())
	}
	s.target = target
	s.result.Activation = ActivationDirectorySyncRequired
	s.state = contentStageStateActivated
	return nil
}

func (s *ContentStage) removeTemporary() error {
	if s.result.Temporary == TemporaryRetained {
		if err := s.root.Remove(s.temporary); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove durable content stage: %w", err)
		}
		s.result.Temporary = TemporaryRemovalSyncRequired
	}
	if s.result.Temporary == TemporaryRemovalSyncRequired {
		if err := syncRootDirectory(s.root, filepath.Dir(s.temporary)); err != nil {
			return errors.Join(core.ErrDurableCleanupIncomplete, err)
		}
		s.result.Temporary = TemporaryRemoved
	}
	s.state = contentStageStateComplete
	return nil
}

func (s *ContentStage) Abort() error {
	if s == nil {
		return nil
	}
	if s.root == nil || s.file == nil {
		return core.ErrDurabilityContract
	}
	switch s.state {
	case contentStageStateOpen:
		closeErr := s.file.Close()
		s.state = contentStageStateSealed
		return errors.Join(closeErr, s.abortSealed())
	case contentStageStateSealed:
		return s.abortSealed()
	case contentStageStateAborted:
		return nil
	default:
		return core.ErrDurabilityContract
	}
}

func (s *ContentStage) abortSealed() error {
	if s.result.Activation != ActivationNotActivated {
		return core.ErrDurabilityContract
	}
	removeErr := s.root.Remove(s.temporary)
	if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
		return removeErr
	}
	s.result.Temporary = TemporaryRemovalSyncRequired
	syncErr := syncRootDirectory(s.root, filepath.Dir(s.temporary))
	if syncErr != nil {
		return syncErr
	}
	s.result.Temporary = TemporaryRemoved
	s.state = contentStageStateAborted
	return s.closeRoot()
}

func (s *ContentStage) closeRoot() error {
	if s.rootClosed {
		return nil
	}
	s.rootClosed = true
	return s.root.Close()
}

func rootedRelativePath(root, target string, allowRoot bool) (string, error) {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || (!allowRoot && relative == ".") {
		return "", errors.Join(core.ErrDurabilityContract, err)
	}
	return relative, nil
}

func requireRootedRealDirectory(ctx context.Context, root *os.Root, relative string) error {
	if relative == "." {
		return nil
	}
	current := "."
	for element := range strings.SplitSeq(relative, string(filepath.Separator)) {
		if err := contextcheck.Validate(ctx); err != nil {
			return err
		}
		current = filepath.Join(current, element)
		info, err := root.Lstat(current)
		if err != nil {
			return fmt.Errorf("lstat rooted durable directory: %w", err)
		}
		if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
			return core.ErrDurabilityContract
		}
	}
	return nil
}
