package diagnostics

import (
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	cases := []struct {
		s, t     string
		expected int
	}{
		{"git", "gti", 2},
		{"docker", "dokcer", 2},
		{"ls", "sl", 2},
		{"cat", "cat", 0},
		{"make", "mkae", 2},
	}

	for _, c := range cases {
		d := LevenshteinDistance(c.s, c.t)
		if d != c.expected {
			t.Errorf("LevenshteinDistance(%q, %q) = %d; expected %d", c.s, c.t, d, c.expected)
		}
	}
}

func TestAnalyzeTypos(t *testing.T) {
	// Base command typos
	d := Analyze("gti status")
	if d == nil || d.Suggestion != "Did you mean 'git'?" {
		t.Fatalf("expected git suggestion for 'gti status', got %+v", d)
	}

	d = Analyze("dokcer ps")
	if d == nil || d.Suggestion != "Did you mean 'docker'?" {
		t.Fatalf("expected docker suggestion for 'dokcer ps', got %+v", d)
	}

	// Git subcommand typos
	d = Analyze("git puch origin")
	if d == nil || d.Suggestion != "Did you mean 'push'?" {
		t.Fatalf("expected push suggestion for 'git puch origin', got %+v", d)
	}

	d = Analyze("git stats")
	if d == nil || d.Suggestion != "Did you mean 'status'?" {
		t.Fatalf("expected status suggestion for 'git stats', got %+v", d)
	}

	// Unclosed quotes
	d = Analyze("git commit -m \"hello world")
	if d == nil || d.Severity != SeverityWarning {
		t.Fatalf("expected unclosed quote warning, got %+v", d)
	}

	// Privilege check
	d = Analyze("apt install neovim")
	if d == nil || d.Severity != SeverityWarning {
		t.Fatalf("expected root privilege warning for 'apt install', got %+v", d)
	}

	// Safe read-only operations should NOT warn
	if d := Analyze("apt search neovim"); d != nil {
		t.Fatalf("expected nil for 'apt search neovim', got %+v", d)
	}
	if d := Analyze("pacman -Ss firefox"); d != nil {
		t.Fatalf("expected nil for 'pacman -Ss firefox', got %+v", d)
	}

	// Valid subcommands should return nil (no false positives)
	if d := Analyze("git remote -v"); d != nil {
		t.Fatalf("expected nil for 'git remote -v', got %+v", d)
	}
	if d := Analyze("git config --global user.name foo"); d != nil {
		t.Fatalf("expected nil for 'git config', got %+v", d)
	}
	if d := Analyze("git blame main.go"); d != nil {
		t.Fatalf("expected nil for 'git blame', got %+v", d)
	}
	if d := Analyze("docker info"); d != nil {
		t.Fatalf("expected nil for 'docker info', got %+v", d)
	}
	if d := Analyze("docker stats"); d != nil {
		t.Fatalf("expected nil for 'docker stats', got %+v", d)
	}

	// Shell builtins should return nil
	if d := Analyze("cd /tmp"); d != nil {
		t.Fatalf("expected nil for builtin 'cd', got %+v", d)
	}
	if d := Analyze("export PATH=$PATH:/foo"); d != nil {
		t.Fatalf("expected nil for builtin 'export', got %+v", d)
	}

	// Sudo unnesting typo check
	d = Analyze("sudo gti status")
	if d == nil || d.Suggestion != "Did you mean 'git'?" {
		t.Fatalf("expected git suggestion for 'sudo gti status', got %+v", d)
	}

	// Pipeline active segment typo check
	d = Analyze("cat main.go | gerp func")
	if d == nil || d.Suggestion != "Did you mean 'grep'?" {
		t.Fatalf("expected grep suggestion for pipeline 'cat main.go | gerp func', got %+v", d)
	}

	// User and system aliases should be recognized and return nil
	RegisterAlias("ll", "ls -alF")
	RegisterAlias("cls", "clear")
	RegisterAlias("gs", "git status")

	if d := Analyze("ll /home"); d != nil {
		t.Fatalf("expected nil for alias 'll', got %+v", d)
	}
	if d := Analyze("cls"); d != nil {
		t.Fatalf("expected nil for alias 'cls', got %+v", d)
	}
	if d := Analyze("gs"); d != nil {
		t.Fatalf("expected nil for alias 'gs', got %+v", d)
	}

	// Register alias from command line
	RegisterAliasFromLine("alias mytest='echo hello'")
	if !IsAlias("mytest") {
		t.Fatalf("expected 'mytest' to be registered as alias")
	}
	if d := Analyze("mytest"); d != nil {
		t.Fatalf("expected nil for alias 'mytest', got %+v", d)
	}

	// Git aliases should return nil
	RegisterGitAlias("deploy")
	RegisterGitAlias("st")
	if d := Analyze("git deploy \"my feature\""); d != nil {
		t.Fatalf("expected nil for git alias 'deploy', got %+v", d)
	}
	if d := Analyze("git st"); d != nil {
		t.Fatalf("expected nil for git alias 'st', got %+v", d)
	}

	// Valid command should return nil
	d = Analyze("git status")
	if d != nil {
		t.Fatalf("expected nil for valid command, got %+v", d)
	}
}
