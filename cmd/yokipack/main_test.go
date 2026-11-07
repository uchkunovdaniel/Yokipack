package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/uchkunovdaniel/yokipack"
)

const nodeProject = `{"scripts": {"build": "vite build", "start": "node server.js"}}`

// project creates a temp directory with the given files.
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// execute runs the CLI with args and returns its output.
func execute(args ...string) (string, error) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetArgs(args)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := cmd.Execute()
	return out.String(), err
}

// read returns a file's contents, or "" if it doesn't exist.
func read(t *testing.T, path string) string {
	t.Helper()
	data, _ := os.ReadFile(path)
	return string(data)
}

func TestWritesFiles(t *testing.T) {
	dir := project(t, map[string]string{"package.json": nodeProject})

	if _, err := execute("-y", dir); err != nil {
		t.Fatal(err)
	}
	df := read(t, filepath.Join(dir, "Dockerfile"))
	for _, want := range []string{"RUN npm run build", "EXPOSE 8080", "CMD npm run start"} {
		if !strings.Contains(df, want) {
			t.Errorf("Dockerfile missing %q:\n%s", want, df)
		}
	}
	if read(t, filepath.Join(dir, ".dockerignore")) == "" {
		t.Error(".dockerignore not written")
	}
}

func TestCustomFileAndPort(t *testing.T) {
	dir := project(t, map[string]string{"package.json": nodeProject})

	if _, err := execute("-y", "-f", "Containerfile", "-p", "3000", dir); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(read(t, filepath.Join(dir, "Containerfile")), "EXPOSE 3000") {
		t.Error("Containerfile missing custom port")
	}
	if read(t, filepath.Join(dir, "Dockerfile")) != "" {
		t.Error("Dockerfile written despite --file")
	}
}

func TestExistingDockerfile(t *testing.T) {
	dir := project(t, map[string]string{
		"package.json":  nodeProject,
		"Dockerfile":    "mine",
		".dockerignore": "mine",
	})

	if _, err := execute("-y", dir); err == nil {
		t.Error("expected an error for an existing Dockerfile")
	}
	if read(t, filepath.Join(dir, "Dockerfile")) != "mine" {
		t.Error("Dockerfile changed without --force")
	}

	if _, err := execute("-y", "--force", dir); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(dir, "Dockerfile")) == "mine" {
		t.Error("--force did not replace the Dockerfile")
	}
	if read(t, filepath.Join(dir, ".dockerignore")) != "mine" {
		t.Error(".dockerignore was replaced")
	}
}

func TestDryRun(t *testing.T) {
	dir := project(t, map[string]string{"package.json": nodeProject})

	out, err := execute("-y", "--dry-run", dir)
	if err != nil || !strings.HasPrefix(out, "FROM ") {
		t.Fatalf("err %v, output %q", err, out)
	}
	if read(t, filepath.Join(dir, "Dockerfile")) != "" {
		t.Error("dry run wrote a Dockerfile")
	}
}

func TestUnsupportedProjectWithFlags(t *testing.T) {
	dir := project(t, map[string]string{"main.rs": ""})

	out, err := execute("-y", "--image", "docker.io/library/rust:1", "--build", "cargo build --release", "--run", "./app", dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No supported project detected") {
		t.Errorf("missing notice, output: %s", out)
	}
	df := read(t, filepath.Join(dir, "Dockerfile"))
	for _, want := range []string{"FROM docker.io/library/rust:1", "RUN cargo build --release", "CMD ./app"} {
		if !strings.Contains(df, want) {
			t.Errorf("Dockerfile missing %q:\n%s", want, df)
		}
	}
}

func TestFlagsOverrideDetected(t *testing.T) {
	dir := project(t, map[string]string{"package.json": nodeProject})

	if _, err := execute("-y", "--run", "node dist/main.js", dir); err != nil {
		t.Fatal(err)
	}
	df := read(t, filepath.Join(dir, "Dockerfile"))
	if !strings.Contains(df, "CMD node dist/main.js") || !strings.Contains(df, "RUN npm run build") {
		t.Errorf("expected overridden run and detected build:\n%s", df)
	}
}

func TestErrors(t *testing.T) {
	tests := map[string][]string{
		"nothing detected": {"-y", project(t, map[string]string{"README.md": ""})},
		"no run command":   {"-y", project(t, map[string]string{"package.json": "{}"})},
		"bad port":         {"-y", "--port", "0", project(t, map[string]string{"package.json": nodeProject})},
		"too many args":    {"a", "b"},
		"unknown flag":     {"--unknown"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := execute(args...); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestOverride(t *testing.T) {
	detected := yokipack.Plan{Image: "img", Install: "install", Build: "build", Run: "run"}

	got := override(detected, yokipack.Plan{Run: "custom"})
	want := yokipack.Plan{Image: "img", Install: "install", Build: "build", Run: "custom"}

	if got.Image != want.Image || got.Install != want.Install || got.Build != want.Build || got.Run != want.Run {
		t.Errorf("override() = %+v, want %+v", got, want)
	}
}

func TestValidPort(t *testing.T) {
	for in, ok := range map[string]bool{
		"8080": true, "1": true, "65535": true,
		"": false, "abc": false, "0": false, "65536": false, "-1": false,
	} {
		if err := validPort(in); (err == nil) != ok {
			t.Errorf("validPort(%q) = %v, want ok=%v", in, err, ok)
		}
	}
}
