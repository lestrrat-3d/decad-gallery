package renderkit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lestrrat-3d/solidlens"
)

// RenderPNG renders scene to a temporary PNG beside output and renames it to output.
func RenderPNG(ctx context.Context, output string, scene solidlens.Scene, width, height int) error {
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(output), ".render-*.png")
	if err != nil {
		return fmt.Errorf("create temporary image: %w", err)
	}
	defer func() { _ = os.Remove(file.Name()) }()
	defer func() { _ = file.Close() }()
	if err := solidlens.RenderPNG(ctx, file, scene, solidlens.Settings{Width: width, Height: height}); err != nil {
		return fmt.Errorf("render PNG: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close rendered PNG: %w", err)
	}
	if err := os.Chmod(file.Name(), 0o644); err != nil {
		return fmt.Errorf("set rendered PNG permissions: %w", err)
	}
	if err := os.Rename(file.Name(), output); err != nil {
		return fmt.Errorf("save rendered PNG: %w", err)
	}
	return nil
}
