package output

import (
	"os"
)

// New stdout output.
func NewStdoutOutput() (Output, error) {
	return newFileOutput(os.Stdout)
}
