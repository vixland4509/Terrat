package autosuggest

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var defaultCommonCommands = []string{
	"git status",
	"git commit -m \"\"",
	"git push origin main",
	"git pull --rebase",
	"git log --oneline -n 10",
	"git diff",
	"git checkout",
	"git branch -a",
	"docker ps -a",
	"docker compose up -d",
	"docker compose down",
	"docker logs -f",
	"make build",
	"make install",
	"make clean",
	"make run",
	"make test",
	"systemctl status",
	"systemctl restart",
	"journalctl -xeu",
	"sudo apt update && sudo apt upgrade",
	"sudo apt install -y",
	"sudo pacman -Syu",
	"go test ./...",
	"go run main.go",
	"go build -o",
	"cargo build --release",
	"cargo run",
	"npm run dev",
	"npm run build",
	"pnpm dev",
	"yarn start",
	"cat /etc/os-release",
	"uname -r",
	"ls -lah",
	"mkdir -p",
	"rm -rf",
	"grep -rn",
	"find . -name",
	"tar -czvf",
	"tar -xzvf",
	"htop",
	"btop",
	"neofetch",
	"fastfetch",
}

type Engine struct {
	mu       sync.RWMutex
	history  []string
	historyM map[string]struct{}
	maxItems int
}

func NewEngine() *Engine {
	e := &Engine{
		historyM: make(map[string]struct{}),
		maxItems: 2000,
	}

	for _, cmd := range defaultCommonCommands {
		e.addCommandInternal(cmd)
	}

	e.loadShellHistory()
	return e
}

func (e *Engine) loadShellHistory() {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return
	}

	historyFiles := []string{
		filepath.Join(home, ".bash_history"),
		filepath.Join(home, ".zsh_history"),
	}

	for _, hf := range historyFiles {
		file, err := os.Open(hf)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		var lines []string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			// Handle zsh extended history format: ": 1600000000:0;cmd"
			if strings.HasPrefix(line, ": ") {
				if semi := strings.Index(line, ";"); semi != -1 && semi+1 < len(line) {
					line = strings.TrimSpace(line[semi+1:])
				}
			}
			if len(line) >= 2 {
				lines = append(lines, line)
			}
		}
		_ = file.Close()

		// Read in reverse (most recent commands first)
		for i := len(lines) - 1; i >= 0; i-- {
			e.addCommandInternal(lines[i])
			if len(e.history) >= e.maxItems {
				break
			}
		}
	}
}

func (e *Engine) addCommandInternal(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || len(cmd) < 2 {
		return
	}
	if _, exists := e.historyM[cmd]; exists {
		return
	}
	e.historyM[cmd] = struct{}{}
	e.history = append(e.history, cmd)
}

// Add learns a new command executed by the user.
func (e *Engine) Add(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" || len(cmd) < 2 {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Prioritize this command by placing it at the start
	if _, exists := e.historyM[cmd]; exists {
		for i, h := range e.history {
			if h == cmd {
				e.history = append(e.history[:i], e.history[i+1:]...)
				break
			}
		}
	} else {
		e.historyM[cmd] = struct{}{}
		if len(e.history) >= e.maxItems {
			last := e.history[len(e.history)-1]
			delete(e.historyM, last)
			e.history = e.history[:len(e.history)-1]
		}
	}

	e.history = append([]string{cmd}, e.history...)
}

// Suggest returns the ghost suffix (continuation) for the given input prefix.
// If input is "git com", and best match is "git commit -m", suffix is "mit -m".
func (e *Engine) Suggest(input string) string {
	if len(input) < 1 {
		return ""
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	lowerInput := strings.ToLower(input)

	for _, cmd := range e.history {
		lowerCmd := strings.ToLower(cmd)
		if strings.HasPrefix(lowerCmd, lowerInput) && len(cmd) > len(input) {
			// Return suffix preserving original case of the match
			return cmd[len(input):]
		}
	}

	return ""
}
