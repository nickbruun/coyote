package main

import (
	"fmt"
	"github.com/nickbruun/coyote/output"
	"log/syslog"
	"net/url"
	"strings"
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

	// syslog output.
	OutputFlag{
		Name: "syslog",
		Usage: `-syslog[=<facility>[:<tag>]]
    Add a syslog output. The facility can be a one of: KERN, USER, MAIL,
    DAEMON, AUTH, SYSLOG, LPR, NEWS, UUCP, CRON, AUTHPRIV, FTP, LOCAL0,
    LOCAL1, LOCAL2, LOCAL3, LOCAL4, LOCAL5, LOCAL6, LOCAL7. If no facility
    is provided, the output defaults to LOCAL0. The tag will be prefixed
     to any log line.`,
		Parse: func(value string) (output.Output, error) {
			var facilityName, tag string
			colonPos := strings.IndexByte(value, ':')

			if colonPos == -1 {
				facilityName = value
			} else {
				facilityName = value[:colonPos]
				tag = value[colonPos+1:]
			}

			var facility syslog.Priority
			switch strings.ToUpper(facilityName) {
			case "KERN":
				facility = syslog.LOG_KERN
			case "USER":
				facility = syslog.LOG_USER
			case "MAIL":
				facility = syslog.LOG_MAIL
			case "DAEMON":
				facility = syslog.LOG_DAEMON
			case "AUTH":
				facility = syslog.LOG_AUTH
			case "SYSLOG":
				facility = syslog.LOG_SYSLOG
			case "LPR":
				facility = syslog.LOG_LPR
			case "NEWS":
				facility = syslog.LOG_NEWS
			case "UUCP":
				facility = syslog.LOG_UUCP
			case "CRON":
				facility = syslog.LOG_CRON
			case "AUTHPRIV":
				facility = syslog.LOG_AUTHPRIV
			case "FTP":
				facility = syslog.LOG_FTP
			case "", "LOCAL0":
				facility = syslog.LOG_LOCAL0
			case "LOCAL1":
				facility = syslog.LOG_LOCAL1
			case "LOCAL2":
				facility = syslog.LOG_LOCAL2
			case "LOCAL3":
				facility = syslog.LOG_LOCAL3
			case "LOCAL4":
				facility = syslog.LOG_LOCAL4
			case "LOCAL5":
				facility = syslog.LOG_LOCAL5
			case "LOCAL6":
				facility = syslog.LOG_LOCAL6
			case "LOCAL7":
				facility = syslog.LOG_LOCAL7
			default:
				return nil, FlagParseErrorf("invalid syslog facility: %s", facilityName)
			}

			o, err := output.NewSyslogOutput("", "", facility, tag)
			if err != nil {
				return nil, fmt.Errorf("Failed to set up syslog output: %s", err)
			}

			return o, nil
		},
	},
}
