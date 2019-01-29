package eexec

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

type RespawnPolicy struct {
	// 0 for unlimited
	Times int
	// TODO: Add Delay between restarts, etc.
}

type Command struct {
	name    string
	args    []string
	env     []string
	stdoutr *os.File
	stdoutw *os.File
	stderrr *os.File
	stderrw *os.File
	// Guards cmd and signal.
	cmdMu  sync.Mutex
	cmd    *exec.Cmd
	signal bool
	exited sync.WaitGroup
}

func NewCommand(name string, args []string, env []string) (*Command, error) {
	stdoutr, stdoutw, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %s", err)
	}
	stderrr, stderrw, err := os.Pipe()
	if err != nil {
		defer stdoutr.Close()
		defer stdoutw.Close()

		return nil, fmt.Errorf("failed to create pipe: %s", err)
	}
	c := &Command{
		name:    name,
		args:    args,
		env:     env,
		stdoutr: stdoutr,
		stdoutw: stdoutw,
		stderrr: stderrr,
		stderrw: stderrw,
	}

	return c, nil
}

func (c *Command) Start(policy RespawnPolicy) error {
	signal := false
	forever := false
	if policy.Times == 0 {
		forever = true
	}
	times := policy.Times
	c.exited.Add(1)

	go func() {
		for !signal && (forever || times > 0) {
			cmd := exec.Command(c.name, c.args...)
			cmd.Env = c.env

			stdout, err := cmd.StdoutPipe()
			if err != nil {
				panic(err) // TODO:
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				panic(err) // TODO:
			}

			go io.CopyBuffer(c.stdoutw, stdout, make([]byte, 4096))
			go io.CopyBuffer(c.stderrw, stderr, make([]byte, 4096))

			err = cmd.Start()
			if err != nil {
				// TODO: Return with Wait()?
				panic(err)
			}
			c.cmdMu.Lock()
			c.cmd = cmd
			c.cmdMu.Unlock()
			err = cmd.Wait()
			if err == nil {
				break
			}
			c.cmdMu.Lock()
			c.cmd = nil
			signal = c.signal
			c.cmdMu.Unlock()

			if times > 0 {
				times--
			}
		}
		c.exited.Done()
	}()

	return nil
}

func (c *Command) RelaySignals(signals <-chan os.Signal) {
	go func() {
		for s := range signals {
			c.cmdMu.Lock()
			if c.cmd != nil {
				c.cmd.Process.Signal(s)
			}
			c.cmdMu.Unlock()
		}
	}()
}

func (c *Command) Kill() error {
	c.cmdMu.Lock()
	if c.cmd != nil {
		c.cmd.Process.Kill()
	}
	c.signal = true
	c.cmdMu.Unlock()

	return nil
}

func (c *Command) Wait() error {
	c.exited.Wait()

	c.stdoutw.Close()
	c.stdoutr.Close()
	c.stderrw.Close()
	c.stderrr.Close()

	// TODO: It would be great if we can catch errors like command not
	//       found and do not respawn in this case but only on non-zero
	//       command exit. And return any final iteration error here.
	return nil
}

func (c *Command) StdOut() io.Reader {
	return c.stdoutr
}

func (c *Command) StdErr() io.Reader {
	return c.stderrr
}
