package fuzz

// GoFuzzDataDirectoryName is the Go toolchain's shared directory token for
// both persisted testdata fuzz corpora and entries beneath GOCACHE.
const GoFuzzDataDirectoryName = "fuzz"

// FuzzLogicalLineCounter owns the fuzz-sidecar line-accounting rule. Newline
// bytes delimit lines; a non-empty trailing fragment is one additional line.
// Write may be called with arbitrary chunk boundaries without changing the
// result.
type FuzzLogicalLineCounter struct {
	lines    uint64
	sawBytes bool
	last     byte
}

// Write observes one retained fuzz-output chunk.
func (c *FuzzLogicalLineCounter) Write(data []byte) {
	for _, value := range data {
		if value == '\n' {
			c.lines++
		}
		c.sawBytes = true
		c.last = value
	}
}

// Lines returns the exact logical-line count observed so far.
func (c FuzzLogicalLineCounter) Lines() uint64 {
	if c.sawBytes && c.last != '\n' {
		return c.lines + 1
	}
	return c.lines
}

// CountFuzzLogicalLines counts one complete retained fuzz sidecar.
func CountFuzzLogicalLines(data []byte) uint64 {
	var counter FuzzLogicalLineCounter
	counter.Write(data)
	return counter.Lines()
}
