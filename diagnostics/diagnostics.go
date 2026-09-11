package diagnostics

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// IssueSeverity indicates the level of the diagnosed issue
type IssueSeverity int

const (
	SeverityNone IssueSeverity = iota
	SeverityHint
	SeverityWarning
	SeverityError
)

type Diagnostic struct {
	Severity   IssueSeverity
	Message    string
	Suggestion string
	QuickFix   string
}

// Thread-safe cache for system binary lookup
var pathCache sync.Map

// isCommandOnPath checks if a command exists in the user's PATH
func isCommandOnPath(name string) bool {
	if len(name) < 2 || len(name) > 40 {
		return false
	}
	for _, r := range name {
		if r > 127 || r == '/' || r == '\\' || r == ':' || r <= 32 {
			return false
		}
	}
	if v, ok := pathCache.Load(name); ok {
		return v.(bool)
	}
	_, err := exec.LookPath(name)
	exists := (err == nil)
	pathCache.Store(name, exists)
	return exists
}

// Shell built-ins and keywords that don't need PATH resolution
var shellBuiltins = map[string]bool{
	"alias": true, "bg": true, "bind": true, "break": true, "builtin": true,
	"case": true, "cd": true, "command": true, "continue": true, "declare": true,
	"dirs": true, "disown": true, "echo": true, "enable": true, "eval": true,
	"exec": true, "exit": true, "export": true, "fc": true, "fg": true,
	"getopts": true, "hash": true, "help": true, "history": true, "jobs": true,
	"kill": true, "let": true, "local": true, "logout": true, "popd": true,
	"pushd": true, "pwd": true, "read": true, "readonly": true, "return": true,
	"set": true, "shift": true, "shopt": true, "source": true, "suspend": true,
	"test": true, "times": true, "trap": true, "type": true, "typeset": true,
	"ulimit": true, "umask": true, "unalias": true, "unset": true, "wait": true,
	"true": true, "false": true, ":": true, ".": true, "[": true, "[[": true,
	"if": true, "then": true, "else": true, "elif": true, "fi": true,
	"for": true, "while": true, "until": true, "do": true, "done": true,
	"in": true, "esac": true, "select": true, "function": true, "time": true,
}

// Known common Linux commands for typo detection fallback
var commonBaseCommands = []string{
	"cat", "cd", "cp", "curl", "chmod", "chown", "clear",
	"df", "diff", "docker", "du",
	"echo", "exit", "export",
	"find", "free",
	"git", "go", "grep", "gzip",
	"head", "history", "htop",
	"ip",
	"journalctl",
	"kill", "killall",
	"less", "ln", "ls",
	"make", "man", "mkdir", "mv",
	"nano", "nc", "netstat", "npm", "nvim",
	"ping", "pkill", "ps", "pwd", "pnpm",
	"rm", "rmdir",
	"sed", "ssh", "sudo", "systemctl",
	"tail", "tar", "top", "touch", "tree",
	"uname", "uptime",
	"vim",
	"wget", "which", "whoami",
	"yarn",
}

// High-confidence common typos map
var commonTypos = map[string]string{
	"sl":     "ls",
	"gti":    "git",
	"gut":    "git",
	"mkae":   "make",
	"dokcer": "docker",
	"dcoker": "docker",
	"clea":   "clear",
	"claer":  "clear",
	"cLear":  "clear",
	"gerp":   "grep",
	"grpe":   "grep",
	"exist":  "exit",
	"exti":   "exit",
	"qui":    "quit",
	"pyhton": "python",
	"phtyon": "python",
	"shh":    "ssh",
	"whcih":  "which",
	"sudp":   "sudo",
	"sduo":   "sudo",
}

// Known git subcommands (comprehensive official list)
var gitSubcommands = []string{
	"add", "am", "annotate", "apply", "archive", "bisect", "blame", "branch", "bundle",
	"checkout", "cherry", "cherry-pick", "clean", "clone", "commit", "config",
	"describe", "diff", "fetch", "format-patch", "gc", "grep", "help", "init",
	"log", "merge", "mv", "notes", "pull", "push", "range-diff", "rebase", "reflog",
	"remote", "repack", "replace", "reset", "restore", "revert", "rm", "shortlog",
	"show", "sparse-checkout", "stash", "status", "submodule", "switch", "tag",
	"version", "whatchanged", "worktree",
}

