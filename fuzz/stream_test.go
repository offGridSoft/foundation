package fuzz

import "testing"

func TestFuzzLogicalLineCounterChunkBoundaryTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		chunks []string
		want   uint64
	}{
		{name: "neutral empty input"},
		{name: "one trailing fragment", chunks: []string{"alpha"}, want: 1},
		{name: "one delimited line", chunks: []string{"alpha\n"}, want: 1},
		{name: "split delimiter", chunks: []string{"alpha", "\n", "beta"}, want: 2},
		{name: "empty chunks do not invent lines", chunks: []string{"", "alpha", "", "\n", ""}, want: 1},
		{name: "consecutive delimiters remain visible", chunks: []string{"\n", "\n"}, want: 2},
		{name: "binary bytes are content", chunks: []string{"\x00\xff", "\n", "\x80"}, want: 2},
		{name: "carriage return is not delimiter", chunks: []string{"alpha\r", "beta"}, want: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var counter FuzzLogicalLineCounter
			var complete []byte
			for _, chunk := range tc.chunks {
				counter.Write([]byte(chunk))
				complete = append(complete, chunk...)
			}
			if got := counter.Lines(); got != tc.want {
				t.Fatalf("FuzzLogicalLineCounter.Lines() = %d, want %d", got, tc.want)
			}
			if got := CountFuzzLogicalLines(complete); got != tc.want {
				t.Fatalf("CountFuzzLogicalLines() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestFuzzLogicalLineCounterEveryChunkBoundary(t *testing.T) {
	t.Parallel()

	data := []byte("alpha\n\nbeta\x00\xff\ngamma")
	want := CountFuzzLogicalLines(data)
	for split := 0; split <= len(data); split++ {
		var counter FuzzLogicalLineCounter
		counter.Write(data[:split])
		before := counter.Lines()
		if counter.Lines() != before {
			t.Fatalf("split %d Lines() mutated counter", split)
		}
		counter.Write(nil)
		counter.Write(data[split:])
		if got := counter.Lines(); got != want {
			t.Fatalf("split %d Lines() = %d, want chunk-invariant %d", split, got, want)
		}
	}
}

func TestGoFuzzDataDirectoryNameContract(t *testing.T) {
	t.Parallel()

	if GoFuzzDataDirectoryName != "fuzz" {
		t.Fatalf("GoFuzzDataDirectoryName = %q, want Go toolchain directory token", GoFuzzDataDirectoryName)
	}
}
