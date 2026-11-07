package providers

import (
	"encoding/json"
	"io/fs"
	"slices"
	"strings"
)

// Project is a read-only view of the project being detected.
type Project struct {
	FS fs.FS
}

// Exists reports whether the named file or directory exists.
func (p *Project) Exists(name string) bool {
	_, err := fs.Stat(p.FS, name)
	return err == nil
}

// AnyExists reports whether at least one of the named files exists.
func (p *Project) AnyExists(names ...string) bool {
	return slices.ContainsFunc(names, p.Exists)
}

// ReadJSON reads the named file and unmarshals it into v.
func (p *Project) ReadJSON(name string, v any) error {
	data, err := p.ReadFile(name)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// ReadFile reads the named file from the file system fs and returns its contents.
func (p *Project) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(p.FS, name)
}

// FirstLine reads the first line of the named file from the file system fs and returns it as a string.
func (p *Project) FirstLine(name string) string {
	data, err := p.ReadFile(name)
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(data), "\n")
	return strings.TrimSpace(line)
}

// Glob returns the names of all files matching the pattern in the file system fs.
func (p *Project) Glob(pattern string) []string {
	matches, _ := fs.Glob(p.FS, pattern)
	return matches
}