// Known docker subcommands
var dockerSubcommands = []string{
	"attach", "build", "builder", "checkpoint", "commit", "compose", "config",
	"container", "context", "cp", "create", "diff", "events", "exec", "export",
	"history", "image", "images", "import", "info", "init", "inspect", "kill",
	"load", "login", "logout", "logs", "manifest", "network", "node", "pause",
	"plugin", "port", "ps", "pull", "push", "rename", "restart", "rm", "rmi",
	"run", "save", "search", "secret", "service", "stack", "start", "stats",
	"stop", "swarm", "system", "tag", "top", "trust", "unpause", "update",
	"version", "volume", "wait",
}

// Known systemctl subcommands
var systemctlSubcommands = []string{
	"cat", "clean", "daemon-reexec", "daemon-reload", "default", "disable",
	"edit", "emergency", "enable", "exit", "halt", "help", "hibernate",
	"hybrid-sleep", "is-active", "is-enabled", "is-failed", "isolate", "kexec",
	"kill", "list-dependencies", "list-jobs", "list-sockets", "list-timers",
	"list-unit-files", "list-units", "mask", "poweroff", "preset", "reboot",
	"reenable", "reload", "reload-or-restart", "rescue", "reset-failed", "restart",
	"set-default", "show", "start", "status", "stop", "suspend", "switch-root",
	"try-restart", "unmask",
}

// Known go subcommands
var goSubcommands = []string{
	"build", "clean", "doc", "env", "fix", "fmt", "generate", "get", "install",
	"list", "mod", "work", "run", "test", "tool", "version", "vet",
}

// Known cargo subcommands
var cargoSubcommands = []string{
	"build", "check", "clean", "doc", "new", "init", "run", "test", "bench",
	"update", "search", "publish", "install", "uninstall", "add", "remove",
	"metadata", "clippy", "fmt", "version",
}

// Known npm subcommands
var npmSubcommands = []string{
	"install", "i", "ci", "test", "run", "start", "build", "init", "publish",
	"audit", "cache", "config", "outdated", "update", "uninstall", "link",
	"list", "pack", "version", "view", "exec",
}

// Commands that almost always require superuser privileges
var privilegedCommands = map[string]string{
	"apt":        "sudo apt",
	"apt-get":    "sudo apt-get",
	"pacman":     "sudo pacman",
	"dnf":        "sudo dnf",
	"yum":        "sudo yum",
	"reboot":     "sudo reboot",
	"poweroff":   "sudo poweroff",
	"shutdown":   "sudo shutdown",
	"useradd":    "sudo useradd",
	"userdel":    "sudo userdel",
	"groupadd":   "sudo groupadd",
	"fdisk":      "sudo fdisk",
	"mkfs":       "sudo mkfs",
	"iptables":   "sudo iptables",
	"ufw":        "sudo ufw",
}

// Read-only operations for privileged commands that DO NOT need root/sudo
var safeReadOperations = map[string][]string{
	"apt":      {"search", "show", "list", "depends", "rdepends", "policy", "help", "--help", "-h", "--version", "-v"},
	"apt-get":  {"help", "--help", "-h", "--version", "-v", "source"},
	"pacman":   {"-ss", "-si", "-q", "-qs", "-qi", "-ql", "-qe", "-qo", "-qu", "-qk", "-qd", "-qdt", "-f", "-fy", "-fs", "-v", "--help", "--version"},
	"dnf":      {"search", "info", "list", "repoquery", "check-update", "help", "--help", "--version"},
	"yum":      {"search", "info", "list", "check-update", "help", "--help", "--version"},
	"ufw":      {"status", "show", "version", "--help"},
	"fdisk":    {"-l", "--list", "-v", "--version"},
	"iptables": {"-l", "-s", "-v", "--list", "-n", "--version"},
}

