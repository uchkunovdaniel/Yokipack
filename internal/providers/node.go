package providers

import "strings"

const (
	nodeImage = "docker.io/library/node:24-alpine"
	bunImage  = "docker.io/oven/bun:1-alpine"
)

var node = Provider{
	Name:     "node",
	Priority: PriorityGeneric, // many non-node projects also have a package.json
	Markers:  []string{"package.json"},
	Toolchains: []Toolchain{
		{
			Name:          "pnpm",
			Lockfiles:     []string{"pnpm-lock.yaml"},
			ConfigFiles:   []string{".npmrc", "pnpm-workspace.yaml"},
			Image:         nodeImage,
			Install:       "npm install -g pnpm && pnpm install --frozen-lockfile",
			InstallNoLock: "npm install -g pnpm && pnpm install",
		},
		{
			Name:        "yarn",
			Lockfiles:   []string{"yarn.lock"},
			ConfigFiles: []string{".npmrc", ".yarnrc", ".yarnrc.yml", ".yarn"},
			Image:       nodeImage,
			Install:     "yarn install",
		},
		{
			Name:          "bun",
			Lockfiles:     []string{"bun.lock", "bun.lockb"},
			ConfigFiles:   []string{".npmrc", "bunfig.toml"},
			Image:         bunImage,
			Install:       "bun install --frozen-lockfile",
			InstallNoLock: "bun install",
		},
		{
			Name:          "npm", // last = fallback
			Lockfiles:     []string{"package-lock.json", "npm-shrinkwrap.json"},
			ConfigFiles:   []string{".npmrc"},
			Image:         nodeImage,
			Install:       "npm ci",
			InstallNoLock: "npm install",
		},
	},
	Prefer:   nodePackageManager,
	Commands: scriptCommands("package.json", "scripts", nodeRunners),
}

// nodeRunners lists toolchains whose script runner differs from "<name> run".
var nodeRunners = map[string]scriptRunner{
	"npm": {run: "npm run", argSep: " --"},
}

func init() { register(node) }

// nodePackageManager reads the "packageManager" field from package.json,
// e.g. "pnpm@9.12.0" → "pnpm". Returns "" if missing or unreadable.
func nodePackageManager(p *Project) string {
	var pkg struct {
		PackageManager string `json:"packageManager"`
	}
	if p.ReadJSON("package.json", &pkg) != nil {
		return ""
	}
	name, _, _ := strings.Cut(pkg.PackageManager, "@")
	return name
}
