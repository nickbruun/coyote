package output

import (
	"bytes"
	"os"
)

// New file output.
func newFileOutput(f *os.File) (Output, error) {
	return newDrainingOutput(10240, func(lines [][]byte) error {
		_, err := f.Write(bytes.Join(append(lines, []byte{}), lineEnding))
		return err
	}, func() {
		f.Sync()

		if f != os.Stdout && f != os.Stderr {
			f.Close()
		}
	})
}
