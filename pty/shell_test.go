package pty

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

type mockFileInfo struct {
	name  string
	isDir bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return 1024 }
func (m mockFileInfo) Mode() os.FileMode  { return 0755 }
func (m mockFileInfo) ModTime() time.Time { return time.Now() }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() any           { return nil }

func TestExtractShellName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`C:\Program Files\Git\bin\bash.exe`, "bash"},
		{`C:\Windows\System32\wsl.exe`, "wsl"},
		{`powershell.exe`, "powershell"},
		{`C:\Windows\System32\cmd.exe`, "cmd"},
		{`/bin/zsh`, "zsh"},
		{"", "bash"},
	}

	for _, tc := range tests {
		got := ExtractShellName(tc.input)
		if got != tc.expected {
			t.Errorf("ExtractShellName(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}

func TestResolveWindowsShell_GitBashStandardPath(t *testing.T) {
	var stderr bytes.Buffer
	detector := &WindowsShellDetector{
		Getenv: func(key string) string {
			if key == "ProgramFiles" {
				return `C:\Program Files`
			}
			return ""
		},
		Stat: func(path string) (os.FileInfo, error) {
			if path == `C:\Program Files\Git\bin\bash.exe` {
				return mockFileInfo{name: "bash.exe", isDir: false}, nil
			}
			return nil, os.ErrNotExist
		},
		LookPath: func(file string) (string, error) {
			return "", errors.New("not found")
		},
		Stderr: &stderr,
	}

	cmd := ResolveWindowsShellWithDetector(detector, "")
	if len(cmd) != 3 || cmd[0] != `C:\Program Files\Git\bin\bash.exe` || cmd[1] != "-l" || cmd[2] != "-i" {
		t.Fatalf("unexpected cmd for default preference: %v", cmd)
	}

	cmd2 := ResolveWindowsShellWithDetector(detector, "bash")
	if len(cmd2) != 3 || cmd2[0] != `C:\Program Files\Git\bin\bash.exe` || cmd2[1] != "-l" || cmd2[2] != "-i" {
		t.Fatalf("unexpected cmd for 'bash': %v", cmd2)
	}

	if stderr.Len() > 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestResolveWindowsShell_GitBashInPath(t *testing.T) {
	var stderr bytes.Buffer
	detector := &WindowsShellDetector{
		Getenv: func(key string) string { return "" },
		Stat: func(path string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
		LookPath: func(file string) (string, error) {
			if file == "bash.exe" || file == "bash" {
				return `D:\Tools\Git\bin\bash.exe`, nil
			}
			return "", errors.New("not found")
		},
		Stderr: &stderr,
	}

	cmd := ResolveWindowsShellWithDetector(detector, "bash")
	if len(cmd) != 3 || cmd[0] != `D:\Tools\Git\bin\bash.exe` || cmd[1] != "-l" || cmd[2] != "-i" {
		t.Fatalf("unexpected cmd for bash in PATH: %v", cmd)
	}
}

func TestResolveWindowsShell_WSL(t *testing.T) {
	var stderr bytes.Buffer
	detector := &WindowsShellDetector{
		Getenv: func(key string) string {
			if key == "SystemRoot" {
				return `C:\Windows`
			}
			return ""
		},
		Stat: func(path string) (os.FileInfo, error) {
			if path == `C:\Windows\System32\wsl.exe` {
				return mockFileInfo{name: "wsl.exe", isDir: false}, nil
			}
			return nil, os.ErrNotExist
		},
		LookPath: func(file string) (string, error) {
			return "", errors.New("not found")
		},
		Stderr: &stderr,
	}

	cmd := ResolveWindowsShellWithDetector(detector, "")
	if len(cmd) != 1 || cmd[0] != `C:\Windows\System32\wsl.exe` {
		t.Fatalf("unexpected cmd for WSL auto-detection: %v", cmd)
	}
	if stderr.Len() > 0 {
		t.Fatalf("expected no stderr warning when WSL is found, got %q", stderr.String())
	}
}

func TestResolveWindowsShell_FallbackPowerShell(t *testing.T) {
	var stderr bytes.Buffer
	detector := &WindowsShellDetector{
		Getenv: func(key string) string {
			if key == "SystemRoot" {
				return `C:\Windows`
			}
			return ""
		},
		Stat: func(path string) (os.FileInfo, error) {
			if path == `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe` {
				return mockFileInfo{name: "powershell.exe", isDir: false}, nil
			}
			return nil, os.ErrNotExist
		},
		LookPath: func(file string) (string, error) {
			return "", errors.New("not found")
		},
		Stderr: &stderr,
	}

	cmd := ResolveWindowsShellWithDetector(detector, "")
	if len(cmd) != 2 || cmd[0] != `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe` || cmd[1] != "-NoLogo" {
		t.Fatalf("unexpected fallback cmd: %v", cmd)
	}

	if !strings.Contains(stderr.String(), "Bash shell requested or preferred, but neither Git Bash nor WSL was found") {
		t.Fatalf("expected descriptive warning in stderr, got %q", stderr.String())
	}
}

func TestResolveWindowsShell_FallbackCMD(t *testing.T) {
	var stderr bytes.Buffer
	detector := &WindowsShellDetector{
		Getenv: func(key string) string {
			if key == "SystemRoot" {
				return `C:\Windows`
			}
			return ""
		},
		Stat: func(path string) (os.FileInfo, error) {
			if path == `C:\Windows\System32\cmd.exe` {
				return mockFileInfo{name: "cmd.exe", isDir: false}, nil
			}
			return nil, os.ErrNotExist
		},
		LookPath: func(file string) (string, error) {
			return "", errors.New("not found")
		},
		Stderr: &stderr,
	}

	cmd := ResolveWindowsShellWithDetector(detector, "bash")
	if len(cmd) != 1 || cmd[0] != `C:\Windows\System32\cmd.exe` {
		t.Fatalf("unexpected fallback cmd: %v", cmd)
	}
	if !strings.Contains(stderr.String(), "Falling back to cmd.exe") {
		t.Fatalf("expected cmd fallback warning in stderr, got %q", stderr.String())
	}
}

func TestResolveWindowsShell_ExplicitCustomShells(t *testing.T) {
	detector := &WindowsShellDetector{
		Getenv: func(key string) string { return "" },
		Stat: func(path string) (os.FileInfo, error) {
			if path == `C:\Program Files\Git\bin\bash.exe` {
				return mockFileInfo{name: "bash.exe", isDir: false}, nil
			}
			return nil, os.ErrNotExist
		},
		LookPath: func(file string) (string, error) {
			if file == "pwsh.exe" || file == "pwsh" {
				return `C:\Program Files\PowerShell\7\pwsh.exe`, nil
			}
			if file == "wsl.exe" {
				return `C:\Windows\System32\wsl.exe`, nil
			}
			return "", errors.New("not found")
		},
		Stderr: &bytes.Buffer{},
	}

	cmdPS := ResolveWindowsShellWithDetector(detector, "pwsh.exe")
	if len(cmdPS) != 2 || cmdPS[0] != `C:\Program Files\PowerShell\7\pwsh.exe` || cmdPS[1] != "-NoLogo" {
		t.Fatalf("unexpected PowerShell command: %v", cmdPS)
	}

	cmdWSL := ResolveWindowsShellWithDetector(detector, "wsl.exe")
	if len(cmdWSL) != 1 || cmdWSL[0] != `C:\Windows\System32\wsl.exe` {
		t.Fatalf("unexpected WSL command: %v", cmdWSL)
	}

	cmdGitBash := ResolveWindowsShellWithDetector(detector, `C:\Program Files\Git\bin\bash.exe`)
	if len(cmdGitBash) != 3 || cmdGitBash[0] != `C:\Program Files\Git\bin\bash.exe` || cmdGitBash[1] != "-l" || cmdGitBash[2] != "-i" {
		t.Fatalf("unexpected Git Bash path command: %v", cmdGitBash)
	}

	cmdCustom := ResolveWindowsShellWithDetector(detector, `"C:\Program Files\Git\bin\bash.exe" -l`)
	if len(cmdCustom) != 3 || cmdCustom[0] != `C:\Program Files\Git\bin\bash.exe` || cmdCustom[1] != "-l" || cmdCustom[2] != "-i" {
		t.Fatalf("unexpected custom bash with -l command: %v", cmdCustom)
	}
}

func TestResolveWindowsShell_CustomCmdWithFlags(t *testing.T) {
	detector := &WindowsShellDetector{
		Getenv: func(key string) string { return "" },
		Stat: func(path string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		},
		LookPath: func(file string) (string, error) {
			return file, nil
		},
		Stderr: &bytes.Buffer{},
	}

	cmd := ResolveWindowsShellWithDetector(detector, "", "bash", "-c", "echo hello")
	if len(cmd) != 3 || cmd[0] != "bash" || cmd[1] != "-c" || cmd[2] != "echo hello" {
		t.Fatalf("unexpected cmd with -c: %v", cmd)
	}
}

func TestSetAndGetDefaultShell(t *testing.T) {
	orig := GetDefaultShell()
	defer SetDefaultShell(orig)

	SetDefaultShell("bash")
	if GetDefaultShell() != "bash" {
		t.Fatalf("expected 'bash', got %q", GetDefaultShell())
	}

	SetDefaultShell(`C:\Program Files\Git\bin\bash.exe`)
	if GetDefaultShell() != `C:\Program Files\Git\bin\bash.exe` {
		t.Fatalf("expected Git Bash path, got %q", GetDefaultShell())
	}
}

func TestResolveUnixShell(t *testing.T) {
	cmd := ResolveUnixShell("zsh")
	if len(cmd) != 1 || cmd[0] != "zsh" {
		t.Fatalf("expected ['zsh'], got %v", cmd)
	}

	cmd2 := ResolveUnixShell("zsh", "htop")
	if len(cmd2) != 1 || cmd2[0] != "htop" {
		t.Fatalf("expected ['htop'], got %v", cmd2)
	}
}
