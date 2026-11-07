package providers

import (
	"strings"
	"testing/fstest"
)

// project builds an in-memory project from file names and contents.
func project(files map[string]string) *Project {
	fsys := fstest.MapFS{}
	for name, data := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(data)}
	}
	return &Project{FS: fsys}
}

// concrete turns a marker like "*.csproj" into a real file name for fixtures.
func concrete(marker string) string {
	return strings.ReplaceAll(marker, "*", "app")
}
