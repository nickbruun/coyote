package output

import (
	"github.com/nickbruun/coyote/utils"
)

// Draining output sink function.
type drainingOutputSink func(lines [][]byte) error

// Draining output close function.
type drainingOutputClose func()

// Draining output.
//
// Output which drains lines while writing previous lines to avoid blocking. If
// sinking fails, it will be retried.
type drainingOutput struct {
	lineCh chan []byte
	done chan struct{}
}

func (o *drainingOutput) Sink(line []byte) {
	o.lineCh <- line
}

func (o *drainingOutput) Close() error {
	// Note: not strictly atomic, but we'll survive for now.
	if o.lineCh != nil {
		close(o.lineCh)
		o.lineCh = nil
	}

	<-o.done

	return nil
}

// New draining output.
//
// The buffer size is the maximum number of lines buffered at once. If the
// buffer is overflown, older lines will be discarded.
func newDrainingOutput(bufferSize int, oSink drainingOutputSink, oClose drainingOutputClose) (Output, error) {
	lineCh := make(chan []byte, 1024)
	done := make(chan struct{})

	go func() {
		lines := utils.NewByteSliceBuffer(bufferSize)

		for l := range lineCh {
			// Drain up until the mark of the buffer size.
			lines.Add(l)

			drained := false
			for !drained && !lines.Full() {
				select {
				case l = <-lineCh:
					lines.Add(l)

				default:
					drained = true
				}
			}

			// Begin sinking the lines.
			sinkLines := lines.Drain()
			sinkDone := make(chan struct{})

			go func() {
				oSink(sinkLines)
				close(sinkDone)
			}()

			done := false
			for !done {
				select {
				case l = <-lineCh:
					lines.Add(l)

				case <-sinkDone:
					done = true
				}
			}
		}

		// Sink any lines left.
		if !lines.Empty() {
			oSink(lines.Drain())
		}

		oClose()
		close(done)
	}()

	return &drainingOutput{
		lineCh: lineCh,
		done: done,
	}, nil
}
