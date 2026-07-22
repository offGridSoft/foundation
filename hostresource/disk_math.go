package hostresource

import (
	"math"
	"math/bits"

	"github.com/offGridSoft/foundation/v2026/core"
)

func blocksToBytes(blocks, blockBytes uint64) (uint64, error) {
	high, low := bits.Mul64(blocks, blockBytes)
	if high != 0 || low > math.MaxInt64 {
		return 0, core.ErrNumericOverflow
	}
	return low, nil
}
