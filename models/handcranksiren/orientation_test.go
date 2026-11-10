package handcranksiren_test

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func isGearbox(name string) bool {
	return strings.HasPrefix(name, "gearbox-") && !strings.Contains(name, "bolt") && !strings.Contains(name, "nut")
}

// requireOrientation checks the held pose of instructions.md: the horn points forward (-X) and is the frontmost part, the
// horn's axis runs through (throatY, 84), the crank is on the right (+Y), the table is Z = 0 and only the gearbox pieces
// reach it, and the pin lies on the arm.
func requireOrientation(t *testing.T, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	box := map[string]decad.Box{}
	for name, part := range parts {
		box[name] = bounds(t, part)
	}
	minX, maxY := math.Inf(1), math.Inf(-1)
	minXName, maxYName := "", ""
	minZ, minZBound := math.Inf(1), 0.0
	for name, b := range box {
		if b.Min.X < minX {
			minX, minXName = b.Min.X, name
		}
		if b.Max.Y > maxY {
			maxY, maxYName = b.Max.Y, name
		}
		if b.Min.Z < minZ {
			minZ, minZBound = b.Min.Z, b.Bound.Base()
		}
	}
	require.Equal(t, "horn", minXName, "the horn is the frontmost part")
	hornBox := box["horn"]
	require.InDelta(t, -65.90, (hornBox.Min.Y+hornBox.Max.Y)/2, 1e-3+hornBox.Bound.Base(), "the horn's axis is at throatY")
	require.InDelta(t, 84, (hornBox.Min.Z+hornBox.Max.Z)/2, 1e-3+hornBox.Bound.Base(), "the horn's axis is at Z 84")
	require.True(t, strings.HasPrefix(maxYName, "crank-"), "%s is the rightmost part", maxYName)
	require.InDelta(t, 0, minZ, 1e-6+minZBound, "the table is Z = 0")
	for name, b := range box {
		if b.Min.Z < 1e-3+b.Bound.Base() {
			require.True(t, isGearbox(name), "%s reaches the table", name)
		}
	}
	state, err := handcranksiren.Motion(p)
	require.NoError(t, err)
	pin := box["crank-pin"]
	centre := r3.Vec{X: (pin.Min.X + pin.Max.X) / 2, Z: (pin.Min.Z + pin.Max.Z) / 2}
	want := state.Crank.Add(state.ArmDirection.Scale(55))
	require.Less(t, math.Hypot(centre.X-want.X, centre.Z-want.Z), 0.5, "the pin lies on the arm")
}

// requireNoOverlap runs the document's pairwise check. A bolt in its head host and in its own nut is proved in the
// fasteners subtest, and the two meshing pairs share space by design (the mesh subtest proves them).
func requireNoOverlap(t *testing.T, doc *decad.Document, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	names := map[*decad.Body]string{}
	for name, part := range parts {
		names[part.Body] = name
	}
	exempt := map[[2]string]struct{}{}
	add := func(a, b string) { exempt[[2]string{a, b}], exempt[[2]string{b, a}] = struct{}{}, struct{}{} }
	add("crank-gear", "intermediate-pinion")
	add("intermediate-gear", "rotor-pinion")
	fasteners, err := handcranksiren.Fasteners(p)
	require.NoError(t, err)
	for _, f := range fasteners {
		add(f.Bolt, f.HeadHost)
		add(f.Bolt, f.Nut)
	}
	report, err := doc.Verify(t.Context())
	require.NoError(t, err)
	for _, pair := range report.Interferences {
		_, ok := exempt[[2]string{names[pair.A], names[pair.B]}]
		require.True(t, ok, "%s and %s share material", names[pair.A], names[pair.B])
	}
	for _, d := range report.Diagnostics {
		if d.Pair == nil {
			continue
		}
		_, ok := exempt[[2]string{names[d.Pair.A], names[d.Pair.B]}]
		require.True(t, ok, "%s and %s are undecided: %s", names[d.Pair.A], names[d.Pair.B], d.Code)
	}
}

// requireTessellates builds a verified mesh of the parts whose boundary or volume Decad can check.
func requireTessellates(t *testing.T, parts map[string]handcranksiren.Part) {
	for _, name := range []string{"gearbox-front", "gearbox-frame", "gearbox-rear", "drum", "rotor", "stator", "horn",
		"collector", "crank-arm", "crank-handle", "crank-gear", "collector-nut-1"} {
		mesh, err := parts[name].Body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, name)
		require.True(t, mesh.BoundaryVerified(), name)
		require.True(t, mesh.VolumeVerified(), name)
	}
}
