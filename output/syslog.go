package output

import (
	"log/syslog"
)

// New syslog TCP output.
func NewSyslogOutput(priority syslog.Priority, tag string) (Output, error) {
	w, err := syslog.New(priority, tag)
	if err != nil {
		return nil, err
	}

	return newDrainingOutput(10240, func(lines [][]byte) error {
		for _, l := range lines {
			w.Write(l)
		}

		return nil
	}, func() {
		w.Close()
	})
}
