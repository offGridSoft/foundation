package release

import (
	"crypto/sha256"
	"encoding/base64"

	"github.com/offGridSoft/foundation/v2026/core"
)

const GarbleSeedDerivationDomain = "foundation-garble-seed-" + core.ContractYear

// DeriveGarbleSeed binds a product-scoped custody root to one exact release.
// The short seed is the only value passed to Garble; the long-lived root stays
// within the release-data custody boundary.
func DeriveGarbleSeed(custody core.GarbleCustodySeed, releaseID ReleaseID) (GarbleSeed, error) {
	if err := custody.Validate(); err != nil {
		return GarbleSeed{}, err
	}
	if err := releaseID.Validate(); err != nil {
		return GarbleSeed{}, err
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte(GarbleSeedDerivationDomain))
	_, _ = hash.Write([]byte{core.SignedMessageSep})
	_, _ = hash.Write([]byte(releaseID.String()))
	_, _ = hash.Write([]byte{core.SignedMessageSep})
	root := custody.Bytes()
	defer clear(root)
	_, _ = hash.Write(root)
	effective := base64.RawStdEncoding.EncodeToString(hash.Sum(nil)[:GarbleSeedBytes])
	return ParseRequiredGarbleSeed(effective)
}
