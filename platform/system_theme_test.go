package platform

import "testing"

func TestDetectSystemColorScheme(t *testing.T) {
	isDark := DetectSystemColorScheme()
	t.Logf("Detected system dark mode: %v", isDark)
}
