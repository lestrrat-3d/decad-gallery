package main

// CI: isolated-tests

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lestrrat-3d/solidlens"
)

func TestRenderReadmeViewsPreservesOrderAndDimensions(t *testing.T) {
	views := []readmeView{
		{view: "hero", file: "hero.png", width: 1600, height: 900},
		{view: "hero", file: "index.png", width: 480, height: 270},
		{view: "top", file: "top.png", width: 1600, height: 900},
	}
	var built, rendered []string
	var dimensions [][2]int
	err := renderReadmeViews(t.Context(), t.TempDir(), views,
		func(_ context.Context, view string) (solidlens.Scene, error) {
			built = append(built, view)
			return solidlens.Scene{}, nil
		}, func(_ context.Context, output string, _ solidlens.Scene, width, height int) error {
			rendered = append(rendered, filepath.Base(output))
			dimensions = append(dimensions, [2]int{width, height})
			return nil
		})
	if err != nil {
		t.Fatalf("renderReadmeViews returned an error: %v", err)
	}
	if want := []string{"hero", "hero", "top"}; !reflect.DeepEqual(built, want) {
		t.Fatalf("built views = %v, want %v", built, want)
	}
	if want := []string{"hero.png", "index.png", "top.png"}; !reflect.DeepEqual(rendered, want) {
		t.Fatalf("rendered files = %v, want %v", rendered, want)
	}
	if want := [][2]int{{1600, 900}, {480, 270}, {1600, 900}}; !reflect.DeepEqual(dimensions, want) {
		t.Fatalf("render dimensions = %v, want %v", dimensions, want)
	}
}

func TestRenderReadmeViewsStopsOnFirstError(t *testing.T) {
	for _, failAt := range []string{"build", "render"} {
		t.Run(failAt, func(t *testing.T) {
			failure := errors.New("render failure")
			var built, rendered []string
			err := renderReadmeViews(t.Context(), t.TempDir(), []readmeView{
				{view: "first", file: "first.png", width: 1, height: 2},
				{view: "second", file: "second.png", width: 3, height: 4},
			}, func(_ context.Context, view string) (solidlens.Scene, error) {
				built = append(built, view)
				if failAt == "build" && view == "first" {
					return solidlens.Scene{}, failure
				}
				return solidlens.Scene{}, nil
			}, func(_ context.Context, output string, _ solidlens.Scene, _, _ int) error {
				file := filepath.Base(output)
				rendered = append(rendered, file)
				if failAt == "render" && file == "first.png" {
					return failure
				}
				return nil
			})
			if !errors.Is(err, failure) {
				t.Fatalf("renderReadmeViews error = %v, want %v", err, failure)
			}
			if failAt == "build" && len(rendered) != 0 {
				t.Fatalf("rendered after scene build failed: %v", rendered)
			}
			if failAt == "render" && !reflect.DeepEqual(rendered, []string{"first.png"}) {
				t.Fatalf("rendered after first render failed: %v", rendered)
			}
		})
	}
}
