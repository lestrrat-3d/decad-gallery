package turbinefan_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// poseTransform returns the transform of PrintPoseFor(sz, pitch, hand).
func poseTransform(t *testing.T, sz turbinefan.Size, pitch float64, hand turbinefan.Hand) (r3.Frame, r3.Transform) {
	t.Helper()
	f, err := turbinefan.PrintPoseFor(sz, pitch, hand)
	require.NoError(t, err)
	xf, err := r3.FromFrame(f)
	require.NoError(t, err)
	return f, xf
}

// posedBlade builds the blade of hand at rotor d and pitch in a fresh NewDocument, moves it into the print pose and
// returns the document and the posed blade.
func posedBlade(t *testing.T, d, pitch float64, hand turbinefan.Hand) (*decad.Document, *decad.Body) {
	t.Helper()
	sz := mustSize(t, d)
	doc := turbinefan.NewDocument()
	build := turbinefan.BuildBlade
	if hand == turbinefan.LeftHand {
		build = turbinefan.BuildBladeMirrored
	}
	blade, err := build(t.Context(), doc, pitch, sz)
	require.NoError(t, err)
	_, xf := poseTransform(t, sz, pitch, hand)
	posed, err := decad.Transform(t.Context(), blade.Body, xf)
	require.NoError(t, err)
	return doc, posed
}

// requireBoxNear requires two boxes to agree within tol on every face; mirror negates y of want first.
func requireBoxNear(t *testing.T, want, got decad.Box, tol float64, mirror bool, name string) {
	t.Helper()
	lo, hi := want.Min, want.Max
	if mirror {
		lo.Y, hi.Y = -want.Max.Y, -want.Min.Y
	}
	for i, pair := range [][2]float64{
		{lo.X, got.Min.X}, {lo.Y, got.Min.Y}, {lo.Z, got.Min.Z}, {hi.X, got.Max.X}, {hi.Y, got.Max.Y}, {hi.Z, got.Max.Z},
	} {
		require.InDelta(t, pair[0], pair[1], tol, "%s bound %d", name, i)
	}
}

// insideFootprint reports whether (x, y) lies inside the keel's plan outline.
func insideFootprint(fp [][2]float64, x, y float64) bool {
	in := false
	for i := range fp {
		a, c := fp[i], fp[(i+len(fp)-1)%len(fp)]
		if (a[1] > y) != (c[1] > y) && x < (c[0]-a[0])*(y-a[1])/(c[1]-a[1])+a[0] {
			in = !in
		}
	}
	return in
}

// requireHarnessBuild builds and checks one harness and returns it.
func requireHarnessBuild(t *testing.T, d, pitch float64, hand turbinefan.Hand) *turbinefan.BladeHarness {
	t.Helper()
	sz := mustSize(t, d)
	doc, blade := posedBlade(t, d, pitch, hand)
	h, err := turbinefan.BuildBladeHarness(t.Context(), doc, blade, sz, hand)
	require.NoError(t, err, "rotor %g, pitch %g, hand %v", d, pitch, hand)
	require.True(t, h.Body.IsSolid())
	require.Equal(t, 23, len(h.Sketches))
	for i, s := range h.Sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
	}
	// Transform retires the blade it moves, so the document holds the posed blade and the harness.
	require.Equal(t, 2, len(doc.Bodies()), "posed blade and harness")
	box, err := h.Body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 0, box.Min.Z, box.Bound.Base()+1e-6)

	pressure := -1.0
	if hand == turbinefan.LeftHand {
		pressure = 1
	}
	require.Equal(t, pressure, h.Walls[0].Side)
	require.Equal(t, -pressure, h.Walls[1].Side)
	require.True(t, h.Walls[0].Pressure)
	require.False(t, h.Walls[1].Pressure)
	_, xf := poseTransform(t, sz, pitch, hand)
	xs := xf.Apply(r3.Vec{Z: sz.BladeStudStart}).X
	for i, w := range h.Walls {
		require.InDelta(t, xs, w.Inner[0][0], 1e-6, "wall %d start", i)
		require.InDelta(t, h.Gates[5].Contact.X+turbinefan.HarnessMargin, w.Inner[len(w.Inner)-1][0], 1e-6, "wall %d end", i)
	}

	head := turbinefan.GateEmbed + turbinefan.GateNeckLength + turbinefan.GateTaper
	for i, g := range h.Gates[:6] {
		require.InDelta(t, -math.Sin(turbinefan.GateRise), g.Dir.Z, 1e-9, "gate %d rise", i)
		require.InDelta(t, 0, g.Dir.X, 1e-9, "gate %d plane", i)
		inner, ok := h.Walls[0].InnerAt(g.Contact.X)
		require.True(t, ok, "gate %d over the wall", i)
		cone := math.Abs(g.Contact.Y-inner) - head*math.Cos(turbinefan.GateRise) -
			turbinefan.GateRadius*math.Sin(turbinefan.GateRise)
		require.GreaterOrEqual(t, cone, turbinefan.GateConeClearance-1e-9, "gate %d cone clearance", i)
	}
	for i := 6; i < 12; i++ {
		g, w := h.Gates[i], h.Walls[1]
		if i == 6 {
			w = h.Walls[0]
		}
		require.InDelta(t, 0, g.Dir.X, 1e-9, "tab %d", i)
		require.InDelta(t, w.Side, g.Dir.Y, 1e-9, "tab %d", i)
		require.InDelta(t, 0, g.Dir.Z, 1e-9, "tab %d", i)
		inner, ok := w.InnerAt(g.Contact.X)
		require.True(t, ok, "tab %d over the wall", i)
		widen := g.Contact.Y + w.Side*(turbinefan.TabNeckLength+turbinefan.TabWidth/2-turbinefan.GateNeckRadius)
		require.GreaterOrEqual(t, w.Side*(inner-widen), turbinefan.TabWideningClearance-1e-9, "tab %d widening", i)
	}
	for i, q := range h.KeelTop {
		require.GreaterOrEqual(t, q[1], turbinefan.HarnessPlateThickness+turbinefan.KeelMinRise, "keel top %d", i)
	}
	for i, g := range h.Gates[12:] {
		require.True(t, insideFootprint(h.KeelFootprint, g.Contact.X, g.Contact.Y), "pin %d on the keel", 12+i)
	}
	return h
}
