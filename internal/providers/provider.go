package providers

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// Plan holds the suggested defaults for building and running a project.
// An empty field means "none".
type Plan struct {
	Image        string
	InstallFiles []string // copied before Install for layer caching
	Install      string
	Build        string
	Run          string
}

// Toolchain is one way to install a project within a provider,
// e.g. npm/pnpm for node, uv/pip for python, or the single go toolchain.
type Toolchain struct {
	Name          string
	Lockfiles     []string // any present → toolchain selected, install uses Install
	ConfigFiles   []string // extra files install needs (e.g. .npmrc); copied if present
	Image         string
	Install       string
	InstallNoLock string // used when no lockfile exists; "" = same as Install
}

type Priority int

const (
	// PrioritySpecific markers that only exist in this ecosystem's projects
	// (go.mod, Cargo.toml, composer.json, Gemfile, pom.xml, *.csproj, pyproject.toml).
	PrioritySpecific Priority = 50

	// PriorityOverlay markers that come with another provider's generic
	// marker and must win over it (deno.json next to package.json).
	PriorityOverlay Priority = 75

	// PriorityGeneric markers that often appear in other ecosystems'
	// projects as tooling (package.json).
	PriorityGeneric Priority = 100

	// PriorityFallback catch-alls that match almost anything
	// (index.html → static site).
	PriorityFallback Priority = 1000
)

// Provider describes one ecosystem. Providers are plain values that register
// themselves from their own file; shared behavior lives in this file so every
// provider works the same way.
type Provider struct {
	Name       string
	Priority   Priority    // lower = checked first; ties broken by Name
	Markers    []string    // any present at the project root → provider applies
	Toolchains []Toolchain // order = priority; the last one is the fallback

	// Prefer optionally returns the name of an explicitly declared toolchain
	// (e.g. a "packageManager" field or a [tool.poetry] section).
	// "" means no preference.
	Prefer func(p *Project) string

	// Commands returns the build and run commands for the chosen toolchain.
	// An empty string means "none".
	Commands func(p *Project, tc Toolchain) (build, run string)
}

// providers holds all registered providers, sorted by Priority then Name.
var providers []Provider

// register adds a provider. Call it from the provider file's init().
// It panics on a duplicate name, since that is a programming error.
func register(pr Provider) {
	for _, existing := range providers {
		if existing.Name == pr.Name {
			panic(fmt.Sprintf("providers: duplicate provider name %q", pr.Name))
		}
	}

	providers = append(providers, pr)
	slices.SortStableFunc(providers, func(a, b Provider) int {
		return cmp.Or(
			cmp.Compare(a.Priority, b.Priority),
			strings.Compare(a.Name, b.Name),
		)
	})
}

// Detect returns default build settings for the project, or false if no
// provider matches.
func Detect(p *Project) (Plan, bool) {
	pr, tc, locked, ok := match(p)
	if !ok {
		return Plan{}, false
	}

	install := tc.InstallNoLock
	if locked || install == "" {
		install = tc.Install
	}
	build, run := pr.Commands(p, tc)

	return Plan{
		Image:        tc.Image,
		InstallFiles: installFiles(p, pr, tc),
		Install:      install,
		Build:        build,
		Run:          run,
	}, true
}

// match finds the provider and toolchain for the project. locked reports
// whether that toolchain's lockfile exists.
func match(p *Project) (pr Provider, tc Toolchain, locked, ok bool) {
	for _, pr := range providers {
		if p.AnyExists(pr.Markers...) {
			tc, locked := pr.pickToolchain(p)
			return pr, tc, locked, true
		}
	}
	return Provider{}, Toolchain{}, false, false
}

// installFiles lists the files to copy before running Install, so the
// "install" layer stays cached until one of them changes: the provider's
// markers, then the toolchain's lockfiles and config files. Only files
// that exist are included, each once.
func installFiles(p *Project, pr Provider, tc Toolchain) []string {
	var files []string
	for _, f := range slices.Concat(pr.Markers, tc.Lockfiles, tc.ConfigFiles) {
		if p.Exists(f) && !slices.Contains(files, f) {
			files = append(files, f)
		}
	}
	return files
}

// pickToolchain chooses the preferred toolchain if declared, otherwise the
// first one whose lockfile exists, otherwise the last (fallback) toolchain.
// The bool reports whether that toolchain's lockfile exists.
func (pr Provider) pickToolchain(p *Project) (Toolchain, bool) {
	if pr.Prefer != nil {
		if name := pr.Prefer(p); name != "" {
			for _, tc := range pr.Toolchains {
				if tc.Name == name {
					return tc, p.AnyExists(tc.Lockfiles...)
				}
			}
		}
	}

	for _, tc := range pr.Toolchains {
		if p.AnyExists(tc.Lockfiles...) {
			return tc, true
		}
	}

	return pr.Toolchains[len(pr.Toolchains)-1], false
}
