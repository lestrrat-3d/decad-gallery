package sevensegment_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// radii returns the least distance of the outline of loops from the shaft axis (to its segments, so that a flat counts
// at its middle) and the greatest distance of its vertices.
func radii(loops [][]r3.Vec) (float64, float64) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, l := range loops {
		for i, a := range l {
			b := l[(i+1)%len(l)]
			ab := b.Sub(a)
			k := 0.0
			if d := ab.Dot(ab); d > 0 {
				k = math.Min(math.Max(-a.Dot(ab)/d, 0), 1)
			}
			lo = math.Min(lo, a.Add(ab.Scale(k)).Len())
			hi = math.Max(hi, a.Len())
		}
	}
	return lo, hi
}

// shaftEdge returns the y in [lo, hi] where pred of the shaft section's least and greatest radius changes value, found
// by bisection on sections of the built shaft. The least radius is the flat's distance wherever the section has a flat
// and the round's radius elsewhere, so the result does not depend on how far the camshaft is turned.
func shaftEdge(t *testing.T, shaft *decad.Body, lo, hi float64, pred func(minR, maxR float64) bool) float64 {
	t.Helper()
	at := func(y float64) bool {
		minR, maxR := radii(sevensegment.SectionAt(t, shaft, y))
		return pred(minR, maxR)
	}
	first := at(lo)
	require.NotEqual(t, first, at(hi), "the shaft's section changes between y %.2f and %.2f", lo, hi)
	for hi-lo > 0.004 {
		m := (lo + hi) / 2
		if at(m) == first {
			lo = m
		} else {
			hi = m
		}
	}
	return (lo + hi) / 2
}

// faceCoord returns the coordinate along axis of the face of body that separates a solid point from a void one, found by
// bisection with the mesh's crossing grid. from is a point in the material and to a point in the void beyond the face; the
// coordinate moves along the axis from one to the other.
func faceCoord(t *testing.T, g *crossingGrid, from, to r3.Vec) r3.Vec {
	t.Helper()
	require.True(t, g.pointIn(from), "%v lies in the material", from)
	require.False(t, g.pointIn(to), "%v lies in the void", to)
	for to.Sub(from).Len() > 0.004 {
		m := from.Add(to).Scale(0.5)
		if g.pointIn(m) {
			from = m
		} else {
			to = m
		}
	}
	return from.Add(to).Scale(0.5)
}

