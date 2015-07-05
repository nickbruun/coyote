package main

import (
	"fmt"
	"github.com/nickbruun/coyote/output"
	"net/url"
	"time"
)

// Output flag.
type OutputFlag struct {
	// Flag name.
	Name string

	// Usage information.
	Usage string

	// Parse flag.
	Parse func(value string) (output.Output, error)
}

// Output flags.
var outputFlags = []OutputFlag{
	// Stdout.
	OutputFlag{
		Name: "stdout",
		Usage: `-stdout
    Add a stdout output.`,
		Parse: func(value string) (output.Output, error) {
			if value != "" {
				return nil, FlagParseErrorf("stdout does not accept a value")
			}

			o, err := output.NewStdoutOutput()
			if err != nil {
				return nil, fmt.Errorf("Failed to create stdout output: %s\n", err)
			}

			return o, nil
		},
	},

	// Token-based TCP output.
	OutputFlag{
		Name: "token-based-tcp",
		Usage: `-token-based-tcp=tcp[s]://<host>:<port>/<token>
    Add a token-based TCP output, which sends token-prefixed lines over
    an optionally SSL-encrypted TCP connection. For example, to use secure
    Logentries token-based TCP output:

        tcps://api.logentries.com:20000/2bfbea1e-10c3-4419-bdad-7e6435882e1f`,
		Parse: func(value string) (output.Output, error) {
			if value == "" {
				return nil, FlagParseErrorf("no URL provided for token-based TCP output.")
			}

			url, err := url.Parse(value)
			if err != nil {
				return nil, FlagParseErrorf("Error: invalid URL provided for token-based TCP output: %s", err)
			}

			var ssl bool

			if url.Scheme == "tcp" {
				ssl = false
			} else if url.Scheme == "tcps" {
				ssl = true
			} else {
				return nil, FlagParseErrorf("invalid URL scheme for token-based TCP output: %s", url.Scheme)
			}

			token := url.Path[1:]
			if token == "" {
				return nil, FlagParseErrorf("no token specified for token-based TCP output.")
			}

			o, err := output.NewTokenBasedTcpOutput(url.Host, token, 5*time.Second, ssl)
			if err != nil {
				return nil, fmt.Errorf("Failed to create token-based TCP output: %s\n", err)
			}

			return o, nil
		},
	},
}
