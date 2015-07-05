package main

import (
	"bufio"
	"fmt"
	"github.com/nickbruun/coyote"
	"github.com/nickbruun/coyote/errorhandlers"
	"github.com/nickbruun/coyote/output"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

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

// Emit error.
func emitError(cmd []string, err error, errorHandlers []errorhandlers.Handler) {
	// Construct the error.
	timestamp := time.Now().UTC()
	hostname, _ := os.Hostname()

	environ := make(map[string]string, len(os.Environ()))
	for _, env := range os.Environ() {
		var k, v string

		equalPos := strings.IndexByte(env, '=')
		if equalPos == -1 {
			k = env
		} else {
			k = env[:equalPos]
			v = env[equalPos+1:]
		}

		environ[k] = v
	}

	errMsg := &errorhandlers.Error{
		Cmd:       cmd,
		Desc:      err.Error(),
		Hostname:  hostname,
		Environ:   environ,
		Timestamp: timestamp,
	}

	// Emit the error.
	for _, h := range errorHandlers {
		if err := h.Handle(errMsg); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to send error message: %s\n", err)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: %s [OPTIONS] <command> [ARGS]

Output options:

`, filepath.Base(os.Args[0]))

	for _, f := range outputFlags {
		fmt.Fprintf(os.Stderr, "%s\n", f.Usage)
	}
}

func usageError(desc string) {
	fmt.Fprintf(os.Stderr, "%s\n\n", desc)
	usage()
	os.Exit(1)
}

func main() {
	var outputs []output.Output
	var errorHandlers []errorhandlers.Handler

	// Parse argument flags.
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

		case "v", "version":
			fmt.Fprintf(os.Stderr, "coyoterun version %s\n", coyote.VERSION)
			os.Exit(0)

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
		emitError(cmdArgs, fmt.Errorf("unable to start process: %s", err), errorHandlers)
		exitStatus = 1
	} else {
		// Set up signal handler.
		var lastSig os.Signal
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGABRT, syscall.SIGALRM, syscall.SIGFPE, syscall.SIGHUP, syscall.SIGILL, syscall.SIGINT, syscall.SIGPIPE, syscall.SIGQUIT, syscall.SIGSEGV, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)

		go func() {
			for sig := range sigs {
				cmd.Process.Signal(sig)
				lastSig = sig
			}
		}()

		// Drain output.
		go drainOutput(stdout, outputs, &drainWg)
		go drainOutput(stderr, outputs, &drainWg)

		// Wait for the process to finish.
		waitErr := cmd.Wait()

		// Close the process output readers and wait for draining to finish.
		drainWg.Wait()

		// Exit with the status of the process.
		exitUnexpected := true

		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			if waitStatus, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitStatus = waitStatus.ExitStatus()

				if lastSig != nil && waitStatus.Signal() == lastSig {
					exitUnexpected = false
				}
			} else {
				exitStatus = 1
			}
		}

		// If the process exited abnormally, write that as a log line to stderr
		// output and emit an error.
		if waitErr != nil && exitUnexpected {
			sinkLine([]byte(fmt.Sprintf("Process exited abnormally: %s", waitErr)), outputs)
			emitError(cmdArgs, waitErr, errorHandlers)
		}
	}

	// Close the outputs.
	for _, o := range outputs {
		o.Close()
	}

	os.Exit(exitStatus)
}
