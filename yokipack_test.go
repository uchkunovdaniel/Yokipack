package yokipack_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/uchkunovdaniel/yokipack"
)

func TestFindContainerFile(t *testing.T) {
	tests := []struct {
		name   string
		fsys   fstest.MapFS
		want   string
		wantOK bool
	}{
		{"none", fstest.MapFS{"package.json": {}}, "", false},
		{"dockerfile", fstest.MapFS{"Dockerfile": {}}, "Dockerfile", true},
		{"containerfile", fstest.MapFS{"Containerfile": {}}, "Containerfile", true},
		{"both prefers containerfile", fstest.MapFS{"Dockerfile": {}, "Containerfile": {}}, "Containerfile", true},
		{"directory named Dockerfile", fstest.MapFS{"Dockerfile/x": {}}, "", false},
		{"only in subfolder", fstest.MapFS{"api/Dockerfile": {}}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := yokipack.FindContainerFile(tt.fsys)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("FindContainerFile() = %q, %v; want %q, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestDetectNothing(t *testing.T) {
	if plan, ok := yokipack.Detect(fstest.MapFS{"README.md": {}}); ok {
		t.Errorf("Detect() = %+v, true; want false", plan)
	}
}

func TestDetectAndRender(t *testing.T) {
	project := fstest.MapFS{
		"package.json":   {Data: []byte(`{"scripts": {"build": "vite build", "preview": "vite preview"}}`)},
		"pnpm-lock.yaml": {},
	}

	plan, ok := yokipack.Detect(project)
	if !ok {
		t.Fatal("Detect() = false, want a plan")
	}

	dockerfile, err := yokipack.Render(plan, yokipack.RenderOptions{Port: 3000})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	for _, want := range []string{
		"COPY pnpm-lock.yaml ./pnpm-lock.yaml",
		"RUN pnpm run build",
		"EXPOSE 3000",
		"CMD pnpm run preview --host 0.0.0.0 --port $PORT",
	} {
		if !strings.Contains(string(dockerfile), want) {
			t.Errorf("Dockerfile missing %q:\n%s", want, dockerfile)
		}
	}
}

func TestRenderRejectsIncompletePlan(t *testing.T) {
	// package.json without scripts → no run command to render
	plan, _ := yokipack.Detect(fstest.MapFS{"package.json": {Data: []byte(`{}`)}})
	if _, err := yokipack.Render(plan, yokipack.RenderOptions{}); err == nil {
		t.Error("Render() succeeded for a plan without a run command")
	}
}

func TestRenderIgnore(t *testing.T) {
	out, err := yokipack.RenderIgnore()
	if err != nil || len(out) == 0 {
		t.Fatalf("RenderIgnore() = %d bytes, %v", len(out), err)
	}
}
