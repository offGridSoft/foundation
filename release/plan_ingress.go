package release

import (
	"fmt"
	"io"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	ReleasePlanMaximumBytes           int64 = 256 << 10
	ReleasePlanStructuralMaximumBytes int64 = 64 << 10
	ReleasePlanJSONExpansionMaximum   int64 = 6
	ReleasePlanFastRawByteMaximum           = (ReleasePlanMaximumBytes - ReleasePlanStructuralMaximumBytes) / ReleasePlanJSONExpansionMaximum
)

func ParseReleasePlan(data []byte) (ReleasePlan, error) {
	if len(data) == 0 || int64(len(data)) > ReleasePlanMaximumBytes {
		return ReleasePlan{}, fmt.Errorf(ErrFmtReleasePlan, core.ErrReleaseContract)
	}
	plan, err := core.DecodeStrictJSON[ReleasePlan](data)
	if err != nil {
		return ReleasePlan{}, wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	return plan, nil
}

func ReadReleasePlan(reader io.Reader) (ReleasePlan, error) {
	if reader == nil {
		return ReleasePlan{}, fmt.Errorf(ErrFmtReleasePlan, core.ErrReleaseContract)
	}
	data, err := io.ReadAll(io.LimitReader(reader, ReleasePlanMaximumBytes+1))
	if err != nil {
		return ReleasePlan{}, wrapReleaseContract(ErrFmtReleasePlan, err)
	}
	return ParseReleasePlan(data)
}
