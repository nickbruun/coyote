package output

import (
	"net"
	"crypto/tls"
	"time"
)

// New token based TCP output.
//
// Compatible with the Logentries token-based TCP data ingestion protocol:
// https://logentries.com/doc/input-token/
func NewTokenBasedTcpOutput(address, token string, timeout time.Duration, ssl bool) (Output, error) {
	linePrefix := append([]byte(token), ' ')
	var conn net.Conn = nil

	dialer := &net.Dialer{
		Timeout: timeout,
		DualStack: true,
	}

	dial := func() error {
		var err error

		if ssl {
			if conn, err = tls.DialWithDialer(dialer, "tcp", address, nil); err != nil {
				conn = nil
				return err
			}
		} else {
			if conn, err = dialer.Dial("tcp", address); err != nil {
				conn = nil
				return err
			}
		}

		return nil
	}

	return newDrainingOutput(10240, func(lines [][]byte) error {
		// Concatenate the data together.
		size := 0
		for _, l := range lines {
			size += len(linePrefix) + len(l) + 1
		}

		payload := make([]byte, size)
		offset := 0
		for _, l := range lines {
			offset += copy(payload[offset:], linePrefix)
			offset += copy(payload[offset:], l)
			payload[offset] = '\n'
			offset++
		}

		// Connect if a connection does not already exist.
		if conn == nil {
			if err := dial(); err != nil {
				return err
			}
		}

		// Send data.
		first := true

		for len(payload) > 0 {
			n, err := conn.Write(payload)

			// If the first send fails without sending any data, let's attempt
			// to reconnect.
			if first {
				first = false

				if n == 0 && err != nil {
					if err = dial(); err != nil {
						return err
					}
				}
			}

			// Update the payload and handle any errors.
			payload = payload[n:]

			if err != nil {
				conn.Close()
				conn = nil
				return err
			}
		}

		return nil
	}, func() {
		if conn != nil {
			conn.Close()
		}
	})
}
