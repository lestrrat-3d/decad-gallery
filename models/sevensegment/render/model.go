package render

import (
	"context"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
)

// Model returns the display rendering presets and jobs.
func Model() renderkit.Model {
	presets := map[string]renderkit.Preset{}
	for _, name := range []string{"hero", "top", "oblique45", "oblique30", "front", "side", "cutaway", "detent", "digits", "cams", "exploded", "exploded-camshaft"} {
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
					width = sevenMotionWidth
				}
				if height == 0 {
					height = sevenMotionHeight
				}
				return renderSevenSegmentMotion(ctx, opts.Output, width, height, sevenMotionFrames)
			},
		},
		Jobs: []renderkit.Job{{Shot: "readme", Out: "images"}, {Shot: "motion", Out: "images/motion.gif"}},
	}
}
