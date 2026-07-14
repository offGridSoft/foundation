package core

import (
	"fmt"
)

type ArtifactSetItem interface {
	Validatable
	ArtifactSetName() string
	ArtifactSetSize() ByteCount
}

type ArtifactSet[T ArtifactSetItem] struct {
	Items      []T
	TotalBytes ByteCount
	Count      uint32
}

func BuildArtifactSet[T ArtifactSetItem](items []T) (ArtifactSet[T], error) {
	copied := append([]T(nil), items...)
	sum, err := sumArtifactSet(copied)
	if err != nil {
		return ArtifactSet[T]{}, err
	}
	count, err := artifactSetCount(copied)
	if err != nil {
		return ArtifactSet[T]{}, err
	}
	total := NewByteCount(sum)
	if err := total.Validate(); err != nil {
		return ArtifactSet[T]{}, fmt.Errorf(ErrFmtArtifactSet, err)
	}
	set := ArtifactSet[T]{
		Items:      copied,
		TotalBytes: total,
		Count:      count,
	}
	return set, nil
}

func artifactSetCount[T ArtifactSetItem](items []T) (uint32, error) {
	count, err := DeriveCollectionCount(len(items), 1, CollectionMaximumDefault)
	if err != nil {
		return 0, fmt.Errorf(ErrFmtArtifactSet, err)
	}
	return count, nil
}

func ValidateArtifactSet[T ArtifactSetItem](set ArtifactSet[T]) error {
	if err := (CollectionCardinality{
		Length:          len(set.Items),
		DeclaredCount:   set.Count,
		Minimum:         1,
		Maximum:         CollectionMaximumDefault,
		RequireDeclared: true,
	}).Validate(); err != nil {
		return fmt.Errorf(ErrFmtArtifactSet, ErrFoundationContract)
	}
	sum, err := sumArtifactSet(set.Items)
	if err != nil {
		return err
	}
	if err := set.TotalBytes.Validate(); err != nil {
		return fmt.Errorf(ErrFmtArtifactSet, err)
	}
	if set.TotalBytes.Uint64() > ArtifactSetMaximumBytes {
		return fmt.Errorf(ErrFmtArtifactSet, ErrFoundationContract)
	}
	if sum != set.TotalBytes.Uint64() {
		return fmt.Errorf(ErrFmtArtifactSet, ErrFoundationContract)
	}
	return nil
}

func sumArtifactSet[T ArtifactSetItem](items []T) (uint64, error) {
	var sum uint64
	for index, item := range items {
		if err := item.Validate(); err != nil {
			return 0, fmt.Errorf(ErrFmtArtifactSet, err)
		}
		for _, prior := range items[:index] {
			if prior.ArtifactSetName() == item.ArtifactSetName() {
				return 0, fmt.Errorf(ErrFmtUniqueToken, ErrFoundationContract)
			}
		}
		size := item.ArtifactSetSize().Uint64()
		if size > ArtifactMaximumBytes || size > ArtifactSetMaximumBytes-sum {
			return 0, fmt.Errorf(ErrFmtArtifactSet, ErrFoundationContract)
		}
		sum += size
	}
	return sum, nil
}
