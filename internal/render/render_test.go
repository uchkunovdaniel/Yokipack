// render_test.go
package render

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/uchkunovdaniel/yokipack/internal/providers"
)

// Run `go test ./internal/render/ -update` to regenerate testdata/*.golden
// after an intended template change, then review the diff.
var update = flag.Bool("update", false, "rewrite golden files")

const nodeImage = "docker.io/library/node:24-alpine"

func TestDockerfile(t *testing.T) {
	tests := []struct {
		name string
		plan providers.Plan
		opts Options
	}{
		{
			name: "full",
			plan: providers.Plan{
				Image:        nodeImage,
				InstallFiles: []string{"package.json", "package-lock.json"},
				Install:      "npm ci",
				Build:        "npm run build",
				Run:          "npm run preview -- --host 0.0.0.0 --port $PORT",
			},
		},
		{
			name: "minimal",
			plan: providers.Plan{Image: nodeImage, Run: "node server.js"},
		},
		{
			name: "install-without-files",
			plan: providers.Plan{Image: nodeImage, Install: "npm install", Run: "npm start"},
		},
		{
			name: "build-without-install",
			plan: providers.Plan{Image: nodeImage, Build: "make", Run: "./app"},
		},
		{
			name: "custom-port",
			plan: providers.Plan{Image: nodeImage, Run: "npm start"},
			opts: Options{Port: 3000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Dockerfile(tt.plan, tt.opts)
			if err != nil {
				t.Fatalf("Dockerfile() error: %v", err)
			}
			golden(t, tt.name+".Dockerfile", got)
		})
	}
}

func TestDockerfileDefaultPort(t *testing.T) {
	got, err := Dockerfile(providers.Plan{Image: nodeImage, Run: "npm start"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ENV PORT=8080", "EXPOSE 8080"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestDockerfileInvalid(t *testing.T) {
	tests := []struct {
		name   string
		modify func(p *providers.Plan)
	}{
		{"missing image", func(p *providers.Plan) { p.Image = "" }},
		{"missing run", func(p *providers.Plan) { p.Run = "" }},
		{"newline in image", func(p *providers.Plan) { p.Image = "node\nFROM evil" }},
		{"newline in install", func(p *providers.Plan) { p.Install = "npm ci\nUSER root" }},
		{"carriage return in install", func(p *providers.Plan) { p.Install = "npm ci\r\nUSER root" }},
		{"newline in build", func(p *providers.Plan) { p.Build = "x\ny" }},
		{"newline in run", func(p *providers.Plan) { p.Run = "npm start\nRUN rm -rf /" }},
		{"newline in install file", func(p *providers.Plan) { p.InstallFiles = []string{"package.json\nRUN x"} }},
		{"space in install file", func(p *providers.Plan) { p.InstallFiles = []string{"my file.json"} }},
		{"tab in install file", func(p *providers.Plan) { p.InstallFiles = []string{"my\tfile.json"} }}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := providers.Plan{Image: nodeImage, Run: "npm start"}
			tt.modify(&p)
			if out, err := Dockerfile(p, Options{}); err == nil {
				t.Errorf("Dockerfile() succeeded, want error; output:\n%s", out)
			}
		})
	}
}

func TestDockerignore(t *testing.T) {
	got, err := Dockerignore()
	if err != nil {
		t.Fatalf("Dockerignore() error: %v", err)
	}
	golden(t, "dockerignore", got)

	// Entries that must never be dropped, whatever else changes.
	lines := strings.Fields(string(got))
	for _, want := range []string{".git", ".env", "node_modules"} {
		if !contains(lines, want) {
			t.Errorf(".dockerignore is missing %q", want)
		}
	}
}

// golden compares got with testdata/<name>.golden, or rewrites it with -update.
func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")

	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden file %s (run with -update to create it): %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("output differs from %s (run with -update if intended)\n--- got ---\n%s\n--- want ---\n%s",
			path, got, want)
	}
}

func contains(list []string, s string) bool {
	return slices.Contains(list, s)
}
