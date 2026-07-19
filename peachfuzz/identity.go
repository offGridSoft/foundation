package peachfuzz

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	ProjectIDMaxBytes             = 64
	PackageImportPathMaxBytes     = 512
	FuzzTargetNameMaxBytes        = 128
	RunIDTextBytes                = 32
	MachineIDTextBytes            = 32
	CommitSHATextBytes            = 40
	FuzzTargetPrefix              = "Fuzz"
	ProjectIDContractName         = "ProjectID"
	PackageImportPathContractName = "PackageImportPath"
	FuzzTargetNameContractName    = "FuzzTargetName"
	RunIDContractName             = "RunID"
	MachineIDContractName         = "MachineID"
	CommitSHAContractName         = "CommitSHA"
)

type ProjectID struct{ value string }
type PackageImportPath struct{ value string }
type FuzzTargetName struct{ value string }
type RunID struct{ value string }
type MachineID struct{ value string }
type CommitSHA struct{ value string }

func ParseProjectID(value string) (ProjectID, error) {
	if len(value) == 0 || len(value) > ProjectIDMaxBytes || !lowerLetter(value[0]) || value[len(value)-1] == '-' {
		return ProjectID{}, identityError(ProjectIDContractName)
	}
	for index := 1; index < len(value); index++ {
		if !lowerLetter(value[index]) && !decimal(value[index]) && value[index] != '-' {
			return ProjectID{}, identityError(ProjectIDContractName)
		}
	}
	return ProjectID{value: value}, nil
}

func ParsePackageImportPath(value string) (PackageImportPath, error) {
	if len(value) == 0 || len(value) > PackageImportPathMaxBytes {
		return PackageImportPath{}, identityError(PackageImportPathContractName)
	}
	for segment := range strings.SplitSeq(value, "/") {
		if !validImportSegment(segment) {
			return PackageImportPath{}, identityError(PackageImportPathContractName)
		}
	}
	return PackageImportPath{value: value}, nil
}

func ParseFuzzTargetName(value string) (FuzzTargetName, error) {
	if len(value) <= len(FuzzTargetPrefix) || len(value) > FuzzTargetNameMaxBytes || !strings.HasPrefix(value, FuzzTargetPrefix) {
		return FuzzTargetName{}, identityError(FuzzTargetNameContractName)
	}
	suffix := value[len(FuzzTargetPrefix):]
	if suffix != "" && lowerLetter(suffix[0]) {
		return FuzzTargetName{}, identityError(FuzzTargetNameContractName)
	}
	for index := range len(suffix) {
		if !identifierByte(suffix[index]) {
			return FuzzTargetName{}, identityError(FuzzTargetNameContractName)
		}
	}
	return FuzzTargetName{value: value}, nil
}

func ParseRunID(value string) (RunID, error) {
	if !fixedLowerHex(value, RunIDTextBytes) {
		return RunID{}, identityError(RunIDContractName)
	}
	return RunID{value: value}, nil
}

func ParseMachineID(value string) (MachineID, error) {
	if !fixedLowerHex(value, MachineIDTextBytes) {
		return MachineID{}, identityError(MachineIDContractName)
	}
	return MachineID{value: value}, nil
}

func ParseCommitSHA(value string) (CommitSHA, error) {
	if !fixedLowerHex(value, CommitSHATextBytes) {
		return CommitSHA{}, identityError(CommitSHAContractName)
	}
	return CommitSHA{value: value}, nil
}

func (id ProjectID) String() string        { return id.value }
func (p PackageImportPath) String() string { return p.value }
func (n FuzzTargetName) String() string    { return n.value }
func (id RunID) String() string            { return id.value }
func (id MachineID) String() string        { return id.value }
func (s CommitSHA) String() string         { return s.value }

func (id ProjectID) Validate() error        { _, err := ParseProjectID(id.value); return err }
func (p PackageImportPath) Validate() error { _, err := ParsePackageImportPath(p.value); return err }
func (n FuzzTargetName) Validate() error    { _, err := ParseFuzzTargetName(n.value); return err }
func (id RunID) Validate() error            { _, err := ParseRunID(id.value); return err }
func (id MachineID) Validate() error        { _, err := ParseMachineID(id.value); return err }
func (s CommitSHA) Validate() error         { _, err := ParseCommitSHA(s.value); return err }

func identityError(name string) error { return fmt.Errorf(ErrFmtIdentity, name, ErrContract) }

func fixedLowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for index := range len(value) {
		if !decimal(value[index]) && (value[index] < 'a' || value[index] > 'f') {
			return false
		}
	}
	return true
}

func validImportSegment(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for index := range len(value) {
		b := value[index]
		if !identifierByte(b) && b != '.' && b != '-' && b != '~' && b != '+' {
			return false
		}
	}
	return true
}

func identifierByte(value byte) bool {
	return lowerLetter(value) || (value >= 'A' && value <= 'Z') || decimal(value) || value == '_'
}
func lowerLetter(value byte) bool { return value >= 'a' && value <= 'z' }
func decimal(value byte) bool     { return value >= '0' && value <= '9' }

func marshalIdentity(value string, validate func() error) ([]byte, error) {
	if err := validate(); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func (id ProjectID) MarshalJSON() ([]byte, error)        { return marshalIdentity(id.value, id.Validate) }
func (p PackageImportPath) MarshalJSON() ([]byte, error) { return marshalIdentity(p.value, p.Validate) }
func (n FuzzTargetName) MarshalJSON() ([]byte, error)    { return marshalIdentity(n.value, n.Validate) }
func (id RunID) MarshalJSON() ([]byte, error)            { return marshalIdentity(id.value, id.Validate) }
func (id MachineID) MarshalJSON() ([]byte, error)        { return marshalIdentity(id.value, id.Validate) }
func (s CommitSHA) MarshalJSON() ([]byte, error)         { return marshalIdentity(s.value, s.Validate) }

func unmarshalIdentity[T any](target *T, data []byte, parse func(string) (T, error)) error {
	if target == nil {
		return ErrContract
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return ErrContract
	}
	parsed, err := parse(value)
	if err == nil {
		*target = parsed
	}
	return err
}

func (id *ProjectID) UnmarshalJSON(data []byte) error {
	return unmarshalIdentity(id, data, ParseProjectID)
}
func (p *PackageImportPath) UnmarshalJSON(data []byte) error {
	return unmarshalIdentity(p, data, ParsePackageImportPath)
}
func (n *FuzzTargetName) UnmarshalJSON(data []byte) error {
	return unmarshalIdentity(n, data, ParseFuzzTargetName)
}
func (id *RunID) UnmarshalJSON(data []byte) error {
	return unmarshalIdentity(id, data, ParseRunID)
}
func (id *MachineID) UnmarshalJSON(data []byte) error {
	return unmarshalIdentity(id, data, ParseMachineID)
}
func (s *CommitSHA) UnmarshalJSON(data []byte) error {
	return unmarshalIdentity(s, data, ParseCommitSHA)
}
