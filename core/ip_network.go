package core

import (
	"fmt"
	// witness:waiver doctrine/purity/io -- net/netip performs deterministic in-memory address parsing and matching; it opens no network or operating-system resource.
	"net/netip"
)

// IPNetwork is a proven canonical exact-IP or CIDR identity. Exact addresses
// are represented internally as full-width prefixes so matching has one path.
type IPNetwork struct {
	prefix netip.Prefix
}

// ParseIPNetwork accepts only canonical unmapped IP addresses and masked CIDR
// prefixes. The original spelling must equal the canonical spelling.
func ParseIPNetwork(raw string) (IPNetwork, error) {
	if address, err := netip.ParseAddr(raw); err == nil {
		return ipNetworkFromAddress(raw, address)
	}
	return ipNetworkFromPrefix(raw)
}

func ipNetworkFromAddress(raw string, address netip.Addr) (IPNetwork, error) {
	if address.Is4In6() || address.Zone() != "" || address.String() != raw {
		return IPNetwork{}, fmt.Errorf(ErrFmtIPNetwork, ErrIPNetworkContract)
	}
	return IPNetwork{prefix: netip.PrefixFrom(address, address.BitLen())}, nil
}

func ipNetworkFromPrefix(raw string) (IPNetwork, error) {
	prefix, err := netip.ParsePrefix(raw)
	if err != nil || prefix.Addr().Is4In6() || prefix.Masked().String() != raw {
		return IPNetwork{}, fmt.Errorf(ErrFmtIPNetwork, ErrIPNetworkContract)
	}
	return IPNetwork{prefix: prefix}, nil
}

// Validate proves that n was created from the canonical parser.
func (n IPNetwork) Validate() error {
	if !n.prefix.IsValid() || n.prefix != n.prefix.Masked() {
		return fmt.Errorf(ErrFmtIPNetwork, ErrIPNetworkContract)
	}
	return nil
}

// String returns the canonical address for an exact match and canonical CIDR
// for a range.
func (n IPNetwork) String() string {
	if !n.prefix.IsValid() {
		return ""
	}
	if n.prefix.Bits() == n.prefix.Addr().BitLen() {
		return n.prefix.Addr().String()
	}
	return n.prefix.String()
}

// Contains reports whether canonicalIP belongs to n. The candidate must itself
// be a canonical, unmapped address; malformed or alternate spellings fail.
func (n IPNetwork) Contains(canonicalIP string) (bool, error) {
	if err := n.Validate(); err != nil {
		return false, err
	}
	address, err := netip.ParseAddr(canonicalIP)
	if err != nil || address.Is4In6() || address.Zone() != "" || address.String() != canonicalIP {
		return false, fmt.Errorf(ErrFmtIPNetwork, ErrIPNetworkContract)
	}
	return n.prefix.Contains(address), nil
}
