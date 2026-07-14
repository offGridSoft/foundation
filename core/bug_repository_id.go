package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	BugRepositoryIDEntropyBytes = 32
	BugRepositoryIDPrefix       = "bug-repository-id:"
	ErrFmtBugRepositoryID       = "core.BugRepositoryID: %w"
)

// BugRepositoryID is the committed, clone-stable identity of one Bug
// repository. Certified writer attestations bind this value so Git objects and
// proof artifacts transplanted into another repository cannot be replayed.
type BugRepositoryID struct {
	value string
}

func NewBugRepositoryID(entropy [BugRepositoryIDEntropyBytes]byte) (BugRepositoryID, error) {
	if entropy == [BugRepositoryIDEntropyBytes]byte{} {
		return BugRepositoryID{}, fmt.Errorf(ErrFmtBugRepositoryID, ErrFoundationContract)
	}
	return ParseBugRepositoryID(BugRepositoryIDPrefix + hex.EncodeToString(entropy[:]))
}

func ParseBugRepositoryID(value string) (BugRepositoryID, error) {
	digest, found := strings.CutPrefix(value, BugRepositoryIDPrefix)
	if !found {
		return BugRepositoryID{}, fmt.Errorf(ErrFmtBugRepositoryID, ErrFoundationContract)
	}
	if _, err := ParseSHA256Hex(digest); err != nil {
		return BugRepositoryID{}, fmt.Errorf(ErrFmtBugRepositoryID, ErrFoundationContract)
	}
	if strings.Trim(digest, "0") == "" {
		return BugRepositoryID{}, fmt.Errorf(ErrFmtBugRepositoryID, ErrFoundationContract)
	}
	return BugRepositoryID{value: value}, nil
}

func (id BugRepositoryID) String() string { return id.value }

func (id BugRepositoryID) IsZero() bool { return id.value == "" }

func (id BugRepositoryID) Validate() error {
	_, err := ParseBugRepositoryID(id.value)
	return err
}

func (id BugRepositoryID) MarshalJSON() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(id.value)
}

//validate:unmarshal_ignore reason="ParseBugRepositoryID validates a temporary before assignment so rejected input cannot mutate the receiver."
func (id *BugRepositoryID) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtBugRepositoryID, ErrFoundationContract)
	}
	parsed, err := ParseBugRepositoryID(value)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}
