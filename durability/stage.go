package durability

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/offGridSoft/foundation/v2026/contextcheck"
	"github.com/offGridSoft/foundation/v2026/core"
)

type InstallMode uint8

const (
	InstallUnknown InstallMode = iota
	InstallReplace
	InstallCreate
)

func (m InstallMode) Validate() error {
	if !m.IsValid() {
		return core.ErrDurabilityContract
	}
	return nil
}

type WriteRequest struct {
	Target       core.AbsoluteFilePath
	Mode         fs.FileMode
	Install      InstallMode
	MaximumBytes core.ByteCount
}

func (r WriteRequest) Validate() error {
	if err := r.Target.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	if err := validateCreationMode(r.Mode); err != nil {
		return err
	}
	if err := r.Install.Validate(); err != nil {
		return err
	}
	if err := r.MaximumBytes.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	if _, err := r.MaximumBytes.Int64(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	return nil
}

type ActivationState uint8

const (
	ActivationUnknown ActivationState = iota
	ActivationNotActivated
	ActivationDirectorySyncRequired
	ActivationDurable
)

func (s ActivationState) Validate() error {
	if !s.IsValid() {
		return core.ErrDurabilityContract
	}
	return nil
}

type TemporaryState uint8

const (
	TemporaryUnknown TemporaryState = iota
	TemporaryRetained
	TemporaryRemovalSyncRequired
	TemporaryRemoved
)

func (s TemporaryState) Validate() error {
	if !s.IsValid() {
		return core.ErrDurabilityContract
	}
	return nil
}

type CommitResult struct {
	Activation ActivationState
	Temporary  TemporaryState
}

func (r CommitResult) Validate() error {
	if err := r.Activation.Validate(); err != nil {
		return err
	}
	return r.Temporary.Validate()
}

type WriteOutcome struct {
	Recovery *Stage
	Result   CommitResult
}

func (o WriteOutcome) Validate() error {
	if err := o.Result.Validate(); err != nil {
		return err
	}
	recoveryRequired := writeRecoveryRequired(o.Result)
	if recoveryRequired != (o.Recovery != nil) {
		return core.ErrDurabilityContract
	}
	if o.Recovery != nil && o.Recovery.result != o.Result {
		return core.ErrDurabilityContract
	}
	return nil
}

func (o *WriteOutcome) Recover(ctx context.Context) error {
	if o == nil || o.Recovery == nil || o.Validate() != nil {
		return core.ErrDurabilityContract
	}
	if err := contextcheck.Validate(ctx); err != nil {
		return err
	}
	var err error
	if o.Result.Activation == ActivationNotActivated {
		err = o.Recovery.Abort()
		o.Result = o.Recovery.result
	} else {
		o.Result, err = o.Recovery.Commit(ctx)
	}
	if !writeRecoveryRequired(o.Result) {
		o.Recovery = nil
	}
	return err
}

func writeRecoveryRequired(result CommitResult) bool {
	if result.Activation == ActivationDirectorySyncRequired {
		return true
	}
	if result.Activation == ActivationDurable {
		return result.Temporary != TemporaryRemoved
	}
	return result.Activation == ActivationNotActivated && result.Temporary != TemporaryRemoved
}

type stageFile interface {
	io.Writer
	Chmod(fs.FileMode) error
	Close() error
	SyncStable() error
	Name() string
}

type stageFilesystem interface {
	CreateTemp(string, string) (stageFile, error)
	Link(string, string) error
	Remove(string) error
	Rename(string, string) error
	SyncParent(core.AbsoluteDirectoryPath) error
}

type operatingSystem struct{}

type operatingFile struct {
	*os.File
}

func (f operatingFile) SyncStable() error {
	return CommitFile(f.File)
}

func (operatingSystem) CreateTemp(directory, pattern string) (stageFile, error) {
	// witness:waiver doctrine/code_form/defer_after_acquire -- the open file's ownership transfers into Stage; every pre-transfer failure closes it and Stage.Commit or Stage.Abort closes it exactly once afterward.
	file, err := os.CreateTemp(directory, pattern)
	if err != nil {
		return nil, err
	}
	return operatingFile{File: file}, nil
}
func (operatingSystem) Link(oldPath, newPath string) error { return os.Link(oldPath, newPath) }
func (operatingSystem) Remove(path string) error           { return os.Remove(path) }
func (operatingSystem) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}
func (operatingSystem) SyncParent(path core.AbsoluteDirectoryPath) error {
	return CommitDirectory(path)
}

