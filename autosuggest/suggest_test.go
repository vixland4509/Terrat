package autosuggest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAutosuggestBasic(t *testing.T) {
	eng := NewEngine()
	eng.Add("git checkout -b feature/test")
	eng.Add("docker ps --format '{{.Names}}'")

	suffix := eng.Suggest("git check")
	expected := "out -b feature/test"
	if suffix != expected {
		t.Fatalf("expected suffix %q, got %q", expected, suffix)
	}

	suffix = eng.Suggest("GIT CHECK")
	if suffix != expected {
		t.Fatalf("expected case-insensitive matching %q, got %q", expected, suffix)
	}

	suffix = eng.Suggest("docker ps")
	expectedDocker := " --format '{{.Names}}'"
	if suffix != expectedDocker {
		t.Fatalf("expected docker suffix %q, got %q", expectedDocker, suffix)
	}
}

func TestAutosuggestAddAndPrioritize(t *testing.T) {
	eng := &Engine{
		historyM: make(map[string]struct{}),
		maxItems: 5,
	}

	eng.Add("make build")
	eng.Add("make test")

	// Most recently added should match first
	suffix := eng.Suggest("make")
	if suffix != " test" {
		t.Fatalf("expected ' test' as most recent match, got %q", suffix)
	}

	// Re-add make build to bump priority
	eng.Add("make build")
	suffix = eng.Suggest("make")
	if suffix != " build" {
		t.Fatalf("expected ' build' after priority bump, got %q", suffix)
	}
}

func TestAutosuggestEmpty(t *testing.T) {
	eng := NewEngine()
	if eng.Suggest("") != "" {
		t.Fatalf("empty input should return empty suggestion")
	}
	if eng.Suggest("completely_unknown_command_xyz_123") != "" {
		t.Fatalf("unknown input should return empty suggestion")
	}
}

func TestAutosuggestPathWithSpaces(t *testing.T) {
	tmpDir := t.TempDir()

	_ = os.Mkdir(filepath.Join(tmpDir, "My Projects Folder"), 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "script with space.sh"), []byte("echo"), 0755)

	eng := NewEngine()

	// 1. Unquoted prefix
	suffix := eng.Suggest("cd My", tmpDir)
	if suffix != "\\ Projects\\ Folder/" {
		t.Fatalf("expected '\\ Projects\\ Folder/', got %q", suffix)
	}

	// 2. Escaped space prefix
	suffix = eng.Suggest("cd My\\ ", tmpDir)
	if suffix != "Projects\\ Folder/" {
		t.Fatalf("expected 'Projects\\ Folder/', got %q", suffix)
	}

	// 3. Double-quoted prefix
	suffix = eng.Suggest("cd \"My", tmpDir)
	if suffix != " Projects Folder/" {
		t.Fatalf("expected ' Projects Folder/', got %q", suffix)
	}

	// 4. Single unescaped space typed after first word
	suffix = eng.Suggest("cd My ", tmpDir)
	if suffix != "Projects\\ Folder/" {
		t.Fatalf("expected 'Projects\\ Folder/', got %q", suffix)
	}

	// 5. File match
	suffix = eng.Suggest("cat scr", tmpDir)
	if suffix != "ipt\\ with\\ space.sh" {
		t.Fatalf("expected 'ipt\\ with\\ space.sh', got %q", suffix)
	}
}

