package release

import "bytes"

// SelfTestOutputBuffer bounds output from an untrusted candidate binary until
// the caller verifies that candidate against the signed release target.
type SelfTestOutputBuffer struct {
	data     []byte
	overflow bool
}

// Write implements io.Writer while retaining at most
// UpdateSelfTestOutputMaximumBytes. It reports the full input length so a
// candidate cannot distinguish retained from discarded output.
func (b *SelfTestOutputBuffer) Write(data []byte) (int, error) {
	written := len(data)
	if written == 0 {
		return 0, nil
	}
	remaining := UpdateSelfTestOutputMaximumBytes - len(b.data)
	if remaining <= 0 {
		b.overflow = true
		return written, nil
	}
	if written > remaining {
		b.data = append(b.data, data[:remaining]...)
		b.overflow = true
		return written, nil
	}
	b.data = append(b.data, data...)
	return written, nil
}

// Bytes returns an owned copy of the retained prefix.
func (b *SelfTestOutputBuffer) Bytes() []byte {
	return bytes.Clone(b.data)
}

// Len returns the retained byte count.
func (b *SelfTestOutputBuffer) Len() int {
	return len(b.data)
}

// Overflowed reports whether any non-empty write crossed the retention cap.
func (b *SelfTestOutputBuffer) Overflowed() bool {
	return b.overflow
}