type Stage struct {
	file       stageFile
	filesystem stageFilesystem
	request    WriteRequest
	written    uint64
	result     CommitResult
	open       bool
}

func NewStage(ctx context.Context, request WriteRequest) (*Stage, error) {
	return newStage(ctx, request, operatingSystem{})
}

func newStage(ctx context.Context, request WriteRequest, filesystem stageFilesystem) (*Stage, error) {
	if err := contextcheck.Validate(ctx); err != nil {
		return nil, err
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if filesystem == nil {
		return nil, core.ErrDurabilityContract
	}
	directory, err := core.ParseAbsoluteDirectoryPath(filepath.Dir(request.Target.String()))
	if err != nil {
		return nil, err
	}
	file, err := filesystem.CreateTemp(directory.String(), core.DurableStagePattern)
	if err != nil {
		return nil, fmt.Errorf("create durable stage: %w", err)
	}
	stage := &Stage{
		file:       file,
		filesystem: filesystem,
		request:    request,
		result: CommitResult{
			Activation: ActivationNotActivated,
			Temporary:  TemporaryRetained,
		},
		open: true,
	}
	if err := file.Chmod(request.Mode); err != nil {
		return stage, errors.Join(fmt.Errorf("set durable stage mode: %w", err), stage.Abort())
	}
	return stage, nil
}

func (s *Stage) Write(data []byte) (int, error) {
	length := uint64(len(data))
	allowed, allowanceErr := s.writeAllowance(length)
	if allowanceErr != nil || allowed == 0 {
		return 0, allowanceErr
	}
	n, err := s.file.Write(data[:allowed])
	written, countErr := validatedWriteCount(n, allowed)
	if countErr != nil {
		return 0, countErr
	}
	s.written += written
	if err != nil {
		return n, err
	}
	if written != allowed {
		return n, core.ErrDurableShortWrite
	}
	if allowed != length {
		return n, core.ErrDurableSizeLimit
	}
	return n, nil
}

func (s *Stage) writeAllowance(length uint64) (uint64, error) {
	if s == nil || !s.open || s.file == nil || s.result.Activation != ActivationNotActivated {
		return 0, core.ErrDurabilityContract
	}
	if length == 0 {
		return 0, nil
	}
	maximum := s.request.MaximumBytes.Uint64()
	if s.written >= maximum {
		return 0, core.ErrDurableSizeLimit
	}
	return min(length, maximum-s.written), nil
}

func validatedWriteCount(count int, allowed uint64) (uint64, error) {
	if count < 0 {
		return 0, core.ErrDurableShortWrite
	}
	// #nosec G115 -- the explicit non-negative check makes this widening conversion exact on every supported integer width.
	written := uint64(count)
	if written > allowed {
		return 0, core.ErrDurableShortWrite
	}
	return written, nil
}

func (s *Stage) Commit(ctx context.Context) (CommitResult, error) {
	if s == nil || s.file == nil {
		return CommitResult{}, core.ErrDurabilityContract
	}
	if err := contextcheck.Validate(ctx); err != nil {
		return s.result, err
	}
	if s.result.Activation == ActivationDurable {
		return s.finishTemporaryCleanup()
	}
	if s.result.Activation == ActivationDirectorySyncRequired {
		return s.syncActivatedDirectory()
	}
	if !s.open || s.result.Activation != ActivationNotActivated {
		return s.result, core.ErrDurabilityContract
	}
	return s.commitOpen()
}

func (s *Stage) commitOpen() (CommitResult, error) {
	if err := s.file.SyncStable(); err != nil {
		return s.failBeforeActivation("sync durable stage", err)
	}
	if err := s.file.Close(); err != nil {
		s.open = false
		return s.failBeforeActivation("close durable stage", err)
	}
	s.open = false
	if err := s.activate(); err != nil {
		if s.result.Activation == ActivationNotActivated {
			return s.failBeforeActivation("activate durable stage", err)
		}
		result, syncErr := s.syncActivatedDirectory()
		return result, errors.Join(fmt.Errorf("clean activated durable stage: %w", err), syncErr)
	}
	return s.syncActivatedDirectory()
}

func (s *Stage) activate() error {
	temporary := s.file.Name()
	target := s.request.Target.String()
	if s.request.Install == InstallReplace {
		if err := s.filesystem.Rename(temporary, target); err != nil {
			return err
		}
		s.result.Temporary = TemporaryRemovalSyncRequired
		s.result.Activation = ActivationDirectorySyncRequired
		return nil
	}
	if err := s.filesystem.Link(temporary, target); err != nil {
		return err
	}
	s.result.Activation = ActivationDirectorySyncRequired
	if err := s.filesystem.Remove(temporary); err != nil {
		return fmt.Errorf("remove activated durable stage: %w", err)
	}
	s.result.Temporary = TemporaryRemovalSyncRequired
	return nil
}

func (s *Stage) syncActivatedDirectory() (CommitResult, error) {
	directory, _ := core.ParseAbsoluteDirectoryPath(filepath.Dir(s.request.Target.String()))
	if err := s.filesystem.SyncParent(directory); err != nil {
		return s.result, errors.Join(core.ErrDurableActivationIncomplete, fmt.Errorf("sync activated durable directory: %w", err))
	}
	s.result.Activation = ActivationDurable
	if s.result.Temporary == TemporaryRemovalSyncRequired {
		s.result.Temporary = TemporaryRemoved
	}
	return s.result, s.result.Validate()
}

func (s *Stage) finishTemporaryCleanup() (CommitResult, error) {
	if s.result.Temporary == TemporaryRemoved {
		return s.result, s.result.Validate()
	}
	if s.result.Temporary == TemporaryRetained {
		err := s.filesystem.Remove(s.file.Name())
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return s.result, fmt.Errorf("retry durable temporary cleanup: %w", err)
		}
		s.result.Temporary = TemporaryRemovalSyncRequired
	}
	directory, _ := core.ParseAbsoluteDirectoryPath(filepath.Dir(s.request.Target.String()))
	if err := s.filesystem.SyncParent(directory); err != nil {
		return s.result, errors.Join(core.ErrDurableCleanupIncomplete, fmt.Errorf("sync durable temporary cleanup: %w", err))
	}
	s.result.Temporary = TemporaryRemoved
	return s.result, s.result.Validate()
}

