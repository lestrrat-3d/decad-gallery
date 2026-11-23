package render

import (
	"context"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
)

// Model returns the display rendering presets and jobs.
func Model() renderkit.Model {
	presets := map[string]renderkit.Preset{}
	for _, name := range []string{"hero", "top", "oblique45", "oblique30", "front", "side", "cutaway", "indexer", "digits", "cams", "exploded", "exploded-camshaft", "exploded-pivot", "pivot-section"} {
		presets[name] = renderkit.Preset{Width: 1600, Height: 900, Scene: sevenSegmentPreset(name)}
	}
	return renderkit.Model{
		Name: "sevensegment", Dir: "models/sevensegment", Presets: presets,
		Shots: map[string]renderkit.ShotFunc{
			"readme": func(ctx context.Context, opts renderkit.Options) error {
				return renderSevenSegmentReadme(ctx, opts.Output)
			},
			"motion": func(ctx context.Context, opts renderkit.Options) error {
				width, height := opts.Width, opts.Height
				if width == 0 {
					width = 960
				}
				if height == 0 {
					height = 540
				}
				return renderSevenSegmentMotion(ctx, opts.Output, width, height)
			},
		},
		Jobs: []renderkit.Job{{Shot: "readme", Out: "images"}, {Shot: "motion", Out: "images/motion.gif"}},
	}
}
