// Package rootcheck reads sections of a bevel gear's teeth from the toe to the heel, to prove that its roots carry
// a fillet. Every cone of a straight bevel tooth passes through the pitch apex, so a section normal to the shaft at
// station s from the apex cuts the root cone at radius RootSlope*s and the pitch cone at PitchSlope*s.
package rootcheck

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/r3"
)

const (
	// Heights is the number of sections between the toe and the heel.
	Heights = 8
	// Margin is the distance in mm from each end of the root cone to the first and last section.
	Margin = 0.3
	// boreMargin is the distance in mm below the root cone at which a window still counts as the teeth's.
	boreMargin = 0.5
	// FineWindow is the outline length in mm over which a fillet's turn must stay small. A sharp root turns 80 to 90
	// degrees in it. A fillet of 0.6 mm radius turns about 5 degrees in it, and the mesh spreads that over one
	// vertex to about 15.
	FineWindow = 0.05
	// FineMaxTurn is the turn in degrees a fillet may show in FineWindow.
	FineMaxTurn = 15.0
	// QuarterWindow and QuarterMinTurn are the window and the turn that prove a fillet's quarter turn is there: the
	// fillet spreads its 80 to 90 degrees over a window of 0.4 mm as 30 to 45 degrees, under the 50 degree limit.
	QuarterWindow  = 0.4
	QuarterMinTurn = 20.0
)

// Section is what one cut through the teeth reads.
type Section struct {
	// Height is the section offset along the axis passed to Read.
	Height float64
	// Station is the section's distance from the pitch apex along the shaft.
	Station float64
	// InTeeth reports whether the heel cone has left the pitch circle behind this section. When it is false the
	// pitch circle is cut by the heel face, and Crossings and the fillet windows are not read.
	InTeeth bool
	// Crossings counts the places the outline crosses the pitch circle: twice per tooth where the section cuts them.
	Crossings int
	// Quarter lists the turns in degrees of the windows of QuarterWindow that turn at least QuarterMinTurn toward the
	// material.
	Quarter []float64
	// FineMax is the largest turn in degrees over a FineWindow, or zero where none passes 15 degrees.
	FineMax float64
}

// Read cuts body at Heights stations from the toe root plus Margin to the heel root minus Margin, along the member's
// shaft. apex is the pitch apex, dir the unit direction of the shaft from the apex into the member, and axis the
// positive axis that cornercheck cuts normal to (dir, or its opposite). It returns each section's reading.
func Read(ctx context.Context, body *decad.Body, band bevelgear.ToothBand, apex, dir, axis r3.Vec) ([]Section, error) {
	var out []Section
	for i := range Heights {
		station := band.ToeRoot + Margin + (band.HeelRoot-band.ToeRoot-2*Margin)*float64(i)/float64(Heights-1)
		height := apex.Add(dir.Scale(station)).Dot(axis)
		outline, err := cornercheck.Outline(ctx, body, axis, height)
		if err != nil {
			return nil, fmt.Errorf("rootcheck: station %.2f: %w", station, err)
		}
		sec := Section{Height: height, Station: station, InTeeth: station < band.HeelPitch-Margin}
		if sec.InTeeth {
			sec.Crossings = cornercheck.Crossings(outline, apex, axis, band.PitchSlope*station)
			// Only the teeth: a window nearer the shaft than the root cone is the bore, whose corners are listed apart.
			onTeeth := func(f cornercheck.Finding) bool {
				d := f.At.Sub(apex)
				return d.Sub(axis.Scale(d.Dot(axis))).Len() > band.RootSlope*station-boreMargin
			}
			for _, f := range cornercheck.Windows(outline, cornercheck.Inside, QuarterWindow, QuarterMinTurn) {
				if onTeeth(f) {
					sec.Quarter = append(sec.Quarter, f.Turn)
				}
			}
			for _, f := range cornercheck.Windows(outline, cornercheck.Inside, FineWindow, FineMaxTurn) {
				if onTeeth(f) {
					sec.FineMax = max(sec.FineMax, f.Turn)
				}
			}
		}
		out = append(out, sec)
	}
	return out, nil
}

// SectionHeights returns the section heights Read cuts, for the corner check proper.
func SectionHeights(secs []Section) []float64 {
	out := make([]float64, len(secs))
	for i, s := range secs {
		out[i] = s.Height
	}
	return out
}
