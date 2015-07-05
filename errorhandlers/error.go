package errorhandlers

import (
	"strings"
	"time"
)

// Error.
type Error struct {
	// Command.
	Cmd []string

	// Description.
	Desc string

	// Hostname.
	Hostname string

	// Environment.
	Environ map[string]string

	// Timestamp.
	Timestamp time.Time
}

// Test if a command part should be quoted.
func shouldQuoteCmdPart(part string) bool {
	for _, c := range part {
		if (c < 42 || c > 95) && (c < 97 || c > 122) {
			return true
		}
	}

	return false
}

// Quoted command.
func (e *Error) QuotedCmd() string {
	parts := make([]string, 0, len(e.Cmd))

	for _, p := range e.Cmd {
		if shouldQuoteCmdPart(p) {
			quoted := append(make([]rune, 0), '"')

			for _, c := range p {
				if c == '"' || c == '\\' {
					quoted = append(quoted, '\\')
				}

				quoted = append(quoted, c)
			}

			quoted = append(quoted, '"')
			parts = append(parts, string(quoted))
		} else {
			parts = append(parts, p)
		}
	}

	return strings.Join(parts, " ")
}
