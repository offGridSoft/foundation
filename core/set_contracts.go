package core

import (
	"fmt"
	"math"
)

type UniqueStringSet struct {
	seen map[string]struct{}
}

func NewUniqueStringSet(capacity int) UniqueStringSet {
	return UniqueStringSet{seen: make(map[string]struct{}, capacity)}
}

func (s *UniqueStringSet) Add(value string) error {
	if value == "" {
		return fmt.Errorf(ErrFmtUniqueToken, ErrFoundationContract)
	}
	if s.seen == nil {
		s.seen = make(map[string]struct{})
	}
	if _, exists := s.seen[value]; exists {
		return fmt.Errorf(ErrFmtUniqueToken, ErrFoundationContract)
	}
	s.seen[value] = struct{}{}
	return nil
}

type ArtifactSetItem interface {
	Validate() error
	ArtifactSetName() string
	ArtifactSetSize() ByteCount
}

type ArtifactSet[T ArtifactSetItem] struct {
	Items      []T
	TotalBytes ByteCount
	Count      uint32
}

func ValidateArtifactSet[T ArtifactSetItem](set ArtifactSet[T]) error {
	if set.Count == 0 || int(set.Count) != len(set.Items) {
		return fmt.Errorf(ErrFmtArtifactSet, ErrFoundationContract)
	}
	sum, err := sumArtifactSet(set.Items)
	if err != nil {
		return err
	}
	if err := set.TotalBytes.Validate(); err != nil {
		return fmt.Errorf(ErrFmtArtifactSet, err)
	}
	if sum != set.TotalBytes.Uint64() {
		return fmt.Errorf(ErrFmtArtifactSet, ErrFoundationContract)
	}
	return nil
}

func sumArtifactSet[T ArtifactSetItem](items []T) (uint64, error) {
	var sum uint64
	names := NewUniqueStringSet(len(items))
	for _, item := range items {
		if err := item.Validate(); err != nil {
			return 0, fmt.Errorf(ErrFmtArtifactSet, err)
		}
		if err := names.Add(item.ArtifactSetName()); err != nil {
			return 0, err
		}
		size := item.ArtifactSetSize().Uint64()
		if size > math.MaxUint64-sum {
			return 0, fmt.Errorf(ErrFmtArtifactSet, ErrFoundationContract)
		}
		sum += size
	}
	return sum, nil
}
