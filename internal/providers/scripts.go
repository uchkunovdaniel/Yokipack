package providers

import "encoding/json"

// scriptRunner describes how a toolchain runs a script.
type scriptRunner struct {
	run    string // e.g. "npm run"
	argSep string // separator to pass flags to a script, e.g. " --" for npm
}

// scriptCommands returns a Commands function that reads scripts from
// file (e.g. "package.json") under key (e.g. "scripts") and picks the
// build and run commands using the runner for the chosen toolchain.
func scriptCommands(file, key string, runners map[string]scriptRunner) func(*Project, Toolchain) (string, string) {
	return func(p *Project, tc Toolchain) (build, run string) {
		r := runnerFor(tc, runners)
		scripts := readScripts(p, file, key)

		if has(scripts, "build") {
			build = r.run + " build"
		}

		listen := r.argSep + " --host 0.0.0.0 --port $PORT"
		switch {
		case has(scripts, "start"):
			run = r.run + " start"
		case has(scripts, "preview"):
			run = r.run + " preview" + listen
		case has(scripts, "dev"):
			run = r.run + " dev" + listen
		}

		return build, run
	}
}

// readScripts returns the string map at key in a JSON file, or nil if the
// file is missing, invalid, or has no such key.
func readScripts(p *Project, file, key string) map[string]string {
	var doc map[string]json.RawMessage
	if p.ReadJSON(file, &doc) != nil {
		return nil
	}

	var scripts map[string]string
	_ = json.Unmarshal(doc[key], &scripts) // missing key → nil map
	return scripts
}

func has(m map[string]string, key string) bool {
	_, ok := m[key]
	return ok
}

// runnerFor returns the toolchain's script runner. Toolchains without an
// entry in runners default to "<name> run" (npm, pnpm, yarn, bun all work).
func runnerFor(tc Toolchain, runners map[string]scriptRunner) scriptRunner {
	if r, ok := runners[tc.Name]; ok {
		return r
	}
	return scriptRunner{run: tc.Name + " run"}
}
