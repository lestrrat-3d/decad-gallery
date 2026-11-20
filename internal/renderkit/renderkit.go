// Package renderkit holds the dispatch, scene, camera and image-output helpers that every item render package and
// tools/render share.
package renderkit

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/solidlens"
)

// SceneFunc builds one scene. ctx carries the call's Options; read them with OptionsFrom.
type SceneFunc func(ctx context.Context) (solidlens.Scene, error)

// Preset is a shot that renders one scene to one PNG.
type Preset struct {
	Width, Height int
	Scene         SceneFunc
}

// Options carries one render call's command-line options. Zero values select the shot's defaults.
type Options struct {
	Output        string // -out: a file, or a directory for the "readme" shot
	Width, Height int    // -width, -height
	Frames        int    // -frames
	Kit           string // -kit
}

// ShotFunc renders a shot whose output is not one preset PNG: a readme set, a GIF or a sheet.
type ShotFunc func(ctx context.Context, opts Options) error

// Job is one committed output that RenderAll regenerates.
type Job struct {
	Shot string
	Kit  string
	Out  string // slash-separated, relative to Model.Dir
}

// Model is one -model value.
type Model struct {
	Name       string              // -model value, e.g. "ballbearing"
	Dir        string              // slash-separated item directory relative to the repository root
	Presets    map[string]Preset   // shot to single PNG
	Shots      map[string]ShotFunc // shot to custom output; checked before Presets
	AcceptsKit bool                // true only for planetarygearbox
	Jobs       []Job               // committed outputs, in render order
}

type optionsKey struct{}

// WithOptions returns a context that carries opts.
func WithOptions(ctx context.Context, opts Options) context.Context {
	return context.WithValue(ctx, optionsKey{}, opts)
}

// OptionsFrom returns the Options that WithOptions stored in ctx, or zero Options when none were set.
func OptionsFrom(ctx context.Context) Options {
	opts, _ := ctx.Value(optionsKey{}).(Options)
	return opts
}

// Run renders one shot of m. It is safe for concurrent use: it keeps every per-call value in ctx and locals.
func Run(ctx context.Context, m Model, shot string, opts Options) error {
	if opts.Kit != "" && !m.AcceptsKit {
		return fmt.Errorf("-kit is not supported by -model %s", m.Name)
	}
	if shot == "readme" && (opts.Width != 0 || opts.Height != 0) {
		return fmt.Errorf("readme output uses fixed image sizes")
	}
	if opts.Output == "" {
		return fmt.Errorf("-out is required")
	}
	ctx = WithOptions(ctx, opts)
	if custom, ok := m.Shots[shot]; ok {
		return custom(ctx, opts)
	}
	selected, ok := m.Presets[shot]
	if !ok {
		return fmt.Errorf("unknown image preset %q for model %q", shot, m.Name)
	}
	if opts.Width < 0 || opts.Height < 0 {
		return fmt.Errorf("image dimensions must be positive")
	}
	width, height := opts.Width, opts.Height
	if width == 0 {
		width = selected.Width
	}
	if height == 0 {
		height = selected.Height
	}
	scene, err := selected.Scene(ctx)
	if err != nil {
		return fmt.Errorf("build %s/%s scene: %w", m.Name, shot, err)
	}
	return RenderPNG(ctx, opts.Output, scene, width, height)
}
