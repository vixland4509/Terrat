//go:build windows

package platform

import (
	"golang.org/x/sys/windows/registry"
)

// DetectSystemColorScheme returns true if Windows is using dark mode for apps.
func DetectSystemColorScheme() bool {
	k, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return true // default to dark
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return true
	}
	// AppsUseLightTheme == 0 means Dark mode, 1 means Light mode
	return val == 0
}
