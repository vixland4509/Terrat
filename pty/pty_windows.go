//go:build windows

package pty

import (
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type TerminalPTY struct {
	hPC        windows.Handle
	hProcess   windows.Handle
	hThread    windows.Handle
	processId  uint32
	stdinPipe  *os.File
	stdoutPipe *os.File
	closeOnce  sync.Once
	shellName  string
}

func Start(cols, rows, pixelWidth, pixelHeight uint16, customCmd ...string) (*TerminalPTY, error) {
	if os.Getenv("TERM") == "" {
		_ = os.Setenv("TERM", "xterm-256color")
	}
	if os.Getenv("COLORTERM") == "" {
		_ = os.Setenv("COLORTERM", "truecolor")
	}
	_ = os.Setenv("TERRAT_TERMINAL", "1")

	var inRead, inWrite windows.Handle
	var outRead, outWrite windows.Handle

	sa := windows.SecurityAttributes{
		Length:        uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		InheritHandle: 1,
	}

	if err := windows.CreatePipe(&inRead, &inWrite, &sa, 0); err != nil {
		return nil, fmt.Errorf("failed to create input pipe: %w", err)
	}

	if err := windows.CreatePipe(&outRead, &outWrite, &sa, 0); err != nil {
		windows.CloseHandle(inRead)
		windows.CloseHandle(inWrite)
		return nil, fmt.Errorf("failed to create output pipe: %w", err)
	}

	coord := windows.Coord{
		X: int16(cols),
		Y: int16(rows),
	}
	var hPC windows.Handle
	err := windows.CreatePseudoConsole(coord, inRead, outWrite, 0, &hPC)
	if err != nil {
		windows.CloseHandle(inRead)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		windows.CloseHandle(outWrite)
		return nil, fmt.Errorf("failed to create pseudo console: %w", err)
	}

	windows.CloseHandle(inRead)
	windows.CloseHandle(outWrite)

	attrList, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		windows.ClosePseudoConsole(hPC)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return nil, fmt.Errorf("failed to create proc thread attribute list: %w", err)
	}
	defer attrList.Delete()

	const PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE = 0x00020016
	err = attrList.Update(
		PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		unsafe.Pointer(&hPC),
		unsafe.Sizeof(hPC),
	)
	if err != nil {
		windows.ClosePseudoConsole(hPC)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return nil, fmt.Errorf("failed to update proc thread attribute list: %w", err)
	}

	var si windows.StartupInfoEx
	si.Cb = uint32(unsafe.Sizeof(si))
	si.ProcThreadAttributeList = attrList.List()

	cmdArgs := ResolveWindowsShell(GetDefaultShell(), customCmd...)
	if len(cmdArgs) == 0 {
		cmdArgs = []string{"powershell.exe", "-NoLogo"}
	}

	cmdLine := syscall.EscapeArg(cmdArgs[0])
	for _, arg := range cmdArgs[1:] {
		cmdLine += " " + syscall.EscapeArg(arg)
	}

	cmdLineUTF16, err := windows.UTF16PtrFromString(cmdLine)
	if err != nil {
		windows.ClosePseudoConsole(hPC)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return nil, err
	}

	var pi windows.ProcessInformation
	creationFlags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT)

	err = windows.CreateProcess(
		nil,
		cmdLineUTF16,
		nil,
		nil,
		false,
		creationFlags,
		nil,
		nil,
		&si.StartupInfo,
		&pi,
	)
	if err != nil {
		windows.ClosePseudoConsole(hPC)
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return nil, fmt.Errorf("failed to start process %q: %w", cmdLine, err)
	}

	return &TerminalPTY{
		hPC:        hPC,
		hProcess:   pi.Process,
		hThread:    pi.Thread,
		processId:  pi.ProcessId,
		stdinPipe:  os.NewFile(uintptr(inWrite), "conpty-stdin"),
		stdoutPipe: os.NewFile(uintptr(outRead), "conpty-stdout"),
		shellName:  ExtractShellName(cmdArgs[0]),
	}, nil
}

func (p *TerminalPTY) Read(b []byte) (int, error) {
	return p.stdoutPipe.Read(b)
}

func (p *TerminalPTY) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	clean := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c == '\x7f' {
			clean = append(clean, '\x08')
		} else if c == '\r' {
			clean = append(clean, '\r')
			if i+1 < len(b) && b[i+1] == '\n' {
				i++
			}
		} else if c == '\n' {
			clean = append(clean, '\r')
		} else {
			clean = append(clean, c)
		}
	}
	return p.stdinPipe.Write(clean)
}

func (p *TerminalPTY) Resize(cols, rows, pixelWidth, pixelHeight uint16) error {
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	coord := windows.Coord{
		X: int16(cols),
		Y: int16(rows),
	}
	return windows.ResizePseudoConsole(p.hPC, coord)
}

func (p *TerminalPTY) Close() error {
	var closeErr error
	p.closeOnce.Do(func() {
		if p.hPC != 0 {
			windows.ClosePseudoConsole(p.hPC)
		}
		if p.stdinPipe != nil {
			_ = p.stdinPipe.Close()
		}
		if p.stdoutPipe != nil {
			_ = p.stdoutPipe.Close()
		}
		if p.hProcess != 0 {
			_ = windows.TerminateProcess(p.hProcess, 0)
			windows.CloseHandle(p.hProcess)
		}
		if p.hThread != 0 {
			windows.CloseHandle(p.hThread)
		}
	})
	return closeErr
}

func (p *TerminalPTY) Wait() (*os.ProcessState, error) {
	if p.hProcess == 0 {
		return nil, nil
	}
	_, err := windows.WaitForSingleObject(p.hProcess, windows.INFINITE)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (p *TerminalPTY) IsForegroundShell() bool {
	return true
}

func (p *TerminalPTY) GetCwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func (p *TerminalPTY) ShellName() string {
	return p.shellName
}

var _ io.ReadWriteCloser = (*TerminalPTY)(nil)


