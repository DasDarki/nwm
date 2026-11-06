package server

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/DasDarki/nwm/internal/process"
)

type Console struct {
	child  *process.Child
	stdin  io.WriteCloser
	mutex  sync.Mutex
	output sync.WaitGroup
}

func StartConsole(directory string, arguments []string, environment []string, onLine func(line string)) (*Console, error) {
	executable, err := LaunchExecutable(directory)
	if err != nil {
		return nil, err
	}
	command := exec.Command(executable, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), environment...)
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	reader, writer := io.Pipe()
	command.Stdout = writer
	command.Stderr = writer
	child, err := process.StartChild(command)
	if err != nil {
		return nil, err
	}
	console := &Console{child: child, stdin: stdin}
	console.output.Add(1)
	go func() {
		defer console.output.Done()
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			onLine(strings.TrimRight(scanner.Text(), "\r"))
		}
	}()
	go func() {
		<-child.Done()
		writer.Close()
	}()
	return console, nil
}

func (c *Console) PID() int {
	return c.child.PID()
}

func (c *Console) Done() <-chan struct{} {
	return c.child.Done()
}

func (c *Console) Err() error {
	err := c.child.Err()
	c.output.Wait()
	return err
}

func (c *Console) Send(line string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	_, err := io.WriteString(c.stdin, strings.TrimRight(line, "\r\n")+"\n")
	return err
}

func (c *Console) Stop(timeout time.Duration) {
	if c.Send("stop") == nil {
		select {
		case <-c.child.Done():
			c.output.Wait()
			return
		case <-time.After(timeout):
		}
	}
	c.child.Stop(5 * time.Second)
	c.output.Wait()
}
