package renderkit

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/lestrrat-3d/solidlens"
)

// ReadmeFromHero builds hero once, writes index.png at 480 × 270, then for each of "top", "low", "high" present in
// cameras writes <name>.png at 1600 × 900 with that camera. cameras may be nil.
func ReadmeFromHero(ctx context.Context, outputDir string, hero SceneFunc, cameras map[string]solidlens.Camera) error {
	scene, err := hero(ctx)
	if err != nil {
		return fmt.Errorf("build hero scene: %w", err)
	}
	if err := RenderPNG(ctx, filepath.Join(outputDir, "index.png"), scene, 480, 270); err != nil {
		return fmt.Errorf("render index image: %w", err)
	}
	for _, shot := range []string{"top", "low", "high"} {
		camera, ok := cameras[shot]
		if !ok {
			continue
		}
		scene.Camera = camera
		if err := RenderPNG(ctx, filepath.Join(outputDir, shot+".png"), scene, 1600, 900); err != nil {
			return fmt.Errorf("render %s image: %w", shot, err)
		}
	}
	return nil
}