// LevenshteinDistance computes edit distance between two strings
func LevenshteinDistance(s, t string) int {
	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for j := 1; j <= len(t); j++ {
		for i := 1; i <= len(s); i++ {
			cost := 0
			if s[i-1] != t[j-1] {
				cost = 1
			}
			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}
	return d[len(s)][len(t)]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// FindClosestMatch finds the closest command from list within a threshold
func FindClosestMatch(word string, candidates []string, maxDist int) string {
	lowerWord := strings.ToLower(word)
	bestDist := maxDist + 1
	bestMatch := ""

	for _, cand := range candidates {
		if strings.EqualFold(cand, word) {
			return "" // exact match, not a typo
		}
		dist := LevenshteinDistance(lowerWord, strings.ToLower(cand))
		if dist <= maxDist && dist < bestDist {
			bestDist = dist
			bestMatch = cand
		}
	}
	return bestMatch
}

// isSafeOperation checks if the given arguments represent a read-only subcommand/flag
func isSafeOperation(cmd string, parts []string) bool {
	safeOps, ok := safeReadOperations[cmd]
	if !ok {
		return false
	}
	if len(parts) <= 1 {
		return false
	}
	sub := strings.ToLower(parts[1])
	for _, safe := range safeOps {
		if sub == safe || strings.HasPrefix(sub, safe) {
			return true
		}
	}
	return false
}

// splitCommandPrefix splits input into (prefix, activeSegment).
// prefix preserves all preceding pipeline/chain commands, delimiters (&&, ||, ;, |), and spacing.
func splitCommandPrefix(input string) (prefix string, activeSeg string) {
	delims := []string{"&&", "||", ";", "|"}
	lastIdx := -1
	delimLen := 0

	for _, delim := range delims {
		idx := strings.LastIndex(input, delim)
		if idx > lastIdx {
			lastIdx = idx
			delimLen = len(delim)
		}
	}

	if lastIdx >= 0 && lastIdx+delimLen <= len(input) {
		p := input[:lastIdx+delimLen]
		rem := input[lastIdx+delimLen:]
		trimmedRem := strings.TrimLeft(rem, " \t")
		p += rem[:len(rem)-len(trimmedRem)]
		return p, strings.TrimSpace(trimmedRem)
	}

	trimmed := strings.TrimLeft(input, " \t")
	return input[:len(input)-len(trimmed)], strings.TrimSpace(trimmed)
}

// extractActiveSegment extracts the currently active command segment from pipelines or chains (&&, ||, ;, |)
func extractActiveSegment(input string) string {
	_, seg := splitCommandPrefix(input)
	return seg
}

// tokenizeCommand splits a command string into tokens respecting quotes and backslash escapes
func tokenizeCommand(seg string) []string {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for i := 0; i < len(seg); i++ {
		b := seg[i]
		if escaped {
			cur.WriteByte(b)
			escaped = false
			continue
		}
		if b == '\\' && !inSingle {
			escaped = true
			continue
		}
		if b == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if b == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if (b == ' ' || b == '\t') && !inSingle && !inDouble {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteByte(b)
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// unnestSudo removes sudo and common sudo flags, returning the underlying command
func unnestSudo(parts []string) (cleanedParts []string, hasSudo bool) {
	if len(parts) == 0 || parts[0] != "sudo" {
		return parts, false
	}
	i := 1
	for i < len(parts) {
		arg := parts[i]
		if strings.HasPrefix(arg, "-") {
			// Flags with arguments like -u user
			if (arg == "-u" || arg == "-g" || arg == "-p" || arg == "-c") && i+1 < len(parts) {
				i += 2
				continue
			}
			i++
			continue
		}
		break
	}
	if i < len(parts) {
		return parts[i:], true
	}
	return nil, true
}

// Analyze inspects the user input command and returns any diagnosed issues
func Analyze(rawInput string) *Diagnostic {
	trimmed := strings.TrimSpace(rawInput)
	if len(trimmed) < 2 {
		return nil
	}

	// 1. Syntax Check: Unclosed quotes
	singleQuoteCount := 0
	doubleQuoteCount := 0
	for _, r := range rawInput {
		if r == '\'' {
			singleQuoteCount++
		} else if r == '"' {
			doubleQuoteCount++
		}
	}
	if singleQuoteCount%2 != 0 {
		return &Diagnostic{
			Severity:   SeverityWarning,
			Message:    "Unclosed single quote string (')",
			Suggestion: "Add closing single quote",
			QuickFix:   rawInput + "'",
		}
	}
	if doubleQuoteCount%2 != 0 {
		return &Diagnostic{
			Severity:   SeverityWarning,
			Message:    "Unclosed double quote string (\")",
			Suggestion: "Add closing double quote",
			QuickFix:   rawInput + "\"",
		}
	}

	// Get active segment in case of pipe or chained command
	prefix, activeSeg := splitCommandPrefix(trimmed)
	if len(activeSeg) < 2 {
		return nil
	}

	parts := tokenizeCommand(activeSeg)
	if len(parts) == 0 {
		return nil
	}

	// Handle sudo unnesting
	cmdParts, hasSudo := unnestSudo(parts)
	if len(cmdParts) == 0 {
		return nil
	}

	baseCmd := cmdParts[0]

	// 2. Privileged Command Check (e.g. apt install without sudo)
	// Skip if running as root or already wrapped in sudo
	if !hasSudo && os.Geteuid() != 0 {
		if fix, ok := privilegedCommands[baseCmd]; ok {
			// Don't warn for read-only operations like apt search, pacman -Ss, ufw status
			if !isSafeOperation(baseCmd, cmdParts) {
				fixedLine := fix + strings.TrimPrefix(activeSeg, baseCmd)
				return &Diagnostic{
					Severity:   SeverityWarning,
					Message:    fmt.Sprintf("'%s' usually requires root privileges", baseCmd),
					Suggestion: fmt.Sprintf("Try: %s", fixedLine),
					QuickFix:   prefix + fixedLine,
				}
			}
		}
	}

	// 3. Subcommand Typo Check: git / docker / systemctl / go / cargo / npm
	if len(cmdParts) >= 2 {
		sub := cmdParts[1]
		// Skip flags (-f, --help)
		if !strings.HasPrefix(sub, "-") {
			var subCandidates []string
			switch baseCmd {
			case "git":
				subCandidates = gitSubcommands
			case "docker":
				subCandidates = dockerSubcommands
			case "systemctl":
				subCandidates = systemctlSubcommands
			case "go":
				subCandidates = goSubcommands
			case "cargo":
				subCandidates = cargoSubcommands
			case "npm":
				subCandidates = npmSubcommands
			}

			if len(subCandidates) > 0 {
				isKnown := false
				for _, sc := range subCandidates {
					if sc == sub {
						isKnown = true
						break
					}
				}

				if !isKnown {
					// Check if it's a known git alias (e.g. git deploy, git st, git co)
					if baseCmd == "git" && IsGitAlias(sub) {
						return nil
					}

					// For short subcommands (<= 3 chars like git co, git st, git br), don't flag as error
					// unless it's a known typo like "puch"
					if len(sub) > 3 || sub == "puch" || sub == "stat" || sub == "pul" || sub == "comit" {
						maxD := 1
						if len(sub) >= 6 {
							maxD = 2
						}
						if match := FindClosestMatch(sub, subCandidates, maxD); match != "" {
							fixedLine := strings.Replace(activeSeg, baseCmd+" "+sub, baseCmd+" "+match, 1)
							return &Diagnostic{
								Severity:   SeverityError,
								Message:    fmt.Sprintf("Unknown %s subcommand '%s'", baseCmd, sub),
								Suggestion: fmt.Sprintf("Did you mean '%s'?", match),
								QuickFix:   prefix + fixedLine,
							}
						}
					}
				}
			}
		}
	}

	// 4. Base Command Validation & Typo Detection
	// Skip comments
	if strings.HasPrefix(baseCmd, "#") {
		return nil
	}

	// Skip paths (./script, /usr/bin/foo, ~/bin/bar)
	if strings.Contains(baseCmd, "/") || strings.HasPrefix(baseCmd, ".") || strings.HasPrefix(baseCmd, "~") {
		return nil
	}

	// Skip environment variable assignments (KEY=val cmd)
	if strings.Contains(baseCmd, "=") {
		return nil
	}

	// A. Check if it's a shell builtin
	if shellBuiltins[baseCmd] {
		return nil // valid builtin, no error
	}

	// B. Check if it's a known user shell alias (e.g. ll, la, cls, gs, gp, alert)
	if IsAlias(baseCmd) {
		return nil // valid alias, NEVER error!
	}

	// C. Check if it exists on the system ($PATH)
	if isCommandOnPath(baseCmd) {
		return nil // valid installed executable, NEVER error!
	}

	// D. High-confidence direct typo map check
	if fix, ok := commonTypos[baseCmd]; ok {
		// Only suggest if the target exists, is a builtin, or is an alias
		if isCommandOnPath(fix) || shellBuiltins[fix] || IsAlias(fix) {
			fixedLine := fix + strings.TrimPrefix(activeSeg, baseCmd)
			return &Diagnostic{
				Severity:   SeverityError,
				Message:    fmt.Sprintf("Command '%s' not recognized", baseCmd),
				Suggestion: fmt.Sprintf("Did you mean '%s'?", fix),
				QuickFix:   prefix + fixedLine,
			}
		}
	}

	// E. Typo matching against common base commands
	if len(baseCmd) >= 2 && len(baseCmd) <= 15 {
		maxD := 1
		if len(baseCmd) >= 5 {
			maxD = 2
		}
		if match := FindClosestMatch(baseCmd, commonBaseCommands, maxD); match != "" {
			// Ensure the suggested match actually exists or is a builtin
			if isCommandOnPath(match) || shellBuiltins[match] || IsAlias(match) {
				fixedLine := match + strings.TrimPrefix(activeSeg, baseCmd)
				return &Diagnostic{
					Severity:   SeverityError,
					Message:    fmt.Sprintf("Command '%s' not recognized", baseCmd),
					Suggestion: fmt.Sprintf("Did you mean '%s'?", match),
					QuickFix:   prefix + fixedLine,
				}
			}
		}
	}

	return nil
}
