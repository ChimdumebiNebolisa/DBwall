package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadFromFile reads and parses a YAML policy file. Returns error if file cannot be read or parsed.
// Sanitizes the path to prevent path traversal outside the current working directory.
// Path comparison normalizes symlinks/short names where the OS supports it so a
// policy inside the working directory is never falsely rejected (audit finding F-009).
func LoadFromFile(path string) (*Policy, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}

	if !pathWithinDir(absPath, cwd) {
		return nil, fmt.Errorf("security: policy path %q is outside the working directory", path)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("read policy file: no such file or directory: %s", absPath)
		}
		return nil, fmt.Errorf("read policy file: %w", err)
	}
	return LoadFromBytes(data)
}

// LoadFromBytes parses YAML policy from bytes. Caller should validate the result.
func LoadFromBytes(data []byte) (*Policy, error) {
	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse policy YAML: %w", err)
	}
	return &p, nil
}

func pathWithinDir(target, dir string) bool {
	target = normalizePathForCompare(target)
	dir = normalizePathForCompare(dir)
	if target == "" || dir == "" {
		return false
	}
	rel, err := filepath.Rel(dir, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return false // the policy must be a file inside dir, not the directory itself
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != string(filepath.Separator)
}

func normalizePathForCompare(p string) string {
	p = cleanPathNormalized(p)
	if resolved, ok := resolveRealPath(p); ok {
		p = cleanPathNormalized(resolved)
	}
	return strings.ToLower(p)
}

func cleanPathNormalized(p string) string {
	vol := filepath.VolumeName(p)
	cleaned := filepath.Clean(p)
	if vol != "" && len(cleaned) > len(vol) {
		rest := cleaned[len(vol):]
		rest = strings.TrimPrefix(rest, `\`)
		rest = strings.TrimPrefix(rest, `/`)
		cleaned = vol + string(os.PathSeparator) + rest
	}
	return cleaned
}
