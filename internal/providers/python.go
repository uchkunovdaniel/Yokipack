package providers

import (
	"bytes"
	"strings"
)

const (
	pythonImage = "docker.io/library/python:3.14-slim"
	pipQuiet    = "pip install --no-cache-dir"
)

var python = Provider{
	Name:     "python",
	Priority: PrioritySpecific,
	Markers:  []string{"requirements.txt", "pyproject.toml", "Pipfile"},
	Toolchains: []Toolchain{
		{
			Name:          "uv",
			Lockfiles:     []string{"uv.lock"},
			ConfigFiles:   []string{".python-version", "uv.toml"},
			Image:         pythonImage,
			Install:       pipQuiet + " uv && uv sync --locked --no-dev --no-install-project",
			InstallNoLock: pipQuiet + " uv && uv sync --no-dev --no-install-project",
		},
		{
			Name:        "poetry",
			Lockfiles:   []string{"poetry.lock"},
			ConfigFiles: []string{"poetry.toml"},
			Image:       pythonImage,
			Install:     pipQuiet + " poetry && poetry config virtualenvs.create false && poetry install --only main --no-root",
		},
		{
			Name:          "pipenv",
			Lockfiles:     []string{"Pipfile.lock"},
			Image:         pythonImage,
			Install:       pipQuiet + " pipenv && pipenv install --deploy --system",
			InstallNoLock: pipQuiet + " pipenv && pipenv install --system --skip-lock",
		},
		{
			Name:    "pip", // last = fallback; requirements.txt is copied as a marker
			Image:   pythonImage,
			Install: pipQuiet + " -r requirements.txt",
		},
	},
	Prefer:   pythonToolchain,
	Commands: pythonCommands,
}

func init() { register(python) }

func pythonToolchain(p *Project) string {
	if p.AnyExists("uv.lock", "poetry.lock", "Pipfile.lock", "requirements.txt") {
		return "" // a lockfile decides
	}
	pyproject, _ := p.ReadFile("pyproject.toml")
	switch {
	case bytes.Contains(pyproject, []byte("[tool.poetry")):
		return "poetry"
	case bytes.Contains(pyproject, []byte("[tool.uv")):
		return "uv"
	case p.Exists("Pipfile"):
		return "pipenv"
	case pyproject != nil:
		return "uv"
	}
	return ""
}

func pythonCommands(p *Project, tc Toolchain) (build, run string) {
	prefix := ""
	if tc.Name == "uv" {
		prefix = "uv run "
	}

	deps := strings.ToLower(pythonDeps(p))
	entry := pythonEntry(p)
	module := strings.ReplaceAll(strings.TrimSuffix(entry, ".py"), "/", ".")

	switch {
	case p.Exists("manage.py"):
		run = "python manage.py runserver 0.0.0.0:$PORT"
	case strings.Contains(deps, "fastapi"):
		run = "uvicorn " + module + ":app --host 0.0.0.0 --port $PORT"
	case strings.Contains(deps, "flask"):
		run = "flask --app " + module + " run --host 0.0.0.0 --port $PORT"
	default:
		run = "python " + entry
	}
	return "", prefix + run
}

func pythonDeps(p *Project) string {
	var all []byte
	for _, f := range []string{"requirements.txt", "pyproject.toml", "Pipfile"} {
		data, _ := p.ReadFile(f)
		all = append(all, data...)
	}
	return string(all)
}

func pythonEntry(p *Project) string {
	for _, f := range []string{"main.py", "app.py", "app/main.py", "src/main.py"} {
		if p.Exists(f) {
			return f
		}
	}
	return "main.py"
}
