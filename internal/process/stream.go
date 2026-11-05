package process

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
)

const tailSize = 40

type CommandError struct {
	Command string
	Err     error
	Tail    []string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("%s failed: %v", e.Command, e.Err)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

func Stream(command *exec.Cmd, onLine func(line string)) error {
	reader, writer := io.Pipe()
	command.Stdout = writer
	command.Stderr = writer
	command.WaitDelay = 5 * time.Second
	name := filepath.Base(command.Path)
	if err := command.Start(); err != nil {
		writer.Close()
		return &CommandError{Command: name, Err: err}
	}
	var tail []string
	scanned := make(chan struct{})
	go func() {
		defer close(scanned)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		scanner.Split(splitLines)
		for scanner.Scan() {
			line := strings.TrimSpace(ansi.Strip(scanner.Text()))
			if line == "" {
				continue
			}
			tail = append(tail, line)
			if len(tail) > tailSize {
				tail = tail[len(tail)-tailSize:]
			}
			if onLine != nil {
				onLine(line)
			}
		}
		io.Copy(io.Discard, reader)
	}()
	err := command.Wait()
	writer.Close()
	<-scanned
	if err != nil {
		return &CommandError{Command: name, Err: err, Tail: tail}
	}
	return nil
}

func splitLines(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if index := bytes.IndexAny(data, "\r\n"); index >= 0 {
		return index + 1, data[:index], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
