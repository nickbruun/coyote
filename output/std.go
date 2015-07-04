package output

import (
	"os"
)

// New stdout output.
func NewStdoutOutput() (Output, error) {
	return newFileOutput(os.Stdout)
}

// New stderr output.
func NewStderrOutput() (Output, error) {
	return newFileOutput(os.Stderr)
}
