package policy

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadFromFile reads and parses a YAML policy file. Returns error if file cannot be read or parsed.
// Sanitizes the path to prevent path traversal outside the current working directory.
func LoadFromFile(path string) (*Policy, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}

	rel, err := filepath.Rel(cwd, absPath)
	if err != nil {
		return nil, fmt.Errorf("calculate relative path: %w", err)
	}

	// Check if the path traverses above the current working directory.
	// ".." means it's outside or at the parent of the CWD.
	if len(rel) >= 2 && rel[:2] == ".." {
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
