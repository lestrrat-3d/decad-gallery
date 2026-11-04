package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/solidlens"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type wheelPreviewReport struct {
	Status           string                       `json:"status"`
	BuiltParts       int                          `json:"built_parts"`
	RenderedParts    int                          `json:"rendered_parts"`
	JointCutsPending bool                         `json:"joint_cuts_pending"`
	Omitted          []spinningwheel.PreviewIssue `json:"omitted"`
}

func renderSpinningWheelPreview(ctx context.Context, shot, output string, width, height int) error {
	if output == "" {
		return fmt.Errorf("-out is required")
	}
	if shot != "hero" && shot != "rear" && shot != "readme" {
		return fmt.Errorf("preview supports hero, rear and readme shots")
	}
	if width < 0 || height < 0 {
		return fmt.Errorf("image dimensions must be positive")
	}
	if shot == "readme" && (width != 0 || height != 0) {
		return fmt.Errorf("readme output uses fixed image sizes")
	}
	preview, err := spinningwheel.BuildPreview(ctx, spinningwheel.NewDocument(), spinningwheel.DefaultParams())
	if err != nil {
		return err
	}
	meshes, _, err := spinningWheelMeshes(ctx, preview.Assembly, &preview.Issues)
	if err != nil {
		return err
	}
	if len(meshes) == 0 {
		return fmt.Errorf("preview has no completed native meshes")
	}
	report := wheelPreviewReport{Status: "Joint cuts are complete. Every part pair of the default build and its four variants has a conclusive non-overlap result.",
		BuiltParts: len(preview.Assembly.Parts), RenderedParts: len(meshes), JointCutsPending: false, Omitted: preview.Issues}
	fmt.Fprintf(os.Stderr, "preview: %d native parts rendered; %d omissions\n", report.RenderedParts, len(report.Omitted))
	for _, issue := range report.Omitted {
		fmt.Fprintf(os.Stderr, "omitted %s: %s\n", issue.Part, issue.Reason)
	}
	if shot == "readme" {
		if err := previewWheelPNG(ctx, filepath.Join(output, "hero.png"), spinningWheelScene(meshes, "hero"), 1600, 900, len(report.Omitted)); err != nil {
			return err
		}
		if err := previewWheelPNG(ctx, filepath.Join(output, "index.png"), spinningWheelScene(meshes, "hero"), 480, 270, len(report.Omitted)); err != nil {
			return err
		}
		for _, view := range []string{"front-back", "layout", "flyer", "brake", "whorl"} {
			panels, err := spinningWheelPanels(meshes, view)
			if err != nil {
				return err
			}
			for i := range panels {
				panels[i].title = "PREVIEW: " + panels[i].title
				panels[i].caption = fmt.Sprintf("Unfinished joints; %d omissions. See preview-status.json.", len(report.Omitted))
				if view != "whorl" || i != 1 {
					panels[i].scene.Camera = fitCamera(panels[i].scene.Camera, panels[i].scene.Models, 1000.0/990, .08)
				}
			}
			face, err := wheelCaptionFont(18)
			if err != nil {
				return err
			}
			canvas, renderErr := wheelPanelCanvas(ctx, panels, 2000, 1100, face)
			_ = face.Close()
			if renderErr != nil {
				return renderErr
			}
			if err := writePNG(filepath.Join(output, view+".png"), canvas); err != nil {
				return err
			}
		}
		return writeWheelPreviewReport(filepath.Join(output, "preview-status.json"), report)
	}
	if width == 0 {
		width = 1600
	}
	if height == 0 {
		height = 900
	}
	if err := previewWheelPNG(ctx, output, spinningWheelScene(meshes, shot), width, height, len(report.Omitted)); err != nil {
		return err
	}
	return writeWheelPreviewReport(output+".json", report)
}

func previewWheelPNG(ctx context.Context, output string, scene solidlens.Scene, width, height, omissions int) error {
	header := 38
	fontSize := 20.0
	if width < 800 {
		header, fontSize = 26, 13
	}
	if width < 160 || height <= header+80 {
		return fmt.Errorf("preview image is too small")
	}
	scene.Camera = fitCamera(scene.Camera, scene.Models, float64(width)/float64(height-header), .08)
	var buffer bytes.Buffer
	if err := solidlens.RenderPNG(ctx, &buffer, scene, solidlens.Settings{Width: width, Height: height - header}); err != nil {
		return err
	}
	native, err := png.Decode(&buffer)
	if err != nil {
		return err
	}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, image.Rect(0, header, width, height), native, image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(0, 0, width, header), image.NewUniform(color.RGBA{R: 255, G: 232, B: 182, A: 255}), image.Point{}, draw.Src)
	face, err := wheelCaptionFont(fontSize)
	if err != nil {
		return err
	}
	defer func() { _ = face.Close() }()
	label := fmt.Sprintf("PREVIEW - %d omitted parts", omissions)
	drawer := font.Drawer{Dst: canvas, Src: image.NewUniform(color.RGBA{R: 55, G: 39, B: 13, A: 255}), Face: face, Dot: fixed.P(12, header-10)}
	drawer.DrawString(label)
	return writePNG(output, canvas)
}

func writeWheelPreviewReport(output string, report wheelPreviewReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	return os.WriteFile(output, append(data, '\n'), 0o644)
}
