package diagnostics

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	shellAliases sync.Map
	gitAliases   sync.Map
	aliasesOnce  sync.Once
)

func RegisterAlias(name, expansion string) {
	name = strings.TrimSpace(name)
	if name != "" {
		shellAliases.Store(name, expansion)
	}
}

func RegisterAliasFromLine(line string) {
	parseAliasLine(line)
}

func IsAlias(name string) bool {
	InitAliases()
	_, ok := shellAliases.Load(name)
	return ok
}

func RegisterGitAlias(name string) {
	name = strings.TrimSpace(name)
	if name != "" {
		gitAliases.Store(name, true)
	}
}

func IsGitAlias(sub string) bool {
	InitAliases()
	_, ok := gitAliases.Load(sub)
	return ok
}

func InitAliases() {
	aliasesOnce.Do(func() {
		loadAliasesFromFiles()
		loadGitAliases()
		go probeShellAliases()
	})
}

func parseAliasLine(line string) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "alias ") {
		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "alias "))
		if eqIdx := strings.IndexByte(rest, '='); eqIdx > 0 {
			name := strings.TrimSpace(rest[:eqIdx])
			val := strings.TrimSpace(rest[eqIdx+1:])
			val = strings.Trim(val, `'"`)
			RegisterAlias(name, val)
		} else if spIdx := strings.IndexByte(rest, ' '); spIdx > 0 {
			name := strings.TrimSpace(rest[:spIdx])
			val := strings.TrimSpace(rest[spIdx+1:])
			val = strings.Trim(val, `'"`)
			RegisterAlias(name, val)
		}
	} else if strings.HasPrefix(trimmed, "abbr ") {
		parts := strings.Fields(trimmed)
		if len(parts) >= 3 {
			if parts[1] == "-a" || parts[1] == "--add" {
				if len(parts) >= 4 {
					RegisterAlias(parts[2], parts[3])
				}
			} else if !strings.HasPrefix(parts[1], "-") {
				RegisterAlias(parts[1], parts[2])
			}
		}
	}
}

func loadAliasesFromFiles() {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return
	}

	filesToScan := []string{
		filepath.Join(home, ".bash_aliases"),
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".config", "fish", "config.fish"),
	}

	for _, p := range filesToScan {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			parseAliasLine(scanner.Text())
		}
		_ = f.Close()
	}
}

func loadGitAliases() {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		gitConfigFile := filepath.Join(home, ".gitconfig")
		if f, err := os.Open(gitConfigFile); err == nil {
			scanner := bufio.NewScanner(f)
			inAliasSection := false
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "[") {
					inAliasSection = strings.EqualFold(line, "[alias]")
					continue
				}
				if inAliasSection && strings.Contains(line, "=") {
					parts := strings.SplitN(line, "=", 2)
					aliasName := strings.TrimSpace(parts[0])
					if aliasName != "" {
						RegisterGitAlias(aliasName)
					}
				}
			}
			_ = f.Close()
		}
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
		defer cancel()
		cmd := exec.CommandContext(ctx, "git", "config", "--get-regexp", `^alias\.`)
		out, err := cmd.Output()
		if err == nil {
			scanner := bufio.NewScanner(strings.NewReader(string(out)))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(line, "alias.") {
					rest := strings.TrimPrefix(line, "alias.")
					parts := strings.Fields(rest)
					if len(parts) > 0 {
						RegisterGitAlias(parts[0])
					}
				}
			}
		}
	}()
}

func probeShellAliases() {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	var cmd *exec.Cmd
	baseShell := filepath.Base(shell)
	if baseShell == "fish" {
		cmd = exec.CommandContext(ctx, shell, "-c", "alias; abbr --show")
	} else {
		cmd = exec.CommandContext(ctx, shell, "-i", "-c", "alias")
	}

	out, err := cmd.Output()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		parseAliasLine(scanner.Text())
	}
}
