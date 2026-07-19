package release

import (
	"bytes"
	"io"
	"testing"
)

func TestSelfTestOutputBufferHostileTable(t *testing.T) {
	t.Parallel()
	maximum := UpdateSelfTestOutputMaximumBytes
	cases := []struct {
		name         string
		chunks       []int
		wantBytes    int
		wantOverflow bool
	}{
		{name: "one byte", chunks: []int{1}, wantBytes: 1},
		{name: "split small", chunks: []int{1, 2, 3}, wantBytes: 6},
		{name: "exact maximum", chunks: []int{maximum}, wantBytes: maximum},
		{name: "maximum then empty", chunks: []int{maximum, 0}, wantBytes: maximum},
		{name: "maximum split", chunks: []int{maximum - 1, 1}, wantBytes: maximum},
		{name: "one beyond", chunks: []int{maximum + 1}, wantBytes: maximum, wantOverflow: true},
		{name: "split beyond", chunks: []int{maximum, 1}, wantBytes: maximum, wantOverflow: true},
		{name: "large first", chunks: []int{maximum * 2}, wantBytes: maximum, wantOverflow: true},
		{name: "empty then overflow", chunks: []int{0, maximum + 1}, wantBytes: maximum, wantOverflow: true},
		{name: "overflow remains sticky", chunks: []int{maximum + 1, 0}, wantBytes: maximum, wantOverflow: true},
		{name: "many chunks", chunks: []int{maximum / 2, maximum / 2, 1}, wantBytes: maximum, wantOverflow: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var output SelfTestOutputBuffer
			for _, size := range tc.chunks {
				payload := bytes.Repeat([]byte{'x'}, size)
				written, err := output.Write(payload)
				if err != nil || written != len(payload) {
					t.Fatalf("Write(%d) = (%d, %v), want (%d, nil)", size, written, err, len(payload))
				}
			}
			if output.Len() != tc.wantBytes || output.Overflowed() != tc.wantOverflow {
				t.Fatalf("output = (%d bytes, overflow %t), want (%d, %t)", output.Len(), output.Overflowed(), tc.wantBytes, tc.wantOverflow)
			}
		})
	}
}

func FuzzSelfTestOutputBuffer(f *testing.F) {
	f.Add([]byte("small"), []byte(nil), []byte(nil))
	f.Add(bytes.Repeat([]byte{'x'}, UpdateSelfTestOutputMaximumBytes), []byte(nil), []byte(nil))
	f.Add(bytes.Repeat([]byte{'x'}, UpdateSelfTestOutputMaximumBytes-1), []byte("y"), []byte("z"))
	f.Add([]byte("prefix"), bytes.Repeat([]byte{'x'}, UpdateSelfTestOutputMaximumBytes), []byte("suffix"))
	f.Fuzz(func(t *testing.T, first, second, third []byte) {
		var output SelfTestOutputBuffer
		chunks := [][]byte{first, second, third}
		for index, chunk := range chunks {
			written, err := output.Write(chunk)
			if err != nil || written != len(chunk) {
				t.Fatalf("Write(chunk %d, %d bytes) = (%d, %v), want (%d, nil)", index, len(chunk), written, err, len(chunk))
			}
		}
		combined := make([]byte, 0, len(first)+len(second)+len(third))
		combined = append(combined, first...)
		combined = append(combined, second...)
		combined = append(combined, third...)
		wantRetained := combined
		if len(wantRetained) > UpdateSelfTestOutputMaximumBytes {
			wantRetained = wantRetained[:UpdateSelfTestOutputMaximumBytes]
		}
		if !bytes.Equal(output.Bytes(), wantRetained) {
			t.Fatalf("retained prefix = %d bytes, want %d bytes", output.Len(), len(wantRetained))
		}
		wantOverflow := len(combined) > UpdateSelfTestOutputMaximumBytes
		if output.Overflowed() != wantOverflow {
			t.Fatalf("Overflowed() = %t for %d total bytes, want %t", output.Overflowed(), len(combined), wantOverflow)
		}
	})
}

var _ io.Writer = (*SelfTestOutputBuffer)(nil)
