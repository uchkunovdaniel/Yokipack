// contract_test.go
//
// Runs against every registered provider. Contributors get these checks for
// free; adding a provider file is enough for it to be tested here.
package providers

import (
	"slices"
	"strings"
	"testing"
)

var priorityLevels = []Priority{PrioritySpecific, PriorityOverlay, PriorityGeneric, PriorityFallback}

func TestContract(t *testing.T) {
	if len(providers) == 0 {
		t.Fatal("no providers registered")
	}

	for _, pr := range providers {
		t.Run(pr.Name, func(t *testing.T) {
			checkFields(t, pr)
			checkToolchainSelection(t, pr)
			checkFallback(t, pr)
		})
	}
}

// checkFields verifies the provider is fully filled in.
func checkFields(t *testing.T, pr Provider) {
	t.Helper()

	if pr.Name == "" {
		t.Error("Name is empty")
	}
	if !slices.Contains(priorityLevels, pr.Priority) {
		t.Errorf("Priority %d is not one of the Priority* constants", pr.Priority)
	}
	if len(pr.Markers) == 0 {
		t.Error("Markers is empty")
	}
	if pr.Commands == nil {
		t.Error("Commands is nil")
	}
	if len(pr.Toolchains) == 0 {
		t.Fatal("Toolchains is empty")
	}

	seen := map[string]bool{}
	for _, tc := range pr.Toolchains {
		switch {
		case tc.Name == "":
			t.Error("toolchain with empty Name")
		case seen[tc.Name]:
			t.Errorf("duplicate toolchain name %q", tc.Name)
		}
		seen[tc.Name] = true

		if tc.Install == "" {
			t.Errorf("toolchain %q: Install is empty", tc.Name)
		}
		if !fullyQualified(tc.Image) {
			t.Errorf("toolchain %q: image %q must include a registry, e.g. docker.io/library/...", tc.Name, tc.Image)
		}
	}
}

// checkToolchainSelection: each lockfile selects its own toolchain, locked.
func checkToolchainSelection(t *testing.T, pr Provider) {
	t.Helper()
	marker := concrete(pr.Markers[0])

	for _, tc := range pr.Toolchains {
		for _, lock := range tc.Lockfiles {
			gotPr, gotTc, locked, ok := match(project(map[string]string{
				marker: "{}",
				lock:   "",
			}))
			switch {
			case !ok:
				t.Errorf("lockfile %s: no provider matched", lock)
			case gotPr.Name != pr.Name:
				t.Errorf("lockfile %s: matched provider %q (check priorities and markers)", lock, gotPr.Name)
			case gotTc.Name != tc.Name:
				t.Errorf("lockfile %s: selected toolchain %q, want %q (check toolchain order)", lock, gotTc.Name, tc.Name)
			case !locked:
				t.Errorf("lockfile %s: not treated as locked", lock)
			}
		}
	}
}

// checkFallback: only a marker → last toolchain, unlocked, and Detect works.
func checkFallback(t *testing.T, pr Provider) {
	t.Helper()
	p := project(map[string]string{concrete(pr.Markers[0]): "{}"})
	want := pr.Toolchains[len(pr.Toolchains)-1]

	_, gotTc, locked, ok := match(p)
	if !ok {
		t.Fatal("marker alone: no provider matched")
	}
	if gotTc.Name != want.Name {
		t.Errorf("marker alone: selected %q, want fallback %q", gotTc.Name, want.Name)
	}
	if locked && len(want.Lockfiles) > 0 {
		t.Error("marker alone: reported as locked")
	}

	plan, _ := Detect(p)
	if plan.Image == "" || plan.Install == "" {
		t.Errorf("marker alone: incomplete plan %+v", plan)
	}
}

// fullyQualified reports whether an image names its registry host,
// e.g. docker.io/library/node or ghcr.io/org/app.
func fullyQualified(image string) bool {
	host, _, found := strings.Cut(image, "/")
	return found && strings.Contains(host, ".")
}

func TestNothingDetectedInEmptyProject(t *testing.T) {
	if plan, ok := Detect(project(map[string]string{})); ok {
		t.Errorf("empty project detected as %+v", plan)
	}
}

// TestConflicts documents which provider wins when markers of several
// providers are present. Add a row when your markers can appear alongside
// another provider's.
func TestConflicts(t *testing.T) {
	tests := []struct {
		files []string
		want  string
	}{
		{[]string{"package.json"}, "node"},
		//{[]string{"composer.json", "package.json"}, "php"},
		//{[]string{"pyproject.toml", "package.json"}, "python"},
		//{[]string{"deno.json", "package.json"}, "deno"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.files, "+"), func(t *testing.T) {
			files := map[string]string{}
			for _, f := range tt.files {
				files[f] = "{}"
			}
			pr, _, _, ok := match(project(files))
			if !ok || pr.Name != tt.want {
				t.Errorf("matched %q, want %q", pr.Name, tt.want)
			}
		})
	}
}
