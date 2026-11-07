// Command yokipack suggests and writes a Dockerfile for a project.
package main

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/uchkunovdaniel/yokipack"
)

type options struct {
	file   string
	port   int
	yes    bool
	force  bool
	dryRun bool
	plan   yokipack.Plan // values given as flags
}

func newRootCmd() *cobra.Command {
	var opts options

	cmd := &cobra.Command{
		Use:          "yokipack [project-dir]",
		Short:        "Suggest and write a Dockerfile for a project",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := projectDir(args, opts.yes, cmd.InOrStdin(), cmd.OutOrStdout())
			if err != nil {
				return err
			}
			return run(cmd, root, opts)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&opts.file, "file", "f", "Dockerfile", "file to write")
	f.IntVarP(&opts.port, "port", "p", 8080, "port the app listens on")
	f.BoolVarP(&opts.yes, "yes", "y", false, "accept settings without prompting")
	f.BoolVarP(&opts.dryRun, "dry-run", "n", false, "print instead of writing")
	f.BoolVar(&opts.force, "force", false, "overwrite an existing Dockerfile")
	f.StringVarP(&opts.plan.Image, "image", "m", "", "base image")
	f.StringVarP(&opts.plan.Install, "install", "i", "", "install command")
	f.StringVarP(&opts.plan.Build, "build", "b", "", "build command")
	f.StringVarP(&opts.plan.Run, "run", "r", "", "run command")

	return cmd
}

// run puts the steps in order.
func run(cmd *cobra.Command, root string, opts options) error {
	in, out := cmd.InOrStdin(), cmd.OutOrStdout()
	fsys := os.DirFS(root)

	if err := validPort(strconv.Itoa(opts.port)); err != nil {
		return fmt.Errorf("--port: %w", err)
	}
	if err := checkExisting(fsys, opts.force); err != nil {
		return err
	}

	plan, ok := yokipack.Detect(fsys)
	if !ok {
		_, err := fmt.Fprintln(out, "No supported project detected, enter the settings manually.")
		if err != nil {
			return err
		}
	}
	plan = override(plan, opts.plan)
	port := opts.port

	if !opts.yes {
		var err error
		if plan, port, err = review(plan, port, in, out); err != nil {
			return err
		}
	}

	dockerfile, err := yokipack.Render(plan, yokipack.RenderOptions{Port: port})
	if err != nil {
		return err
	}
	if opts.dryRun {
		_, err = out.Write(dockerfile)
		return err
	}
	if err := write(filepath.Join(root, opts.file), dockerfile, opts.force); err != nil {
		return err
	}
	return writeIgnore(root)
}

// projectDir returns the argument if given, "." with --yes, or asks.
func projectDir(args []string, yes bool, in io.Reader, out io.Writer) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	dir := "."
	if yes {
		return dir, nil
	}
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Project directory").
			Value(&dir).
			SuggestionsFunc(func() []string { return subdirs(dir) }, &dir),
	)).WithInput(in).WithOutput(out).Run()
	return dir, err
}

// subdirs lists directories that complete the typed path, e.g. "sr" → "src/".
func subdirs(typed string) []string {
	parent := filepath.Dir(typed)
	if strings.HasSuffix(typed, "/") {
		parent = typed
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(parent, e.Name())+"/")
		}
	}
	return dirs
}

// checkExisting fails if the project already has a container file.
func checkExisting(fsys fs.FS, force bool) error {
	if name, ok := yokipack.FindContainerFile(fsys); ok && !force {
		return fmt.Errorf("%s already exists (use --force to replace it)", name)
	}
	return nil
}

// override replaces detected values with the ones given as flags.
func override(p, flags yokipack.Plan) yokipack.Plan {
	p.Image = cmp.Or(flags.Image, p.Image)
	p.Install = cmp.Or(flags.Install, p.Install)
	p.Build = cmp.Or(flags.Build, p.Build)
	p.Run = cmp.Or(flags.Run, p.Run)
	return p
}

// review lets the user check and edit the settings.
func review(p yokipack.Plan, port int, in io.Reader, out io.Writer) (yokipack.Plan, int, error) {
	portText := strconv.Itoa(port)
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Image").Value(&p.Image),
		huh.NewInput().Title("Install").Value(&p.Install),
		huh.NewInput().Title("Build").Value(&p.Build),
		huh.NewInput().Title("Port").Value(&portText).Validate(validPort),
		huh.NewInput().Title("Run").Value(&p.Run),
	)).WithInput(in).WithOutput(out).Run()
	if err != nil {
		return p, port, err
	}
	port, err = strconv.Atoi(portText)
	return p, port, err
}

// validPort reports whether s is a usable TCP port.
func validPort(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 65535 {
		return errors.New("must be a number between 1 and 65535")
	}
	return nil
}

// write creates path; it refuses to replace an existing file unless overwrite.
func write(path string, data []byte, overwrite bool) error {
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if overwrite {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		err := f.Close()
		if err != nil {
			return err
		}
		return err
	}
	return f.Close()
}

// writeIgnore adds a .dockerignore unless one already exists.
func writeIgnore(root string) error {
	data, err := yokipack.RenderIgnore()
	if err != nil {
		return err
	}
	err = write(filepath.Join(root, ".dockerignore"), data, false)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	return err
}
