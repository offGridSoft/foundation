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