// TestSevenSegmentAxialFits reads the axial fits of the default display from the built bodies: each gap is the
// distance between two faces, one measured on each part (the bearings by their bounds, the shaft by its sections, the walls
// by bisection on their meshes). It covers T1, T2, R4, L3, L4, L8 and S1 (detent plate and guide plate) and S2 of the part
// fits table in instructions.md.
func TestSevenSegmentAxialFits(t *testing.T) {
	startDefault()
	t.Parallel()
	_, m := builtDefault(t)
	bounds := func(b *decad.Body) decad.Box {
		box, err := b.Bounds()
		require.NoError(t, err)
		return box
	}
	front, rear := bounds(m.Bearings[0].Body), bounds(m.Bearings[1].Body)
	frontWall, rearWall := m.FrontWall, m.RearWall
	wallGrid := func(b *decad.Body) *crossingGrid {
		mesh, _ := meshOf(t, b, 0.05)
		return newCrossingGrid(mesh)
	}
	// A point in the plate of a wall, clear of its ribs, seat, holes and bolt pockets, at the given y.
	probe := func(y float64) r3.Vec { return r3.Vec{X: -22, Y: y, Z: sevensegment.ShaftZ - 12} }
	frontG, rearG := wallGrid(frontWall), wallGrid(rearWall)
	frontInner := faceCoord(t, frontG, probe(-52), probe(-49)).Y
	frontOuter := faceCoord(t, frontG, probe(-52), probe(-60)).Y
	rearInner := faceCoord(t, rearG, probe(46), probe(40)).Y
	require.InDelta(t, 8.0, frontInner-frontOuter, 0.01, "the front wall plate's thickness")
	require.InDelta(t, front.Max.Y, frontInner, 0.01, "the front bearing is flush with the front wall's inner face")
	require.InDelta(t, rear.Min.Y, rearInner, 0.01, "the rear bearing is flush with the rear wall's inner face")

	// Faces of the shaft, found on its sections.
	frontStep := shaftEdge(t, m.Shaft, front.Max.Y-0.4, front.Max.Y+0.4, func(_, maxR float64) bool { return maxR < 3.0 })
	rearCollar := shaftEdge(t, m.Shaft, rear.Min.Y-0.6, rear.Min.Y+0.2, func(minR, _ float64) bool { return minR > 3.0 })
	coneStart := shaftEdge(t, m.Shaft, 39.4, 40.5, func(_, maxR float64) bool { return maxR < 3.21 })
	camFlatEnd := shaftEdge(t, m.Shaft, 38.9, 39.5, func(minR, _ float64) bool { return minR < 3.0 })
	stubFlatEnd := shaftEdge(t, m.Shaft, -58.2, -57.5, func(minR, _ float64) bool { return minR < 2.25 })
	camA, ring, ratchet := bounds(m.Cams[sevensegment.SegA]), bounds(m.CollarRing), bounds(m.RatchetWheel)
	detent, guide := bounds(m.DetentPlate), bounds(m.GuidePlate)
	// The ledge ribs' top faces, read inside a rib of each wall: the first of a few x positions that lies in the rib's
	// material, clear of the bolt pockets.
	ledge := func(g *crossingGrid, y float64) float64 {
		for _, x := range []float64{21, 22, 23, 24, 25, 26} {
			if g.pointIn(r3.Vec{X: x, Y: y, Z: 52}) && !g.pointIn(r3.Vec{X: x, Y: y, Z: 55}) {
				return faceCoord(t, g, r3.Vec{X: x, Y: y, Z: 52}, r3.Vec{X: x, Y: y, Z: 55}).Z
			}
		}
		require.Fail(t, "no ledge rib material found", "y %.1f", y)
		return 0
	}
	// A column's top face, read at a corner of its 5 x 5 section, clear of the peg that stands on it.
	columnTop := faceCoord(t, wallGrid(m.GuidePlate), r3.Vec{X: 14.3, Y: -42.7, Z: 78}, r3.Vec{X: 14.3, Y: -42.7, Z: 80.4}).Z
	cradle := bounds(m.CradlePlate)
	frontLedge, rearLedge := ledge(frontG, frontInner+4), ledge(rearG, rearInner-4)

	for _, c := range []struct {
		id        string
		got, want float64
	}{
		{"T1 front step to the front bearing's inner face", frontStep - front.Max.Y, sevensegment.TightFit},
		{"T2 rear collar face to the rear bearing's inner face", rear.Min.Y - rearCollar, sevensegment.TightFit},
		{"R4 collar ring to the front wall's outer face", frontOuter - ring.Max.Y, sevensegment.RegularFit},
		{"L3 ratchet front face to the front bearing", ratchet.Min.Y - front.Max.Y, 1.0},
		{"L3 ratchet front face to the front wall", ratchet.Min.Y - frontInner, 1.0},
		{"L4 cam a back face to the rear collar cone", coneStart - camA.Max.Y, sevensegment.LooseFit},
		{"L8 stub flat end to the collar ring", stubFlatEnd - ring.Max.Y, sevensegment.PressFit},
		{"L8 cam flat end to cam a", camFlatEnd - camA.Max.Y, sevensegment.PressFit},
		{"S1 detent plate to the front wall", detent.Min.Y - frontInner, sevensegment.Seam},
		{"S1 guide plate to the front ledge ribs", guide.Min.Z - frontLedge, sevensegment.Seam},
		{"S1 guide plate to the rear ledge ribs", guide.Min.Z - rearLedge, sevensegment.Seam},
		{"S2 columns to the cradle plate", cradle.Min.Z - columnTop, 0.85},
	} {
		t.Logf("%s: %.3f (modeled %.2f)", c.id, c.got, c.want)
		// The cone's start is read 0.017 past its apex, where the section has grown by the probe's 0.01.
		tol := 0.02
		if c.id[:2] == "L4" {
			tol = 0.04
		}
		require.InDelta(t, c.want, c.got, tol, c.id)
	}
}
