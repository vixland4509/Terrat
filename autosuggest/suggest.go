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
// If cwd is provided, it prioritizes local file/directory path completion (including paths with spaces).
// Otherwise or as fallback, it searches command history.
func (e *Engine) Suggest(input string, cwd ...string) string {
	if len(input) < 1 {
		return ""
	}

	// 1. Try local filesystem path completion if cwd is provided
	if len(cwd) > 0 && cwd[0] != "" {
		if pathSuffix := suggestPath(input, cwd[0]); pathSuffix != "" {
			return pathSuffix
		}
	}

	// 2. Fall back to command history
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

// suggestPath checks the active directory for matching files or folders (supporting spaces and quotes)
func suggestPath(input string, cwd string) string {
	if cwd == "" || len(input) == 0 {
		return ""
	}

	// Check if input looks like a command with arguments or a path
	inDouble := false
	inSingle := false
	escaped := false
	lastTokenStart := 0

	for i := 0; i < len(input); i++ {
		b := input[i]
		if escaped {
			escaped = false
			continue
		}
		if b == '\\' && !inSingle {
			escaped = true
			continue
		}
		if b == '"' && !inSingle {
			inDouble = !inDouble
			if inDouble {
				lastTokenStart = i
			}
			continue
		}
		if b == '\'' && !inDouble {
			inSingle = !inSingle
			if inSingle {
				lastTokenStart = i
			}
			continue
		}
		if (b == ' ' || b == '\t') && !inDouble && !inSingle {
			lastTokenStart = i + 1
		}
	}

	rawToken := input[lastTokenStart:]

	// Determine quote mode
	isQuoted := false
	quoteChar := byte(0)
	cleanToken := rawToken

	if len(rawToken) > 0 {
		if rawToken[0] == '"' {
			isQuoted = true
			quoteChar = '"'
			cleanToken = rawToken[1:]
		} else if rawToken[0] == '\'' {
			isQuoted = true
			quoteChar = '\''
			cleanToken = rawToken[1:]
		}
	}

	// Unescape if unquoted (e.g. "my\ " -> "my ")
	var unescapedToken strings.Builder
	for i := 0; i < len(cleanToken); i++ {
		if cleanToken[i] == '\\' && !isQuoted && i+1 < len(cleanToken) {
			i++
			unescapedToken.WriteByte(cleanToken[i])
		} else {
			unescapedToken.WriteByte(cleanToken[i])
		}
	}
	pathToSearch := unescapedToken.String()

	// Split directory and base
	dirPart := filepath.Dir(pathToSearch)
	basePart := filepath.Base(pathToSearch)

	searchDir := cwd
	if filepath.IsAbs(pathToSearch) {
		if dirPart == "/" {
			searchDir = "/"
		} else {
			searchDir = dirPart
		}
	} else if strings.HasPrefix(pathToSearch, "~") {
		home, _ := os.UserHomeDir()
		if len(pathToSearch) == 1 {
			searchDir = home
			basePart = ""
		} else {
			rem := pathToSearch[2:]
			d := filepath.Dir(rem)
			b := filepath.Base(rem)
			searchDir = filepath.Join(home, d)
			basePart = b
		}
	} else if strings.Contains(pathToSearch, "/") {
		searchDir = filepath.Join(cwd, dirPart)
	} else {
		searchDir = cwd
		basePart = pathToSearch
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return ""
	}

	lowerBase := strings.ToLower(basePart)

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(basePart, ".") {
			continue // Skip hidden files unless explicitly requested
		}

		if len(basePart) > 0 && strings.HasPrefix(strings.ToLower(name), lowerBase) && len(name) >= len(basePart) {
			suffix := name[len(basePart):]
			if entry.IsDir() {
				suffix += "/"
			}

			if isQuoted {
				if entry.IsDir() {
					return suffix
				}
				return suffix + string(quoteChar)
			}

			// In unquoted mode, spaces in suffix must be escaped
			escapedSuffix := strings.ReplaceAll(suffix, " ", "\\ ")
			return escapedSuffix
		}
	}

	// Also check if the user typed an unescaped space at the end, e.g. "cd my "
	// where the folder name contains a space ("my space folder")
	if !isQuoted && len(rawToken) == 0 && lastTokenStart > 1 && input[lastTokenStart-1] == ' ' {
		prevTokenStart := strings.LastIndexAny(strings.TrimRight(input[:lastTokenStart-1], " \t"), " \t")
		var prevToken string
		if prevTokenStart == -1 {
			prevToken = strings.TrimSpace(input[:lastTokenStart-1])
		} else {
			prevToken = strings.TrimSpace(input[prevTokenStart+1 : lastTokenStart-1])
		}
		if len(prevToken) > 0 {
			targetPrefix := strings.ToLower(prevToken + " ")
			for _, entry := range entries {
				name := entry.Name()
				if strings.HasPrefix(strings.ToLower(name), targetPrefix) {
					suffix := name[len(targetPrefix):]
					if entry.IsDir() {
						suffix += "/"
					}
					escapedSuffix := strings.ReplaceAll(suffix, " ", "\\ ")
					return escapedSuffix
				}
			}
		}
	}

	return ""
}

