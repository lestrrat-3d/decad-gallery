package handcranksiren_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// requireCorners applies "Corners and roots" to every printed part in its print orientation: the horn and the grip
// on their X caps, the crank pin's halves on their cut faces (print axis the split plane's normal, which runs along the
// arm), every other part along Y. Bolts, nuts and bearings are purchased parts. The exemptions match "Retained sharp
// corners" in instructions.md.
func requireCorners(t *testing.T, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	t.Helper()
	state, err := handcranksiren.Motion(p)
	require.NoError(t, err)
	exempt := crankExempt(t, p)
	for _, name := range partNames() {
		if strings.Contains(name, "-bolt") || strings.Contains(name, "-nut") || strings.HasPrefix(name, "bearing-") {
			continue
		}
		axis := r3.Vec{Y: 1}
		switch name {
		case "horn", "grip":
			axis = r3.Vec{X: 1}
		case "crank-pin":
			axis = state.ArmDirection
		}
		t.Run(name, func(t *testing.T) {
			fs, err := cornercheck.Check(t.Context(), parts[name].Body, axis, exempt[name])
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
}

// crankExempt lists the crank's retained corners as models/crankhandle lists them for its 608-2RS crank, read in
// crankhandle's local frame (+Z out of the mechanism, +Y along the arm to the pin): the arm's hexagonal pocket, which
// matches the pin's stub (Mating corners), the clamp bolt's head pocket floor rim, and the six corners of each cross
// nut's hexagonal pocket in the pin's halves. No slit end round is listed: at the 1.5 mm slit it reads no finding.
func crankExempt(t *testing.T, p handcranksiren.Params) map[string][]cornercheck.Exempt {
	t.Helper()
	toLocal, err := handcranksiren.CrankLocal(p)
	require.NoError(t, err)
	dm, err := crankhandle.Params6082RS().Derive()
	require.NoError(t, err)
	throw := crankhandle.Params6082RS().Throw
	local := func(near func(r3.Vec) bool) func(r3.Vec) bool {
		return func(v r3.Vec) bool { return near(toLocal.Apply(v)) }
	}
	corner := crankhandle.PocketFlats / math.Sqrt(3)
	floor := crankhandle.ArmRadius - dm.ClampSites[0].Cavity
	return map[string][]cornercheck.Exempt{
		"crank-arm": {{
			Reason: "hexagonal pocket corners",
			Near:   local(cornercheck.Annulus(r3.Vec{Y: throw}, r3.Vec{Z: 1}, corner-0.3, corner+0.3)),
		}, {
			// The clamp pocket's pointed roof and floor slopes lie 30 degrees off the bed, so a layer cuts the 0.6 mm
			// round of the floor's outline at a slant and reads it as a turn over 45 degrees, as in crankhandle.
			Reason: "clamp nut pocket floor round cut at a slant",
			Near: local(func(v r3.Vec) bool {
				return v.X < floor-0.05 && v.X > floor-crankhandle.NutPocketRound-0.1 &&
					math.Hypot(v.Y-dm.ClampY, v.Z-crankhandle.ArmThickness/2) <= m2fastener.NutCavityRadius+0.05
			}),
		}, {
			Reason: "clamp head pocket floor rim",
			Near: local(func(v r3.Vec) bool {
				return math.Abs(v.X-(crankhandle.ArmRadius-m2fastener.HeadPocketDepth)) < 0.05 &&
					math.Hypot(v.Y-dm.ClampY, v.Z-crankhandle.ArmThickness/2) <= m2fastener.HeadPocketRadius+0.05
			}),
		}},
		"crank-pin": crossPocketCorners(local, throw),
	}
}

// crossPocketCorners lists the six corners of each cross nut pocket in the pin, as crankhandle's crossPocketNear reads
// them: in the pocket's band along the bolt, within the circle through its corners, and within 10 degrees about the
// bolt's line of a corner direction, which points across the pin (30, 90, ... 330 degrees from local Z, so 90 is +X).
// The points are read in crankhandle's frame.
func crossPocketCorners(local func(func(r3.Vec) bool) func(r3.Vec) bool, throw float64) []cornercheck.Exempt {
	var out []cornercheck.Exempt
	for i, z := range crankhandle.CrossZ {
		out = append(out, cornercheck.Exempt{
			Reason: fmt.Sprintf("cross nut pocket %d corners", i+1),
			Near: local(func(v r3.Vec) bool {
				if v.Y < throw-crankhandle.CrossFlat-0.05 ||
					v.Y > throw-crankhandle.CrossFlat+m2fastener.NutCavityHeight+0.05 {
					return false
				}
				if r := math.Hypot(v.X, v.Z-z); r > m2fastener.NutCavityRadius+0.05 || r < 2.2 {
					return false
				}
				deg := math.Abs(math.Atan2(v.X, v.Z-z))*180/math.Pi - 30
				rem := math.Mod(deg+60, 60)
				return math.Min(rem, 60-rem) <= 10
			}),
		})
	}
	return out
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
// layer and the crank arm (requireArmWall).
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
		{"rotor", state.Rotor, -20.50},
	} {
		t.Run(c.name, func(t *testing.T) {
			section, err := parts[c.name].Body.SectionOutline(t.Context(), atY(t, c.y), units.Millimeters(roundFitTolerance))
			require.NoError(t, err)
			wall := boreWall(t, section.Loops, c.axis)
			require.GreaterOrEqual(t, wall, minPressWall-0.01, "%s: wall round the bore at Y %v", c.name, c.y)
		})
	}
	t.Run("crank-arm", func(t *testing.T) { requireArmWall(t, p, parts["crank-arm"].Body) })
}

// requireArmWall measures the crank arm's wall round its D bore as models/crankhandle does: the slit joins the bore to
// the outline, so the bore is the section's points within r_b + 0.1 of the axis outside the slit, read in crankhandle's
// local frame, and the wall runs from them to every edge that is neither the bore nor the slit. It cuts the arm at Y
// 40.0, 45.1, 48.6 and 53.6 (local z 1.4, 6.5, 10 and 15), between the clamp bolts' holes, whose webs over the slit are
// 1.80 mm with the double flat.
func requireArmWall(t *testing.T, p handcranksiren.Params, arm *decad.Body) {
	t.Helper()
	toLocal, err := handcranksiren.CrankLocal(p)
	require.NoError(t, err)
	dm, err := crankhandle.Params6082RS().Derive()
	require.NoError(t, err)
	half := crankhandle.SlitWidth / 2
	for _, y := range []float64{40.0, 45.1, 48.6, 53.6} {
		o, err := arm.SectionOutline(t.Context(), atY(t, y), units.Millimeters(armTolerance))
		require.NoError(t, err)
		bound := o.OffSurface.Base() + 1e-6
		inSlit := func(v r3.Vec) bool { return math.Abs(v.X) <= half+bound && v.Y < 0 }
		onBore := func(v r3.Vec) bool { return math.Hypot(v.X, v.Y) < dm.BoreRadius+0.1 && !inSlit(v) }
		var bore []r3.Vec
		var edges [][2]r3.Vec
		for _, loop := range o.Loops {
			for i := range loop {
				a, b := toLocal.Apply(loop[i]), toLocal.Apply(loop[(i+1)%len(loop)])
				if onBore(a) {
					bore = append(bore, a)
				}
				if !onBore(a) && !onBore(b) && !inSlit(a) && !inSlit(b) {
					edges = append(edges, [2]r3.Vec{a, b})
				}
			}
		}
		require.NotEmpty(t, bore, "Y %v: bore", y)
		wall := math.Inf(1)
		for _, v := range bore {
			for _, e := range edges {
				wall = math.Min(wall, planarSegment(v, e[0], e[1]))
			}
		}
		require.GreaterOrEqual(t, wall, minPressWall-bound, "crank-arm: wall round the bore at Y %v", y)
	}
}

// planarSegment is the distance from v to the segment ab in the local XY plane.
func planarSegment(v, a, b r3.Vec) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	f := 0.0
	if l := dx*dx + dy*dy; l > 0 {
		f = math.Max(0, math.Min(1, ((v.X-a.X)*dx+(v.Y-a.Y)*dy)/l))
	}
	return math.Hypot(v.X-a.X-f*dx, v.Y-a.Y-f*dy)
}
