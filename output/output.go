package output

// Output.
//
// Receives output lines and sinks them.
type Output interface {
	// Sink a line.
	//
	// Must never block to ensure we get things drained to everywhere as
	// quickly as possible.
	Sink(line []byte)

	// Close the output.
	Close() error
}
