package hostresource

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"

	"github.com/offGridSoft/foundation/v2026/contextcheck"
	"github.com/offGridSoft/foundation/v2026/core"
)

type DiskCapacity struct {
	FreeBytes  uint64
	TotalBytes uint64
}

func (c DiskCapacity) Validate() error {
	if c.TotalBytes == 0 || c.FreeBytes > c.TotalBytes || c.TotalBytes > math.MaxInt64 {
		return core.ErrHostResourceContract
	}
	return nil
}

type DiskPressureState uint8

const (
	DiskPressureUnknown DiskPressureState = iota
	DiskPressureDisabled
	DiskPressureHealthy
	DiskPressureReached
)

func (s DiskPressureState) Validate() error {
	if !s.IsValid() {
		return core.ErrHostResourceContract
	}
	return nil
}

type DiskPressurePolicy struct {
	FloorBytes uint64
}

func (p DiskPressurePolicy) Validate() error {
	if p.FloorBytes > math.MaxInt64 {
		return core.ErrHostResourceContract
	}
	return nil
}

type DiskAssessment struct {
	Capacity DiskCapacity
	Policy   DiskPressurePolicy
	State    DiskPressureState
}

func NewDiskAssessment(capacity DiskCapacity, policy DiskPressurePolicy) (DiskAssessment, error) {
	if err := capacity.Validate(); err != nil {
		return DiskAssessment{}, err
	}
	if err := policy.Validate(); err != nil {
		return DiskAssessment{}, err
	}
	state := DiskPressureHealthy
	if policy.FloorBytes == 0 {
		state = DiskPressureDisabled
	} else if capacity.FreeBytes <= policy.FloorBytes {
		state = DiskPressureReached
	}
	assessment := DiskAssessment{Capacity: capacity, Policy: policy, State: state}
	return assessment, assessment.Validate()
}

func (a DiskAssessment) Validate() error {
	if err := a.Capacity.Validate(); err != nil {
		return err
	}
	if err := a.Policy.Validate(); err != nil {
		return err
	}
	if err := a.State.Validate(); err != nil {
		return err
	}
	expected, err := NewDiskAssessmentState(a.Capacity, a.Policy)
	if err != nil || expected != a.State {
		return core.ErrHostResourceContract
	}
	return nil
}

func NewDiskAssessmentState(capacity DiskCapacity, policy DiskPressurePolicy) (DiskPressureState, error) {
	if err := capacity.Validate(); err != nil {
		return DiskPressureUnknown, err
	}
	if err := policy.Validate(); err != nil {
		return DiskPressureUnknown, err
	}
	if policy.FloorBytes == 0 {
		return DiskPressureDisabled, nil
	}
	if capacity.FreeBytes <= policy.FloorBytes {
		return DiskPressureReached, nil
	}
	return DiskPressureHealthy, nil
}

func CheckDiskFloor(assessment DiskAssessment) error {
	if err := assessment.Validate(); err != nil {
		return err
	}
	if assessment.State == DiskPressureReached {
		return core.ErrDiskFloorReached
	}
	return nil
}

func ProbeDisk(ctx context.Context, path core.AbsoluteDirectoryPath) (DiskCapacity, error) {
	if err := contextcheck.Validate(ctx); err != nil {
		return DiskCapacity{}, err
	}
	if err := path.Validate(); err != nil {
		return DiskCapacity{}, err
	}
	capacity, err := probeDisk(path)
	if err != nil {
		return DiskCapacity{}, fmt.Errorf("probe disk capacity: %w", err)
	}
	if err := capacity.Validate(); err != nil {
		return DiskCapacity{}, err
	}
	return capacity, nil
}

type MissingPathPolicy uint8

const (
	MissingPathUnknown MissingPathPolicy = iota
	MissingPathReject
	MissingPathIsEmpty
)

func (p MissingPathPolicy) Validate() error {
	if !p.IsValid() {
		return core.ErrHostResourceContract
	}
	return nil
}

type TreeUsageRequest struct {
	Root          core.AbsoluteDirectoryPath
	MissingPolicy MissingPathPolicy
}

func (r TreeUsageRequest) Validate() error {
	if err := r.Root.Validate(); err != nil {
		return err
	}
	return r.MissingPolicy.Validate()
}

type TreeUsage struct {
	RegularFileBytes uint64
	RegularFileCount uint64
}

func (u TreeUsage) Validate() error {
	if u.RegularFileBytes > math.MaxInt64 {
		return core.ErrHostResourceContract
	}
	return nil
}

func MeasureTree(ctx context.Context, request TreeUsageRequest) (TreeUsage, error) {
	return measureTree(ctx, request, filepath.WalkDir)
}

type walkDirectory func(string, fs.WalkDirFunc) error

func measureTree(ctx context.Context, request TreeUsageRequest, walk walkDirectory) (TreeUsage, error) {
	if err := contextcheck.Validate(ctx); err != nil {
		return TreeUsage{}, err
	}
	if err := request.Validate(); err != nil {
		return TreeUsage{}, err
	}
	root := request.Root.String()
	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, fs.ErrNotExist) && request.MissingPolicy == MissingPathIsEmpty {
			return TreeUsage{}, nil
		}
		return TreeUsage{}, err
	}
	usage := TreeUsage{}
	err := walk(root, treeVisitor(ctx, &usage))
	if err != nil {
		return TreeUsage{}, fmt.Errorf("measure filesystem tree: %w", err)
	}
	return usage, usage.Validate()
}

func treeVisitor(ctx context.Context, usage *TreeUsage) fs.WalkDirFunc {
	return func(_ string, entry fs.DirEntry, walkErr error) error {
		if contextErr := contextcheck.Validate(ctx); contextErr != nil {
			return contextErr
		}
		if walkErr != nil {
			return walkErr
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		return addRegularFile(usage, info.Size())
	}
}

func addRegularFile(usage *TreeUsage, size int64) error {
	if size < 0 || uint64(size) > math.MaxInt64-usage.RegularFileBytes || usage.RegularFileCount == math.MaxUint64 {
		return core.ErrNumericOverflow
	}
	usage.RegularFileBytes += uint64(size)
	usage.RegularFileCount++
	return nil
}
