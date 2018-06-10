package shell

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Shell represents the shell process that runs in the background and executes the commands.
type Shell struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

// StartShell starts a shell as a background process
func StartShell() (Shell, error) {
	shell := "/bin/sh"
	cmd := exec.Command(shell)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return Shell{}, fmt.Errorf("Unable to set up input stream for shell %s: %v", shell, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Shell{}, fmt.Errorf("Unable to set up output stream for shell %s: %v", shell, err)
	}
	err = cmd.Start()
	if err != nil {
		return Shell{}, fmt.Errorf("Unable to start shell %s: %v", shell, err)
	}
	return Shell{cmd, stdin, stdout}, nil
}

// ExecuteCommand runs a command in the shell and returns its output and exit code
func (shell *Shell) ExecuteCommand(command string) ([]string, int, error) {
	const (
		beginMarker = ">>>>>>>>>>SHELLDOC_MARKER>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
		endMarker   = "<<<<<<<<<<SHELLDOC_MARKER"
	)
	instruction := fmt.Sprintf("%s\n", strings.TrimSpace(command))
	io.WriteString(shell.stdin, fmt.Sprintf("echo \"%s\"\n", beginMarker))
	io.WriteString(shell.stdin, instruction)
	io.WriteString(shell.stdin, fmt.Sprintf("echo \"%s $?\"\n", endMarker))

	// read output (TODO: with timeout), watch for markers:
	beginEx := fmt.Sprintf("^%s$", beginMarker)
	beginRx := regexp.MustCompile(beginEx)
	endEx := fmt.Sprintf("^%s (.+)$", endMarker)
	endRx := regexp.MustCompile(endEx)

	var output []string
	var rc int
	beginFound := false
	scanner := bufio.NewScanner(shell.stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if beginRx.MatchString(line) {
			beginFound = true
			continue
		}
		if beginFound == false {
			continue
		}
		match := endRx.FindStringSubmatch(line)
		if len(match) > 1 {
			value, err := strconv.Atoi(match[1])
			if err != nil {
				return nil, -1, fmt.Errorf("unable to read exit code for shell command: %v", err)
			}
			rc = value
			break
		}
		output = append(output, line)
	}
	return output, rc, nil
}

// Exit tells a running shell to exit and waits for it
func (shell *Shell) Exit() error {
	io.WriteString(shell.stdin, "exit\n")
	return shell.cmd.Wait()
}
