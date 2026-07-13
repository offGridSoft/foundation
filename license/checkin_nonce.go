package license

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/offGridSoft/foundation/v2026/core"
)

const CheckInNonceBytes = 32

// CheckInNonce is the client-generated challenge that binds one signed
// check-in response to exactly one request.
type CheckInNonce struct {
	value [CheckInNonceBytes]byte
}

func NewCheckInNonce() (CheckInNonce, error) {
	var nonce CheckInNonce
	if _, err := rand.Read(nonce.value[:]); err != nil {
		return CheckInNonce{}, fmt.Errorf(ErrFmtCheckInPayload, err)
	}
	return nonce, nil
}

func ParseCheckInNonce(value string) (CheckInNonce, error) {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != CheckInNonceBytes {
		return CheckInNonce{}, fmt.Errorf(ErrFmtCheckInPayload, core.ErrCheckInNonce)
	}
	var nonce CheckInNonce
	copy(nonce.value[:], decoded)
	if nonce.IsZero() {
		return CheckInNonce{}, fmt.Errorf(ErrFmtCheckInPayload, core.ErrCheckInNonce)
	}
	return nonce, nil
}

func (n CheckInNonce) String() string {
	return hex.EncodeToString(n.value[:])
}

func (n CheckInNonce) IsZero() bool {
	return n == CheckInNonce{}
}

func (n CheckInNonce) Validate() error {
	if n.IsZero() {
		return fmt.Errorf(ErrFmtCheckInPayload, core.ErrCheckInNonce)
	}
	return nil
}

func (n CheckInNonce) MarshalJSON() ([]byte, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(n.String())
}

//validate:unmarshal_ignore reason="ParseCheckInNonce validates a temporary before assignment so rejected input cannot mutate the receiver."
func (n *CheckInNonce) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf(ErrFmtCheckInPayload, core.ErrCheckInNonce)
	}
	parsed, err := ParseCheckInNonce(value)
	if err != nil {
		return err
	}
	*n = parsed
	return nil
}
