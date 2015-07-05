package output

import (
	"fmt"
	"log/syslog"
	"os"
)

// New syslog TCP output.
//
// If the network is empty, the local syslog daemon will be used.
func NewSyslogOutput(network, raddr string, priority syslog.Priority, tag string) (Output, error) {
	desc := "syslog"
	if network != "" {
		desc = fmt.Sprintf("syslog at %s://%s", network, raddr)
	}
	failing := false
	var w *syslog.Writer

	dial := func() error {
		var err error

		w, err = syslog.Dial(network, raddr, priority, tag)

		if err != nil {
			if !failing {
				failing = true
				fmt.Fprintf(os.Stderr, "Failed to connect to %s: %s\n", desc, err)
			}

			w = nil
			return err
		} else if failing {
			fmt.Fprintf(os.Stderr, "Connected to %s\n", desc)
			failing = false
		}

		return nil
	}

	return newDrainingOutput(10240, func(lines [][]byte) error {
		// Connect if a connection does not already exist.
		if w == nil {
			if err := dial(); err != nil {
				return err
			}
		}

		// Send data.
		first := true

		for _, l := range lines {
			if len(l) == 0 {
				continue
			}

			n, err := w.Write(l)

			// If the first send fails without sending any data, let's attempt
			// to reconnect.
			if first {
				first = false

				if n == 0 && err != nil {
					fmt.Fprintf(os.Stderr, "Failed to send data to %s: %s - reconnecting...\n", desc, err)

					if err = dial(); err != nil {
						return err
					}
				}
			}

			// Update the payload and handle any errors.
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to send data to %s: %s\n", desc, err)

				failing = true
				w.Close()
				w = nil
				return err
			}
		}

		return nil
	}, func() {
		if w != nil {
			w.Close()
		}
	})
}
