//go:build !windows

package policy

import "path/filepath"

// resolveRealPath resolves symlinks on non-Windows platforms. Failures are
// non-fatal: the containment check falls back to the cleaned path form.
func resolveRealPath(p string) (string, bool) {
	if p == "" {
		return "", false
	}
	resolved, err := filepath.EvalSymlinks(p)
	return resolved, err == nil
}
