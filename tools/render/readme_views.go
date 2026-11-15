package main

import (
	"context"
	"path/filepath"

	"github.com/lestrrat-3d/solidlens"
)

type readmeView struct {
	view, file    string
	width, height int
}

func renderReadmeViews(
	ctx context.Context,
	outputDir string,
	views []readmeView,
	build func(context.Context, string) (solidlens.Scene, error),
	render func(context.Context, string, solidlens.Scene, int, int) error,
) error {
	for _, view := range views {
		scene, err := build(ctx, view.view)
		if err != nil {
			return err
		}
		if err := render(ctx, filepath.Join(outputDir, view.file), scene, view.width, view.height); err != nil {
			return err
		}
	}
	return nil
}
