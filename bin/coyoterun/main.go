package main

import (
	"fmt"
	"bufio"
	"io"
	"os"
	"os/exec"
	"github.com/nickbruun/coyote/output"
	"sync"
	"syscall"
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

func main() {
	stderrOutputs := make([]output.Output, 0)
	stdoutOutputs := make([]output.Output, 0)

	stdoutOutput, _ := output.NewStdoutOutput()
	stdoutOutputs = append(stdoutOutputs, stdoutOutput)

	stderrOutput, _ := output.NewStderrOutput()
	stderrOutputs = append(stderrOutputs, stderrOutput)

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	// "bash", "-c", `while true; do echo "HELLO!"; sleep 1; echo "error" 1>&2; sleep 1; done`

	// Set up and drain output.
	var stdoutWg, stderrWg sync.WaitGroup

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Failed to set up stdout pipe: %s\n", err))
		os.Exit(1)
	}
	stdoutWg.Add(1)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Failed to set up stderr pipe: %s\n", err))
		os.Exit(1)
	}
	stderrWg.Add(1)

	// Start the process.
	var exitStatus int

	if err := cmd.Start(); err != nil {
		sinkLine([]byte(fmt.Sprintf("Unable to start process: %s", err)), stderrOutputs)
		exitStatus = 1
	} else {
		go drainOutput(stdout, stdoutOutputs, &stdoutWg)
		go drainOutput(stderr, stderrOutputs, &stderrWg)

		// Wait for the process to finish.
		waitErr := cmd.Wait()

		// Close the process output readers and wait for draining to finish.
		stdoutWg.Wait()
		stderrWg.Wait()

		// If the process exited abnormally, write that as a log line to stderr
		// output.
		if waitErr != nil {
			sinkLine([]byte(fmt.Sprintf("Process exited abnormally: %s", waitErr)), stderrOutputs)
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
	for _, o := range stdoutOutputs {
		o.Close()
	}

	for _, o := range stderrOutputs {
		o.Close()
	}

	os.Exit(exitStatus)
}
