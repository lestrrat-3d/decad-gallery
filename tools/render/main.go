// The render tool turns gallery models into images.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
)

var galleryFresh bool

func main() {
	if err := runCLI(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runCLI parses args and renders. -all and -list work on the registry; otherwise one shot of one model is rendered.
func runCLI(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("render", flag.ContinueOnError)
	name := flags.String("model", "spurgear", "model directory name")
	shot := flags.String("shot", "hero", "image preset")
	output := flags.String("out", "", "PNG output path (required)")
	width := flags.Int("width", 0, "override preset width in pixels")
	height := flags.Int("height", 0, "override preset height in pixels")
	frames := flags.Int("frames", 0, "frames in the hand-crank fan animation, 0 for the preset")
	kit := flags.String("kit", "", "planetary gearbox bearing kit; empty selects the default")
	all := flags.Bool("all", false, "regenerate every committed image, or those of an explicit -model")
	list := flags.Bool("list", false, "print one line per committed image job")
	root := flags.String("root", "", "output root for -all; default is the nearest ancestor holding go.mod")
	flags.BoolVar(&galleryFresh, "gallery-fresh", false, "ignore cached gallery contact-sheet tiles")
	if err := flags.Parse(args); err != nil {
		return err
	}
	set := map[string]bool{}
	flags.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if *all || *list {
		return runBatch(ctx, flags, set, *all, *list, *name, *root, stdout)
	}
	if set["root"] {
		return errors.New("-root applies only with -all")
	}
	m, ok := lookup(*name)
	if !ok {
		if *shot == "readme" {
			return fmt.Errorf("unknown model %q", *name)
		}
		return fmt.Errorf("unknown image preset %q for model %q", *shot, *name)
	}
	return renderkit.Run(ctx, m, *shot, renderkit.Options{
		Output: *output, Width: *width, Height: *height, Frames: *frames, Kit: *kit,
	})
}

// runBatch handles -all and -list.
func runBatch(ctx context.Context, flags *flag.FlagSet, set map[string]bool, all, list bool, name, root string,
	stdout io.Writer) error {
	if all && list {
		return errors.New("-all and -list cannot be combined")
	}
	allowed := map[string]bool{"all": true, "list": true, "model": true, "root": all, "gallery-fresh": true}
	for flagName := range set {
		if allowed[flagName] {
			continue
		}
		if all {
			return errors.New("-all takes only -model and -root")
		}
		return errors.New("-list takes only -model")
	}
	models := registry()
	if set["model"] {
		m, ok := lookup(name)
		if !ok {
			return fmt.Errorf("unknown model %q", name)
		}
		models = []renderkit.Model{m}
	}
	if list {
		for _, m := range models {
			for _, j := range m.Jobs {
				kit := j.Kit
				if kit == "" {
					kit = "-"
				}
				if _, err := fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", m.Name, j.Shot, kit, path.Join(m.Dir, j.Out)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if root == "" {
		var err error
		if root, err = moduleRoot(); err != nil {
			return err
		}
	}
	return renderkit.RenderAll(ctx, root, models, stdout)
}

func lookup(name string) (renderkit.Model, bool) {
	for _, m := range registry() {
		if m.Name == name {
			return m, true
		}
	}
	return renderkit.Model{}, false
}

// moduleRoot returns the nearest ancestor of the working directory that holds go.mod.
func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no go.mod above the working directory; pass -root")
		}
		dir = parent
	}
}
