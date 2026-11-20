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
	doc, blade := posedBlade(t, d, pitch, hand)
	h, err := turbinefan.BuildBladeHarness(t.Context(), doc, blade, mustSize(t, d), hand)
	require.NoError(t, err, "rotor %g, pitch %g, hand %v", d, pitch, hand)
	require.True(t, h.Body.IsSolid())
	require.Len(t, h.Sketches, 14)
	for i, s := range h.Sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
	}
	// Transform retires the blade it moves, so the document holds the posed blade and the harness.
	require.Equal(t, 2, len(doc.Bodies()), "posed blade and harness")
	box, err := h.Body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 0, box.Min.Z, box.Bound.Base()+1e-6)
	head := turbinefan.GateEmbed + turbinefan.GateNeckLength + turbinefan.GateTaper
	for i, g := range h.Gates[:6] {
		require.InDelta(t, -math.Sin(turbinefan.GateRise), g.Dir.Z, 1e-9, "gate %d rise", i)
		require.InDelta(t, 0, g.Dir.X, 1e-9, "gate %d plane", i)
		cone := math.Abs(g.Contact.Y-h.WallY) - head*math.Cos(turbinefan.GateRise) - turbinefan.WallThickness/2 -
			turbinefan.GateRadius*math.Sin(turbinefan.GateRise)
		require.GreaterOrEqual(t, cone, turbinefan.GateConeClearance-1e-9, "gate %d cone clearance", i)
	}
	for i, q := range h.KeelTop {
		require.GreaterOrEqual(t, q[1], turbinefan.HarnessPlateThickness+turbinefan.KeelMinRise, "keel top %d", i)
	}
	for i, g := range h.Gates[6:] {
		require.True(t, insideFootprint(h.KeelFootprint, g.Contact.X, g.Contact.Y), "pin %d on the keel", 6+i)
	}
	return h
}
