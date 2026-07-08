package release

import (
	"errors"
	"fmt"

	"github.com/offGridSoft/foundation/core"
)

func wrapReleaseContract(format string, err error) error {
	return fmt.Errorf(format, errors.Join(core.ErrReleaseContract, err))
}