func (s *Stage) failBeforeActivation(operation string, cause error) (CommitResult, error) {
	abortErr := s.Abort()
	return s.result, errors.Join(fmt.Errorf("%s: %w", operation, cause), abortErr)
}

func (s *Stage) Abort() error {
	if s == nil || s.file == nil || s.result.Activation != ActivationNotActivated {
		return nil
	}
	var closeErr error
	if s.open {
		closeErr = s.file.Close()
		s.open = false
	}
	if s.result.Temporary == TemporaryRetained {
		removeErr := s.filesystem.Remove(s.file.Name())
		if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			return errors.Join(closeErr, removeErr)
		}
		s.result.Temporary = TemporaryRemovalSyncRequired
	}
	if s.result.Temporary == TemporaryRemovalSyncRequired {
		directory, _ := core.ParseAbsoluteDirectoryPath(filepath.Dir(s.request.Target.String()))
		if err := s.filesystem.SyncParent(directory); err != nil {
			return errors.Join(closeErr, core.ErrDurableCleanupIncomplete, fmt.Errorf("sync aborted durable stage removal: %w", err))
		}
		s.result.Temporary = TemporaryRemoved
	}
	return closeErr
}

func Write(ctx context.Context, request WriteRequest, source io.Reader) (WriteOutcome, error) {
	return write(ctx, request, source, operatingSystem{})
}

func write(ctx context.Context, request WriteRequest, source io.Reader, filesystem stageFilesystem) (WriteOutcome, error) {
	if source == nil {
		return WriteOutcome{}, core.ErrDurabilityContract
	}
	stage, err := newStage(ctx, request, filesystem)
	if err != nil {
		if stage != nil {
			return newWriteOutcome(stage), err
		}
		return WriteOutcome{}, err
	}
	buffer := make([]byte, core.DurableCopyBufferBytes)
	if _, err := io.CopyBuffer(stage, ContextReader{Context: ctx, Reader: source}, buffer); err != nil {
		writeErr := errors.Join(fmt.Errorf("stream durable stage: %w", err), stage.Abort())
		return newWriteOutcome(stage), writeErr
	}
	_, err = stage.Commit(ctx)
	return newWriteOutcome(stage), err
}

func newWriteOutcome(stage *Stage) WriteOutcome {
	outcome := WriteOutcome{Result: stage.result}
	if writeRecoveryRequired(stage.result) {
		outcome.Recovery = stage
	}
	return outcome
}
