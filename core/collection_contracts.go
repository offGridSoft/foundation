package core

import "fmt"

const (
	CollectionMaximumDefault    uint32 = 1024
	HTTPHeaderMaximumDefault    uint32 = 64
	PlatformMaximumDefault      uint32 = 16
	ArtifactMaximumBytes        uint64 = 256 << 20
	ArtifactSetMaximumBytes     uint64 = 512 << 20
	ErrFmtCollectionCardinality        = "core.CollectionCardinality: %w"
)

// CollectionCardinality owns the shared bounded-collection invariant.
// DeclaredCount participates only when RequireDeclared is true.
type CollectionCardinality struct {
	Length          int
	DeclaredCount   uint32
	Minimum         uint32
	Maximum         uint32
	RequireDeclared bool
}

func DeriveCollectionCount(length int, minimum, maximum uint32) (uint32, error) {
	if err := (CollectionCardinality{Length: length, Minimum: minimum, Maximum: maximum}).Validate(); err != nil {
		return 0, err
	}
	var count uint32
	for range length {
		count++
	}
	return count, nil
}

func (c CollectionCardinality) Validate() error {
	if c.Length < 0 || c.Maximum == 0 || c.Minimum > c.Maximum {
		return fmt.Errorf(ErrFmtCollectionCardinality, ErrFoundationContract)
	}
	length := uint64(c.Length)
	if length < uint64(c.Minimum) || length > uint64(c.Maximum) {
		return fmt.Errorf(ErrFmtCollectionCardinality, ErrFoundationContract)
	}
	if c.RequireDeclared && uint64(c.DeclaredCount) != length {
		return fmt.Errorf(ErrFmtCollectionCardinality, ErrFoundationContract)
	}
	if !c.RequireDeclared && c.DeclaredCount != 0 {
		return fmt.Errorf(ErrFmtCollectionCardinality, ErrFoundationContract)
	}
	return nil
}
