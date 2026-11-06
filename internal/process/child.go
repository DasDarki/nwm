package process

import (
	"os/exec"
	"sync"
	"time"
)

type Child struct {
	command *exec.Cmd
	tree    *processTree
	done    chan struct{}
	err     error
	once    sync.Once
}

func StartChild(command *exec.Cmd) (*Child, error) {
	command.SysProcAttr = childAttributes()
	if err := command.Start(); err != nil {
		return nil, err
	}
	tree, err := attachTree(command.Process.Pid)
	if err != nil {
		command.Process.Kill()
		command.Wait()
		return nil, err
	}
	child := &Child{command: command, tree: tree, done: make(chan struct{})}
	go func() {
		child.err = command.Wait()
		close(child.done)
	}()
	return child, nil
}

func (c *Child) PID() int {
	return c.command.Process.Pid
}

func (c *Child) Done() <-chan struct{} {
	return c.done
}

func (c *Child) Err() error {
	<-c.done
	return c.err
}

func (c *Child) Stop(timeout time.Duration) {
	c.once.Do(func() {
		c.tree.terminate()
		select {
		case <-c.done:
		case <-time.After(timeout):
			c.tree.kill()
			<-c.done
		}
		c.tree.close()
	})
}
