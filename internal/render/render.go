package render

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/uchkunovdaniel/yokipack/internal/providers"
)

//go:embed templates/*
var files embed.FS

var templates = template.Must(
	template.New("").Funcs(template.FuncMap{"join": strings.Join}).
		ParseFS(files, "templates/*.tmpl"),
)

type Options struct {
	Port int // default 8080
}

// Dockerfile renders a Dockerfile for the plan.
func Dockerfile(p providers.Plan, opts Options) ([]byte, error) {
	if err := validate(p); err != nil {
		return nil, err
	}
	if opts.Port == 0 {
		opts.Port = 8080
	}
	return execute("Dockerfile.tmpl", struct {
		Plan providers.Plan
		Port int
	}{p, opts.Port})
}

// Dockerignore renders a .dockerignore file.
func Dockerignore() ([]byte, error) {
	return execute("dockerignore.tmpl", nil)
}

func execute(name string, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, fmt.Errorf("render %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

func validate(p providers.Plan) error {
	if p.Image == "" {
		return errors.New("render: image is required")
	}
	if p.Run == "" {
		return errors.New("render: run command is required")
	}
	for _, v := range slices.Concat([]string{p.Image, p.Install, p.Build, p.Run}, p.InstallFiles) {
		if strings.ContainsAny(v, "\r\n") {
			return fmt.Errorf("render: value must be a single line: %q", v)
		}
	}
	for _, f := range p.InstallFiles {
		if strings.ContainsAny(f, " \t") {
			return fmt.Errorf("render: install file must not contain spaces: %q", f)
		}
	}
	return nil
}
