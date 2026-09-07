//go:build !windows

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func DetectSystemColorScheme() bool {
	if out, err := exec.Command("dbus-send", "--session", "--print-reply=literal",
		"--dest=org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop",
		"org.freedesktop.portal.Settings.Read",
		"string:org.freedesktop.appearance", "string:color-scheme",
	).Output(); err == nil {
		s := string(out)
		if strings.Contains(s, "uint32 1") {
			return true
		}
		if strings.Contains(s, "uint32 2") {
			return false
		}
	}

	if out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output(); err == nil {
		s := strings.ToLower(string(out))
		if strings.Contains(s, "dark") {
			return true
		}
		if strings.Contains(s, "light") {
			return false
		}
	}

	for _, cmd := range []string{"kreadconfig6", "kreadconfig5"} {
		if out, err := exec.Command(cmd, "--group", "General", "--key", "ColorScheme").Output(); err == nil {
			s := strings.ToLower(string(out))
			if strings.Contains(s, "dark") {
				return true
			}
			if strings.Contains(s, "light") || strings.Contains(s, "breeze") {
				return false
			}
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		kdeGlobalsPath := filepath.Join(home, ".config", "kdeglobals")
		if data, err := os.ReadFile(kdeGlobalsPath); err == nil {
			content := strings.ToLower(string(data))
			for _, line := range strings.Split(content, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "colorscheme=") {
					if strings.Contains(line, "dark") {
						return true
					}
					if strings.Contains(line, "light") {
						return false
					}
				}
			}
		}
	}

	return true
}
