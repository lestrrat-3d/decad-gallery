package handcranksiren_test

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// requireCorners applies "Corners and roots" to every printed part in its print orientation: the horn and the grip
// on their X caps, every other part along Y. Bolts, nuts and bearings are purchased parts. The one exemption matches
// "Retained sharp corners" in instructions.md.
func requireCorners(t *testing.T, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	t.Helper()
	state, err := handcranksiren.Motion(p)
	require.NoError(t, err)
	// The crank arm's hexagonal pocket keeps its six sharp corners to match the pin's stub (Mating corners), as in
	// models/crankhandle. They lie PocketFlats/√3 from the pin axis.
	pin := state.Crank.Add(state.ArmDirection.Scale(crankhandle.Params6082RS().Throw))
	corner := crankhandle.PocketFlats / math.Sqrt(3)
	exempt := map[string][]cornercheck.Exempt{
		"crank-arm": {{
			Reason: "hexagonal pocket corners",
			Near:   cornercheck.Annulus(pin, r3.Vec{Y: 1}, corner-0.3, corner+0.3),
		}},
	}
	for _, name := range partNames() {
		if strings.Contains(name, "-bolt") || strings.Contains(name, "-nut") || strings.HasPrefix(name, "bearing-") {
			continue
		}
		axis := r3.Vec{Y: 1}
		if name == "horn" || name == "grip" {
			axis = r3.Vec{X: 1}
		}
		t.Run(name, func(t *testing.T) {
			fs, err := cornercheck.Check(t.Context(), parts[name].Body, axis, exempt[name])
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
}

// minPressWall is the press-fit wall of "Corners and roots": four 0.4 mm lines.
const minPressWall = 1.6

// pointSegment is the distance from p to the segment ab, in the (X, Z) plane.
func pointSegment(p, a, b r3.Vec) float64 {
	dx, dz := b.X-a.X, b.Z-a.Z
	length := dx*dx + dz*dz
	f := 0.0
	if length > 0 {
		f = math.Max(0, math.Min(1, ((p.X-a.X)*dx+(p.Z-a.Z)*dz)/length))
	}
	return math.Hypot(p.X-a.X-f*dx, p.Z-a.Z-f*dz)
}

// boreWall returns the least distance between the loop of a section that stays nearest axis (the bore) and every other
// loop, measured between the vertices of each and the segments of the other.
func boreWall(t *testing.T, loops [][]r3.Vec, axis r3.Vec) float64 {
	t.Helper()
	require.Greater(t, len(loops), 1, "a section with a bore has at least two loops")
	bore := 0
	var reach float64
	for i, l := range loops {
		_, far := loopDistances(l, axis)
		if i == 0 || far < reach {
			bore, reach = i, far
		}
	}
	least := math.Inf(1)
	between := func(a, b []r3.Vec) {
		for _, v := range a {
			for j := range b {
				least = math.Min(least, pointSegment(v, b[j], b[(j+1)%len(b)]))
			}
		}
	}
	for i, l := range loops {
		if i != bore {
			between(loops[bore], l)
			between(l, loops[bore])
		}
	}
	return least
}

// requirePressFitWalls requires every separate printed part pressed on a shaft to keep at least 1.6 mm of wall round
// its bore (the press-fit clause of "Corners and roots"): the four gears at their mid-planes, the rotor in its vane
// layer and the crank arm.
func requirePressFitWalls(t *testing.T, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	t.Helper()
	state, err := handcranksiren.Motion(p)
	require.NoError(t, err)
	for _, c := range []struct {
		name string
		axis r3.Vec
		y    float64
	}{
		{"crank-gear", state.Crank, 23.6}, {"intermediate-pinion", state.Intermediate, 23.6},
		{"intermediate-gear", state.Intermediate, 15.6}, {"rotor-pinion", state.Rotor, 15.6},
		{"rotor", state.Rotor, -20.50}, {"crank-arm", state.Crank, 40.0},
	} {
		t.Run(c.name, func(t *testing.T) {
			section, err := parts[c.name].Body.SectionOutline(t.Context(), atY(t, c.y), units.Millimeters(roundFitTolerance))
			require.NoError(t, err)
			wall := boreWall(t, section.Loops, c.axis)
			require.GreaterOrEqual(t, wall, minPressWall-0.01, "%s: wall round the bore at Y %v", c.name, c.y)
		})
	}
}
