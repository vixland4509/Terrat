package pty

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	defaultShellMu sync.RWMutex
	defaultShell   string
)

func SetDefaultShell(shell string) {
	defaultShellMu.Lock()
	defer defaultShellMu.Unlock()
	defaultShell = shell
}

func GetDefaultShell() string {
	defaultShellMu.RLock()
	defer defaultShellMu.RUnlock()
	return defaultShell
}

type WindowsShellDetector struct {
	Getenv   func(string) string
	Stat     func(string) (os.FileInfo, error)
	LookPath func(string) (string, error)
	Stderr   io.Writer
}

func DefaultWindowsShellDetector() *WindowsShellDetector {
	return &WindowsShellDetector{
		Getenv:   os.Getenv,
		Stat:     os.Stat,
		LookPath: exec.LookPath,
		Stderr:   os.Stderr,
	}
}

func cleanBaseName(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = strings.ReplaceAll(p, "\\", "/")
	return filepath.Base(p)
}

func ExtractShellName(exe string) string {
	exe = strings.TrimSpace(exe)
	if exe == "" {
		return "bash"
	}
	base := strings.ToLower(cleanBaseName(exe))
	base = strings.TrimSuffix(base, ".exe")
	if base == "" || base == "." {
		return "bash"
	}
	return base
}

func isBash(exe string) bool {
	base := strings.ToLower(cleanBaseName(exe))
	base = strings.TrimSuffix(base, ".exe")
	return base == "bash" || base == "git-bash" || base == "sh"
}

func isPowerShell(exe string) bool {
	base := strings.ToLower(cleanBaseName(exe))
	base = strings.TrimSuffix(base, ".exe")
	return base == "powershell" || base == "pwsh"
}

func formatBashArgs(cmd []string) []string {
	if len(cmd) == 0 {
		return cmd
	}
	hasLogin := false
	hasInteractive := false
	for _, arg := range cmd[1:] {
		if arg == "-l" || arg == "--login" {
			hasLogin = true
		}
		if arg == "-i" {
			hasInteractive = true
		}
		if arg == "-c" {
			return cmd
		}
	}
	res := append([]string{}, cmd...)
	if !hasLogin {
		res = append(res, "-l")
	}
	if !hasInteractive {
		res = append(res, "-i")
	}
	return res
}

func formatPowerShellArgs(cmd []string) []string {
	if len(cmd) == 0 {
		return cmd
	}
	hasNoLogo := false
	for _, arg := range cmd[1:] {
		if strings.EqualFold(arg, "-NoLogo") {
			hasNoLogo = true
			break
		}
	}
	res := append([]string{}, cmd...)
	if !hasNoLogo {
		res = append(res, "-NoLogo")
	}
	return res
}

