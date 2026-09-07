//go:build !windows

package pty

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

type TerminalPTY struct {
	file *os.File
	cmd  *exec.Cmd
}

func Start(cols, rows, pixelWidth, pixelHeight uint16, customCmd ...string) (*TerminalPTY, error) {
	var cmd *exec.Cmd
	if len(customCmd) > 0 && customCmd[0] != "" {
		cmd = exec.Command(customCmd[0], customCmd[1:]...)
	} else {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
			if _, err := os.Stat(shell); err != nil {
				shell = "/bin/sh"
			}
		}
		cmd = exec.Command(shell)
	}

	env := os.Environ()
	customEnv := []string{
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"TERRAT_TERMINAL=1",
	}
	cmd.Env = append(env, customEnv...)

	ws := &pty.Winsize{
		Rows: rows,
		Cols: cols,
		X:    pixelWidth,
		Y:    pixelHeight,
	}

	f, err := pty.StartWithSize(cmd, ws)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}

	return &TerminalPTY{
		file: f,
		cmd:  cmd,
	}, nil
}

func (p *TerminalPTY) Read(b []byte) (int, error) {
	return p.file.Read(b)
}

func (p *TerminalPTY) Write(b []byte) (int, error) {
	return p.file.Write(b)
}

func (p *TerminalPTY) Resize(cols, rows, pixelWidth, pixelHeight uint16) error {
	ws := &pty.Winsize{
		Rows: rows,
		Cols: cols,
		X:    pixelWidth,
		Y:    pixelHeight,
	}
	return pty.Setsize(p.file, ws)
}

func (p *TerminalPTY) Close() error {
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Signal(syscall.SIGHUP)
	}
	return p.file.Close()
}

func (p *TerminalPTY) Wait() (*os.ProcessState, error) {
	if p.cmd == nil {
		return nil, nil
	}
	return p.cmd.ProcessState, p.cmd.Wait()
}
