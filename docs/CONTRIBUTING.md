<p align="center">
  <img src="docs/assets/yokipack.svg" width="120" alt="Nozoki, the Yōkipack mascot">
</p>

<h1 align="center">Contributing</h1>

<p align="center">
  Teach Yōkipack a new language by adding one Go file.<br>
  <sub>Everything else, including the tests, comes for free.</sub>
</p>

## Ways to help

| | |
|---|---|
| **Add a provider** | Support a new ecosystem (Go, Python, Rust, Ruby, Java, .NET, Deno…). One file in `internal/providers/`. |
| **Fix a suggestion** | A detected image or command is wrong for a common setup? Open an issue with the project's files listed. |
| **Improve the docs** | README, this guide, or code comments. |

## Setup

```sh
git clone https://github.com/uchkunovdaniel/yokipack
cd yokipack
go test ./...
```

Requires Go 1.26 or newer. The library has no dependencies, and the CLI uses
[cobra](https://github.com/spf13/cobra) and [huh](https://github.com/charmbracelet/huh).

## Adding a provider

A provider describes one ecosystem: which files identify it, which toolchains
can install it, and which commands build and run it. Providers register
themselves, so **you only add one file**. You don't need to edit a list, add a
test file or touch the CLI.

### 1. Create `internal/providers/<language>.go`

A simplified Go provider looks like this:

```go
package providers

const goImage = "docker.io/library/golang:1.26-alpine"

var golang = Provider{
	Name:     "go",
	Priority: PrioritySpecific, // go.mod only exists in Go projects
	Markers:  []string{"go.mod"},
	Toolchains: []Toolchain{
		{
			Name:      "go",
			Lockfiles: []string{"go.sum"},
			Image:     goImage,
			Install:   "go mod download",
		},
	},
	Commands: func(p *Project, tc Toolchain) (build, run string) {
		return "go build -o /usr/local/bin/app .", "app"
	},
}

func init() { register(golang) }
```

[`node.go`](../internal/providers/node.go) is the full real example, with four
toolchains and a preferred toolchain read from `package.json`.

### 2. Fill in the fields

**Provider**

| Field | Required | What it does |
|---|---|---|
| `Name` | yes | Unique name. Registering the same name twice panics. |
| `Priority` | yes | Which provider wins when several match. See below. |
| `Markers` | yes | Files at the project root that identify the ecosystem. Any one is enough. |
| `Toolchains` | yes | Ways to install the project, in order of preference. **The last one is the fallback.** |
| `Prefer` | no | Returns the name of a toolchain the project declares explicitly (e.g. `packageManager` in `package.json`), or `""`. |
| `Commands` | yes | Returns the build and run commands for the chosen toolchain. `""` means none. |

**Toolchain**

| Field | Required | What it does |
|---|---|---|
| `Name` | yes | Unique within the provider. |
| `Lockfiles` | no | If any exists, this toolchain is selected and `Install` is used. |
| `ConfigFiles` | no | Extra files install needs (e.g. `.npmrc`). Copied before install if present. |
| `Image` | yes | Base image, **fully qualified** with its registry (`docker.io/library/…`), because Podman rejects short names. |
| `Install` | yes | Install command when a lockfile exists. |
| `InstallNoLock` | no | Install command without a lockfile. Defaults to `Install`. |

Toolchain selection goes in this order: the `Prefer` result, then the first
toolchain whose lockfile exists, then the last toolchain.

Markers, lockfiles and config files that exist are copied before `Install`,
so the install layer stays cached until one of them changes.

### 3. Pick a priority

| Constant | Use it when your markers… | Examples |
|---|---|---|
| `PrioritySpecific` | only exist in this ecosystem | `go.mod`, `Cargo.toml`, `Gemfile`, `pom.xml`, `pyproject.toml`, `composer.json` |
| `PriorityOverlay` | come with another provider's marker and must win over it | `deno.json` next to `package.json` |
| `PriorityGeneric` | often show up in other ecosystems as tooling | `package.json` |
| `PriorityFallback` | match almost anything | `index.html` for a static site |

Ties are broken by `Name`, so detection is always deterministic.

### 4. Use the helpers

`Commands` and `Prefer` get a `*Project`, a read-only view of the project's
files:

| Method | Returns |
|---|---|
| `Exists(name)` | whether a file or directory exists |
| `AnyExists(names...)` | whether at least one exists |
| `ReadFile(name)` | the file's bytes |
| `ReadJSON(name, &v)` | the file decoded into `v` |
| `FirstLine(name)` | the first line, trimmed (e.g. `.python-version`) |
| `Glob(pattern)` | matching file names |

If your ecosystem runs named scripts from a JSON file the way npm does, reuse
`scriptCommands(file, key, runners)` instead of writing `Commands` yourself. It
builds with the `build` script and runs `start`, then `preview`, then `dev`.

### 5. Test it

```sh
go test ./internal/providers/        # contract tests run against every provider
go run ./cmd/yokipack -n path/to/a/real/project
```

The contract tests in [`contract_test.go`](../internal/providers/contract_test.go)
pick up your provider automatically. They check that:

- every required field is set and the priority is one of the constants,
- toolchain names are unique and every image is fully qualified,
- each lockfile selects its own toolchain,
- a project with only a marker falls back to the last toolchain and still gets a complete plan.

**If your markers can appear next to another provider's**, add a row to
`TestConflicts` so the expected winner is written down:

```go
{[]string{"pyproject.toml", "package.json"}, "python"},
```

## Guidelines

- **Suggest, don't decide.** Detection gives defaults that the user can override.
  A good guess for most projects beats a perfect guess for a few.
- **No framework-specific rules.** Read what the project declares (scripts,
  lockfiles, version files) instead of special-casing Next.js, Django or Rails.
- **If it can't be expressed, leave it empty.** An empty build or run command makes
  the CLI ask the user, which is better than a wrong answer.
- **Fully qualified images**, with official images preferred, and a major-version
  tag (`node:24-alpine`) rather than `latest`.
- **One file, short functions.** Each function does one thing, and shared logic
  belongs in `provider.go` or `scripts.go`, not copied between providers.

## Changing the Dockerfile template

The template lives in `internal/render/templates/`. Rendering is covered by
golden files, so regenerate them after a change and review the diff:

```sh
go test ./internal/render/ -update
git diff internal/render/testdata/
```

## Commits and pull requests

Commits follow [Conventional Commits](https://www.conventionalcommits.org/):

```text
feat(providers): add go provider
fix(providers): use bun image for bun lockfiles
docs: explain provider priorities
test(render): cover custom port
```

Keep one provider or fix per pull request, and make sure `go test ./...` and
`go vet ./...` pass.

<p align="center">
  <img src="docs/assets/yokify.svg" width="72" alt="Hakobi, the Yōkify mascot"><br>
  <sub>Thanks for helping. Yōkify will deliver it.</sub>
</p>

## Releasing

Releases are made from tags. Pushing a tag that starts with `v` runs
`.github/workflows/release.yml`, which tests the code, builds binaries for
macOS, Linux and Windows, and publishes a GitHub release. The release
description lists the commit messages since the previous tag.

Versions follow [semantic versioning](https://semver.org): bump `patch` for
fixes, `minor` for new features, `major` for breaking changes.

```sh
git tag -a v1.1.0 -m v1.1.0
git push origin v1.1.0
```
