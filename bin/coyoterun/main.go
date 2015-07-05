package main

import (
	"bufio"
	"fmt"
	"github.com/nickbruun/coyote/output"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

// Outputs.
var outputs []output.Output

// Sink a line.
func sinkLine(l []byte, outputs []output.Output) {
	for _, o := range outputs {
		o.Sink(l)
	}
}

// Drain and sink output from a reader.
func drainOutput(r io.Reader, outputs []output.Output, wg *sync.WaitGroup) {
	br := bufio.NewReader(r)

	for {
		// Read as many lines as we can.
		line, err := br.ReadBytes('\n')

		if len(line) > 0 {
			// Strip any [CR]LF from the line.
			line = line[:len(line)-1]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}

			// Sink the output.
			sinkLine(line, outputs)
		}

		if err != nil {
			break
		}
	}

	wg.Done()
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: %s [OPTIONS] <command> [ARGS]

Output options:

-stdout
    Add a stdout output.
-token-based-tcp=tcp[s]://<host>:<port>/<token>
    Add a token-based TCP output, which sends token-prefixed lines over
    an optionally SSL-encrypted TCP connection. For example, to use secure
    Logentries token-based TCP output:

        tcps://api.logentries.com:20000/2bfbea1e-10c3-4419-bdad-7e6435882e1f
`, filepath.Base(os.Args[0]))
}

func usageError(desc string) {
	fmt.Fprintf(os.Stderr, "%s\n\n", desc)
	usage()
	os.Exit(1)
}

func main() {
	var i int
	var arg string
	for i, arg = range os.Args[1:] {
		// Skip empty arguments.
		if arg == "" {
			continue
		}

		// Break at the first non-flag argument or a clean "-" argument.
		if arg[0] != '-' || arg == "-" {
			break
		}

		// Parse the argument into a flag and a value.
		equalPos := strings.IndexByte(arg, '=')
		var flag, value string
		if equalPos == -1 {
			flag = arg[1:]
		} else {
			flag = arg[1:equalPos]
			value = arg[equalPos+1:]
		}

		// Attempt to handle the argument with an output flag parser.
		handled := false
		for _, f := range outputFlags {
			if flag == f.Name {
				handled = true

				o, err := f.Parse(value)
				if err != nil {
					if flagErr, ok := err.(*FlagParseError); ok {
						usageError(flagErr.Error())
					} else {
						fmt.Fprintf(os.Stderr, "%s\n", err)
						os.Exit(1)
					}
				} else {
					outputs = append(outputs, o)
				}
			}
		}

		if handled {
			continue
		}

		// Fall back to default flag parsing.
		switch flag {
		case "h", "help", "?":
			usage()
			os.Exit(1)

		default:
			usageError(fmt.Sprintf("Error: unknown flag: %s", arg))
		}
	}

	// Ensure output are provided.
	if len(outputs) == 0 {
		usageError("Error: no outputs specified.")
	}

	// Parse the command line.
	if arg == "-" {
		i++
	}

	cmdArgs := os.Args[i+1:]

	if len(cmdArgs) == 0 {
		usageError("Error: no command specified.")
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

	// Set up and drain output.
	var drainWg sync.WaitGroup

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set up stdout pipe: %s\n", err)
		os.Exit(1)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set up stderr pipe: %s\n", err)
		os.Exit(1)
	}

	drainWg.Add(2)

	// Start the process.
	var exitStatus int

	if err := cmd.Start(); err != nil {
		sinkLine([]byte(fmt.Sprintf("Unable to start process: %s", err)), outputs)
		exitStatus = 1
	} else {
		go drainOutput(stdout, outputs, &drainWg)
		go drainOutput(stderr, outputs, &drainWg)

		// Wait for the process to finish.
		waitErr := cmd.Wait()

		// Close the process output readers and wait for draining to finish.
		drainWg.Wait()

		// If the process exited abnormally, write that as a log line to stderr
		// output.
		if waitErr != nil {
			sinkLine([]byte(fmt.Sprintf("Process exited abnormally: %s", waitErr)), outputs)
		}

		// Exit with the status of the process.
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			if waitStatus, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitStatus = waitStatus.ExitStatus()
			} else {
				exitStatus = 1
			}
		}
	}

	// Close the outputs.
	for _, o := range outputs {
		o.Close()
	}

	os.Exit(exitStatus)
}
