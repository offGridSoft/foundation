package peachfuzz

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
)

const (
	FuzzCorpusEntryNameMinBytes = 8
	FuzzCorpusEntryNameMaxBytes = 64
)

// FuzzCorpusEntryName is one Go-toolchain corpus filename. Go names corpus
// entries with lowercase hexadecimal content identities.
type FuzzCorpusEntryName struct{ value string }

func ParseFuzzCorpusEntryName(value string) (FuzzCorpusEntryName, error) {
	if len(value) < FuzzCorpusEntryNameMinBytes || len(value) > FuzzCorpusEntryNameMaxBytes {
		return FuzzCorpusEntryName{}, fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
	}
	for _, item := range []byte(value) {
		if (item < '0' || item > '9') && (item < 'a' || item > 'f') {
			return FuzzCorpusEntryName{}, fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
		}
	}
	return FuzzCorpusEntryName{value: value}, nil
}

func (n FuzzCorpusEntryName) String() string { return n.value }

func (n FuzzCorpusEntryName) Validate() error {
	_, err := ParseFuzzCorpusEntryName(n.value)
	return err
}

func (n FuzzCorpusEntryName) MarshalJSON() ([]byte, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(n.value)
}

func (n *FuzzCorpusEntryName) UnmarshalJSON(data []byte) error {
	if n == nil {
		return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtFuzzEvidence, errors.Join(ErrContract, err))
	}
	parsed, err := ParseFuzzCorpusEntryName(value)
	if err != nil {
		return err
	}
	*n = parsed
	return nil
}

// FuzzCorpusSelection retains the canonical bounded prefix of valid corpus
// names. Its zero value is ready for use and memory never exceeds the shared
// artifact-index bound.
type FuzzCorpusSelection struct {
	entries []FuzzCorpusEntryName
	dropped uint64
}

func (s *FuzzCorpusSelection) Observe(name FuzzCorpusEntryName) error {
	if s == nil {
		return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
	}
	if err := name.Validate(); err != nil {
		return err
	}
	position := sort.Search(len(s.entries), func(index int) bool {
		return s.entries[index].String() >= name.String()
	})
	if position < len(s.entries) && s.entries[position] == name {
		return nil
	}
	if len(s.entries) < FuzzArtifactIndexMaxEntries {
		s.entries = append(s.entries, FuzzCorpusEntryName{})
		copy(s.entries[position+1:], s.entries[position:])
		s.entries[position] = name
		return nil
	}
	if s.dropped != math.MaxUint64 {
		s.dropped++
	}
	if position == len(s.entries) {
		return nil
	}
	copy(s.entries[position+1:], s.entries[position:len(s.entries)-1])
	s.entries[position] = name
	return nil
}

func (s FuzzCorpusSelection) Entries() []FuzzCorpusEntryName {
	return append([]FuzzCorpusEntryName(nil), s.entries...)
}

func (s FuzzCorpusSelection) Dropped() uint64 { return s.dropped }

func (s FuzzCorpusSelection) Validate() error {
	if len(s.entries) > FuzzArtifactIndexMaxEntries {
		return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
	}
	for position, entry := range s.entries {
		if err := entry.Validate(); err != nil {
			return err
		}
		if position != 0 && s.entries[position-1].String() >= entry.String() {
			return fmt.Errorf(ErrFmtFuzzEvidence, ErrContract)
		}
	}
	return nil
}