func parseCommandLine(cmdStr string, stat func(string) (os.FileInfo, error)) []string {
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return nil
	}

	if stat != nil {
		if fi, err := stat(cmdStr); err == nil && !fi.IsDir() {
			return []string{cmdStr}
		}
	}

	var parts []string
	var cur strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(cmdStr); i++ {
		ch := cmdStr[i]
		if inQuotes {
			if ch == quoteChar {
				inQuotes = false
			} else {
				cur.WriteByte(ch)
			}
		} else {
			if ch == '"' || ch == '\'' {
				inQuotes = true
				quoteChar = ch
			} else if ch == ' ' || ch == '\t' {
				if cur.Len() > 0 {
					parts = append(parts, cur.String())
					cur.Reset()
				}
			} else {
				cur.WriteByte(ch)
			}
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

func findGitBash(d *WindowsShellDetector) (string, bool) {
	var candidates []string

	if pf := d.Getenv("ProgramFiles"); pf != "" {
		candidates = append(candidates, filepath.Join(pf, "Git", "bin", "bash.exe"))
	}
	if pf86 := d.Getenv("ProgramFiles(x86)"); pf86 != "" {
		candidates = append(candidates, filepath.Join(pf86, "Git", "bin", "bash.exe"))
	}
	if pw64 := d.Getenv("ProgramW6432"); pw64 != "" {
		candidates = append(candidates, filepath.Join(pw64, "Git", "bin", "bash.exe"))
	}
	if localAppData := d.Getenv("LocalAppData"); localAppData != "" {
		candidates = append(candidates, filepath.Join(localAppData, "Programs", "Git", "bin", "bash.exe"))
	}
	candidates = append(candidates,
		`C:\Program Files\Git\bin\bash.exe`,
		`C:\Program Files (x86)\Git\bin\bash.exe`,
		`C:\Git\bin\bash.exe`,
	)

	for _, p := range candidates {
		if p == "" {
			continue
		}
		if fi, err := d.Stat(p); err == nil && !fi.IsDir() {
			return p, true
		}
	}

	if p, err := d.LookPath("bash.exe"); err == nil && p != "" {
		return p, true
	}
	if p, err := d.LookPath("bash"); err == nil && p != "" {
		return p, true
	}

	return "", false
}

func findWSL(d *WindowsShellDetector) (string, bool) {
	if p, err := d.LookPath("wsl.exe"); err == nil && p != "" {
		return p, true
	}
	if p, err := d.LookPath("wsl"); err == nil && p != "" {
		return p, true
	}

	var candidates []string
	if sysRoot := d.Getenv("SystemRoot"); sysRoot != "" {
		candidates = append(candidates, filepath.Join(sysRoot, "System32", "wsl.exe"))
	}
	if winDir := d.Getenv("windir"); winDir != "" {
		candidates = append(candidates, filepath.Join(winDir, "System32", "wsl.exe"))
	}
	candidates = append(candidates,
		`C:\Windows\System32\wsl.exe`,
		`C:\Windows\Sysnative\wsl.exe`,
	)

	for _, p := range candidates {
		if p == "" {
			continue
		}
		if fi, err := d.Stat(p); err == nil && !fi.IsDir() {
			return p, true
		}
	}

	return "", false
}

func findFallbackShell(d *WindowsShellDetector) (string, []string) {
	if p, err := d.LookPath("powershell.exe"); err == nil && p != "" {
		return "PowerShell", formatPowerShellArgs([]string{p})
	}
	var psCandidates []string
	if sysRoot := d.Getenv("SystemRoot"); sysRoot != "" {
		psCandidates = append(psCandidates, filepath.Join(sysRoot, "System32", "WindowsPowerShell", "v1.0", "powershell.exe"))
	}
	psCandidates = append(psCandidates, `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`)
	for _, p := range psCandidates {
		if fi, err := d.Stat(p); err == nil && !fi.IsDir() {
			return "PowerShell", formatPowerShellArgs([]string{p})
		}
	}

	if p, err := d.LookPath("cmd.exe"); err == nil && p != "" {
		return "cmd.exe", []string{p}
	}
	var cmdCandidates []string
	if sysRoot := d.Getenv("SystemRoot"); sysRoot != "" {
		cmdCandidates = append(cmdCandidates, filepath.Join(sysRoot, "System32", "cmd.exe"))
	}
	cmdCandidates = append(cmdCandidates, `C:\Windows\System32\cmd.exe`)
	for _, p := range cmdCandidates {
		if fi, err := d.Stat(p); err == nil && !fi.IsDir() {
			return "cmd.exe", []string{p}
		}
	}

	return "powershell.exe", []string{"powershell.exe", "-NoLogo"}
}

func resolveWindowsBashPreference(d *WindowsShellDetector) []string {
	if gitBash, found := findGitBash(d); found {
		return formatBashArgs([]string{gitBash})
	}

	if wsl, found := findWSL(d); found {
		return []string{wsl}
	}

	fallbackName, fallbackCmd := findFallbackShell(d)
	fmt.Fprintf(d.Stderr, "terrat: Bash shell requested or preferred, but neither Git Bash nor WSL was found. Falling back to %s\n", fallbackName)
	return fallbackCmd
}

func ResolveWindowsShell(configuredShell string, customCmd ...string) []string {
	return ResolveWindowsShellWithDetector(DefaultWindowsShellDetector(), configuredShell, customCmd...)
}

func isExistingFile(path string, stat func(string) (os.FileInfo, error)) bool {
	if stat == nil {
		return false
	}
	fi, err := stat(path)
	return err == nil && !fi.IsDir()
}

func ResolveWindowsShellWithDetector(d *WindowsShellDetector, configuredShell string, customCmd ...string) []string {
	if d == nil {
		d = DefaultWindowsShellDetector()
	}

	if len(customCmd) > 0 && customCmd[0] != "" {
		if isBash(customCmd[0]) {
			if len(customCmd) == 1 {
				return resolveWindowsBashPreference(d)
			}
			return formatBashArgs(customCmd)
		}
		if isPowerShell(customCmd[0]) {
			return formatPowerShellArgs(customCmd)
		}
		return customCmd
	}

	shellParts := parseCommandLine(configuredShell, d.Stat)
	if len(shellParts) == 0 || isBash(shellParts[0]) {
		if len(shellParts) == 0 {
			return resolveWindowsBashPreference(d)
		}

		var bashCmd []string
		if isExistingFile(shellParts[0], d.Stat) {
			bashCmd = shellParts
		} else {
			baseBash := resolveWindowsBashPreference(d)
			if len(baseBash) == 0 {
				return baseBash
			}
			if len(shellParts) > 1 {
				bashCmd = append([]string{baseBash[0]}, shellParts[1:]...)
			} else {
				return baseBash
			}
		}
		return formatBashArgs(bashCmd)
	}

	exe := shellParts[0]

	if strings.EqualFold(strings.TrimSuffix(cleanBaseName(exe), ".exe"), "wsl") {
		if wslPath, found := findWSL(d); found {
			shellParts[0] = wslPath
		}
		return shellParts
	}

	if isPowerShell(exe) {
		if p, err := d.LookPath(exe); err == nil && p != "" {
			shellParts[0] = p
		}
		return formatPowerShellArgs(shellParts)
	}

	if strings.EqualFold(strings.TrimSuffix(cleanBaseName(exe), ".exe"), "cmd") {
		if p, err := d.LookPath(exe); err == nil && p != "" {
			shellParts[0] = p
		}
		return shellParts
	}

	if fi, err := d.Stat(exe); err == nil && !fi.IsDir() {
		if isBash(exe) {
			return formatBashArgs(shellParts)
		}
		return shellParts
	}
	if p, err := d.LookPath(exe); err == nil && p != "" {
		shellParts[0] = p
		if isBash(p) {
			return formatBashArgs(shellParts)
		}
		return shellParts
	}

	fallbackName, fallbackCmd := findFallbackShell(d)
	fmt.Fprintf(d.Stderr, "terrat: Configured shell %q not found; falling back to %s\n", configuredShell, fallbackName)
	return fallbackCmd
}

func ResolveUnixShell(configuredShell string, customCmd ...string) []string {
	if len(customCmd) > 0 && customCmd[0] != "" {
		return customCmd
	}

	shellParts := parseCommandLine(configuredShell, os.Stat)
	if len(shellParts) > 0 {
		return shellParts
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
		if _, err := os.Stat(shell); err != nil {
			shell = "/bin/sh"
		}
	}
	return []string{shell}
}
