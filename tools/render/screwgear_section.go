package main

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/lestrrat-3d/solidlens"
)

// screwGearSectionTolerance is the chord tolerance in millimeters of the ribbon meshes the outlines are read from.
const screwGearSectionTolerance = 0.1

// screwGearSectionHeights are the cutting planes, each its distance in millimeters from the mid-plane between the
// two gear axes, toward gear B. The crests of the two ribbons overlap by the 1.15 mm engagement around the
// mid-plane, so these three planes cut through the interleaved teeth.
var screwGearSectionHeights = []struct {
	file   string
	height float64
}{
	{"section-below.png", -0.4},
	{"section-mid.png", 0},
	{"section-above.png", 0.4},
}

// Window half-widths in millimeters, along ê and k̂, around the centre of the mechanism.
const (
	screwGearSectionHalfU = 8.0
	screwGearSectionHalfV = 6.0
)

var (
	screwGearSectionFillA = solidlens.RGBA(0.14, 0.4, 0.58, 0.55)
	screwGearSectionFillB = solidlens.RGBA(0.72, 0.4, 0.14, 0.55)
)

// renderScrewGearSections builds the default mechanism once and writes one 1200 x 900 pixel plane section of
// the two ribbons for each of screwGearSectionHeights into outputDir.
func renderScrewGearSections(ctx context.Context, outputDir string) error {
	if outputDir == "" {
		return fmt.Errorf("-out is required")
	}
	mechanism, err := screwgear.Build(ctx, screwgear.NewDocument(), screwgear.DefaultParams())
	if err != nil {
		return fmt.Errorf("build screw gear: %w", err)
	}
	for _, shot := range screwGearSectionHeights {
		section, err := mechanism.SectionPair(ctx, shot.height, screwGearSectionTolerance)
		if err != nil {
			return err
		}
		view := solidlens.SectionView{
			Plane: solidlens.SectionPlane{Origin: section.Plane.Origin(), U: section.Plane.U(), V: section.Plane.V()},
			Layers: []solidlens.SectionLayer{
				{Name: "gear A", Loops: section.GearA.Loops, Fill: screwGearSectionFillA, Outline: screwGearColors[0]},
				{Name: "gear B", Loops: section.GearB.Loops, Fill: screwGearSectionFillB, Outline: screwGearColors[1]},
			},
			Window: &solidlens.SectionWindow{
				MinU: -screwGearSectionHalfU, MaxU: screwGearSectionHalfU,
				MinV: -screwGearSectionHalfV, MaxV: screwGearSectionHalfV,
			},
			Width: 1200, Height: 900,
			Caption: sectionCaption(shot.height, section),
		}
		img, _, err := solidlens.RenderSection(ctx, view)
		if err != nil {
			return fmt.Errorf("render screwgear %s: %w", shot.file, err)
		}
		if err := writeSectionPNG(filepath.Join(outputDir, shot.file), img); err != nil {
			return err
		}
	}
	return nil
}

// sectionCaption names the plane and states the least gap between the two outlines: the reading and the
// part of it decad proves, or that a ribbon misses the plane.
func sectionCaption(height float64, section screwgear.PairSection) string {
	plane := fmt.Sprintf("plane %+.1f mm from the mid-plane toward gear B", height)
	c := section.Clearance
	if c == nil {
		return plane + ": a ribbon misses the plane"
	}
	if c.Overlaps {
		return plane + ": the outlines overlap"
	}
	return fmt.Sprintf("%s: least gap %.2f mm, at least %.2f mm proven", plane, c.Distance.Value.Base(), c.Lower().Base())
}

func writeSectionPNG(path string, img *image.NRGBA) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	return nil
}
