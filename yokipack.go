// Package yokipack suggests Dockerfiles for projects. It detects the
// ecosystem from files in the project root, proposes default image,
// install, build and run commands, and renders them as a Dockerfile.
// The suggestions are defaults for the user to review, not guarantees.
package yokipack

import (
	"io/fs"

	"github.com/uchkunovdaniel/yokipack/internal/providers"
	"github.com/uchkunovdaniel/yokipack/internal/render"
)

// Plan holds the suggested build settings for a project.
// An empty field means "none".
type Plan = providers.Plan

// RenderOptions configures Render.
type RenderOptions = render.Options

// containerFiles in order of preference.
var containerFiles = []string{"Containerfile", "Dockerfile"}

// FindContainerFile reports the existing Containerfile or Dockerfile in the
// project root, preferring Containerfile when both exist.
func FindContainerFile(fsys fs.FS) (string, bool) {
	for _, name := range containerFiles {
		if info, err := fs.Stat(fsys, name); err == nil && !info.IsDir() {
			return name, true
		}
	}
	return "", false
}

// Detect suggests a Plan for the project, or reports false if no
// supported ecosystem was found.
func Detect(fsys fs.FS) (Plan, bool) {
	return providers.Detect(&providers.Project{FS: fsys})
}

// Render returns a Dockerfile for the plan. It fails if the plan has no
// image or run command, or if a value would break the Dockerfile.
func Render(p Plan, opts RenderOptions) ([]byte, error) {
	return render.Dockerfile(p, opts)
}

// RenderIgnore returns a .dockerignore suitable for most projects.
func RenderIgnore() ([]byte, error) {
	return render.Dockerignore()
}
