package main

import (
	"fmt"
)

// Flag parse error.
type FlagParseError struct {
	Msg string
}

func (e *FlagParseError) Error() string {
	return e.Msg
}

// Flag parse error.
func FlagParseErrorf(format string, a ...interface{}) error {
	return &FlagParseError{fmt.Sprintf(format, a...)}
}
