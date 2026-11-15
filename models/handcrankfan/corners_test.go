package handcrankfan_test

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// shared is the default fan, built once for the tests that only read it. The build takes about two minutes.
var shared struct {
	once  sync.Once
	doc   *decad.Document
	parts map[string]handcrankfan.Part
	a     *handcrankfan.Assembly
	err   error
}

// sharedDefault returns the default fan. It builds under its own context, so a test that ends does not cancel it.
func sharedDefault(t *testing.T) (*decad.Document, map[string]handcrankfan.Part, *handcrankfan.Assembly) {
	t.Helper()
	shared.once.Do(func() {
		shared.doc = handcrankfan.NewDocument()
		shared.a, shared.err = handcrankfan.Build(context.Background(), shared.doc, handcrankfan.DefaultParams())
		if shared.err != nil {
			return
		}
		shared.parts = map[string]handcrankfan.Part{}
		for _, part := range shared.a.Parts {
			shared.parts[part.Name] = part
		}
	})
	require.NoError(t, shared.err)
	return shared.doc, shared.parts, shared.a
}

// earRootCorners lists the gearbox's pending corners: the root of each ear, where it meets the floor's underside or
// the wall top, and the corners of the fan block against the wall's edge round. They wait for two decad gaps. A side
// root that lies in the tray's end round is refused as "the tangent chain branches or cannot be decided here: 0 edges
// continue it and 2 cannot be decided ... (modify-reach SX2)" at the junction of the end round and the wall's outer
// round. Several ears that share one floor face are refused as "a vertex of the wall lies on an end face's plane".
// Each predicate matches one set of corners from the scan of the default fan, so a corner of another kind in the same
// part is still a finding. Remove each predicate when the rounds land.
//
//   - gearbox-post, X: inside, 90°, y -23.9, at z 101.4 and 190 and x -18.5, -15.3 and -11.8 (the post-side ears).
//   - gearbox-crank-wall, X: inside, 90°, y -23.9, at z 101.4 to 102.08 and 190, x 23.5 to 30.7 (the crank-side ears).
//   - gearbox-fan-wall, Y: inside, 57.5° to 122°, at y -39.6, -38.6 and -34.96 where the ear sides meet the tray, and
//     at y -41.2, x ±14.5, where the bearing block meets the wall's edge round.
func earRootCorners(name string) []cornercheck.Exempt {
	near := func(v, want float64) bool { return math.Abs(v-want) < 0.15 }
	earEnd := func(p r3.Vec) bool {
		return near(p.Y, -23.9) && (near(p.Z, 101.4) || near(p.Z, 190) || (p.Z > 101.3 && p.Z < 102.2))
	}
	switch name {
	case "gearbox-post":
		return []cornercheck.Exempt{{Reason: "post ear roots", Near: earEnd}}
	case "gearbox-crank-wall":
		return []cornercheck.Exempt{{Reason: "crank-side ear roots", Near: earEnd}}
	case "gearbox-fan-wall":
		return []cornercheck.Exempt{
			{Reason: "fan-wall ear sides", Near: func(p r3.Vec) bool {
				return near(p.Y, -39.6) || near(p.Y, -38.6) || near(p.Y, -34.96)
			}},
			{Reason: "fan block against the wall's edge round", Near: func(p r3.Vec) bool { return near(p.Y, -41.2) }},
		}
	}
	return nil
}

// requireCorners applies "Corners and roots" to every printed part in its print orientation (the README's table).
// The exemptions match "Retained sharp corners" in instructions.md. The bolts, nuts and bearings are purchased parts.
// The driving gear's section is read at the one height MidHeights finds, 17.5, which cuts the hub behind the teeth, so
// its teeth are not checked there. The pinion has no run of layers that share one outline, so MidHeights finds no
// height and Check reads nothing of it. TestBevelToothRootCorners reads the teeth of both and records that their roots
// are not rounded. The gearbox's ear roots are exempt by earRootCorners, which keeps this model on CORNER_PENDING in
// scripts/check_repository.py until decad rounds them.
func requireCorners(t *testing.T, p handcrankfan.Params, parts map[string]handcrankfan.Part) {
	t.Helper()
	state, err := handcrankfan.Motion(p)
	require.NoError(t, err)
	fasteners, err := handcrankfan.Fasteners(p)
	require.NoError(t, err)

	x, y, z := r3.Vec{X: 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}
	// The spline bore's three teeth meet its round bore, and the groove floors of the shaft meet the grooves: mating
	// corners at 0.05 press gap. The pin's hexagonal pocket mates with the pin's stub.
	bore := func(centre, axis r3.Vec) []cornercheck.Exempt {
		return []cornercheck.Exempt{{Reason: "spline bore", Near: cornercheck.Annulus(centre, axis, 2.9, 4.2)}}
	}
	grooves := func(centre, axis r3.Vec) []cornercheck.Exempt {
		return []cornercheck.Exempt{{Reason: "groove floor", Near: cornercheck.Annulus(centre, axis, 2.8, 4.1)}}
	}
	pin := state.Crank.Add(state.ArmDirection.Scale(55))
	// A nut cavity whose axis lies in the layer plane shows its two flat ends as corners: mating corners with the nut,
	// which a round would hit. Each lies within 3.2 of its cavity's centre, along the print axis.
	cavities := func(host string, axis r3.Vec) []cornercheck.Exempt {
		var out []cornercheck.Exempt
		for _, f := range fasteners {
			if f.NutHost != host || !strings.HasPrefix(f.Bolt, "gearbox-") {
				continue
			}
			centre := f.Head.Add(f.Dir.Scale((f.CavityNear + f.CavityFar) / 2))
			out = append(out, cornercheck.Exempt{Reason: "nut cavity of " + f.Nut, Near: cornercheck.Disc(centre, axis, 3.2)})
		}
		return out
	}
	type check struct {
		axis   r3.Vec
		exempt []cornercheck.Exempt
	}
	checks := map[string]check{
		"gearbox-post":       {x, append(cavities("gearbox-post", x), earRootCorners("gearbox-post")...)},
		"gearbox-crank-wall": {x, append(cavities("gearbox-crank-wall", x), earRootCorners("gearbox-crank-wall")...)},
		"gearbox-fan-wall":   {y, earRootCorners("gearbox-fan-wall")},
		"gearbox-base":       {y, cavities("gearbox-base", y)},
		"fan-hub":            {y, bore(state.Apex, y)},
		"crank-gear":         {x, bore(state.Crank, x)},
		"drive-gear":         {x, bore(state.Apex, x)},
		"crank-arm": {x, append(bore(state.Crank, x),
			cornercheck.Exempt{Reason: "pin pocket", Near: cornercheck.Annulus(pin, x, 3.4, 4.4)})},
		"driving-gear": {x, bore(state.Apex, x)},
		"crank-shaft":  {x, grooves(state.Crank, x)},
		"drive-shaft":  {x, grooves(state.Apex, x)},
		"fan-shaft":    {y, grooves(state.Apex, y)},
		"crank-pin":    {x, nil},
		"crank-grip":   {x, nil},
	}
	for name, c := range checks {
		t.Run(name, func(t *testing.T) {
			fs, err := cornercheck.Check(t.Context(), parts[name].Body, c.axis, c.exempt)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
	// The grip is a loft with no inner face levels, so MidHeights cuts it only near its top: section it every 4 mm.
	t.Run("handle", func(t *testing.T) {
		var heights []float64
		for h := -18.0; h <= 98; h += 4 {
			heights = append(heights, h)
		}
		fs, err := cornercheck.CheckAt(t.Context(), parts["handle"].Body, z, heights, nil)
		require.NoError(t, err)
		require.Empty(t, fs, cornercheck.Report(fs, 5))
	})
	for k := 1; k <= p.Blades; k++ {
		t.Run(fmt.Sprintf("fan-blade-%d", k), func(t *testing.T) { requireBladeCorners(t, p, state, k, parts) })
	}
}

// requireBladeCorners checks blade k along its radial line d = (cos θ, 0, sin θ), with θ the fan angle plus the
// blade's step. The stud is cut at the radii where its bolt hole crosses it; the hole's four corners (r 9.5 and 10.2)
// are the one exemption, where the material angle is 64° to 90° and the checker adds the stud's curvature. The loft is
// cut at the listed radii, because it has no planar faces to give heights. The exemption claims the hole corners only
// if the radial line is right, so an unused exemption fails the test.
func requireBladeCorners(t *testing.T, p handcrankfan.Params, state handcrankfan.State, k int, parts map[string]handcrankfan.Part) {
	t.Helper()
	theta := state.FanAngle + 2*math.Pi*float64(k-1)/float64(p.Blades)
	d := r3.Vec{X: math.Cos(theta), Z: math.Sin(theta)}
	height := func(r float64) float64 { return state.Apex.Dot(d) + r }
	body := parts[fmt.Sprintf("fan-blade-%d", k)].Body
	// The radial line runs in the hub's mid plane, Y -86 (the model's hubMid), not through the apex at Y 0.
	radial := state.Apex.Add(r3.Vec{Y: -86})

	stud := []float64{height(7.0), height(9.5), height(10.2), height(12.5)}
	fs, err := cornercheck.CheckAt(t.Context(), body, d, stud, []cornercheck.Exempt{
		{Reason: "bolt hole across the stud", Near: cornercheck.Disc(radial, d, 3.1)},
	})
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))

	var loft []float64
	for _, r := range []float64{14.6, 16.5, 18, 22, 26, 30, 36, 39.2, 44, 52, 56, 60, 62, 63, 63.8} {
		loft = append(loft, height(r))
	}
	fs, err = cornercheck.CheckAt(t.Context(), body, d, loft, nil)
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))
}

// toothRoots cuts body, normal to axis through the apex, at each height and checks the bevel teeth's root corners.
// A section holds the spline bore's six corners (r 4.15), where the bore reaches it, and one root corner per tooth on
// the root cone. The cone's
// radius about the shaft at height h is r0 + slope*(h-h0). An empty or short section cannot pass, because it holds
// fewer than the tooth count.
func toothRoots(t *testing.T, body *decad.Body, axis, apex r3.Vec, teeth int, heights []float64, bores func(h float64) int,
	r0, h0, slope, turnMin, turnMax float64) {
	t.Helper()
	for _, h := range heights {
		fs, err := cornercheck.Section(t.Context(), body, axis, h)
		require.NoError(t, err)
		var bore, roots int
		for _, f := range fs {
			d := f.At.Sub(apex)
			r := d.Sub(axis.Scale(d.Dot(axis))).Len()
			switch {
			case f.Kind == cornercheck.Inside && r < 5:
				bore++
			case f.Kind == cornercheck.Inside:
				roots++
				require.InDelta(t, r0+slope*(h-h0), r, 0.15, "root corner off the root cone at %.2f", h)
				require.GreaterOrEqual(t, f.Turn, turnMin, "at %.2f", h)
				require.LessOrEqual(t, f.Turn, turnMax, "at %.2f", h)
			default:
				t.Errorf("unexpected %s at %.2f: r %.2f turn %.1f", f.Kind, h, r, f.Turn)
			}
		}
		require.Equal(t, bores(h), bore, "spline bore corners at %.2f", h)
		require.Equal(t, teeth, roots, "root corners at %.2f: %s", h, cornercheck.Report(fs, 3))
	}
}

// TestBevelToothRootCorners records that the bevel pair's tooth roots have no fillet. Each tooth leaves one sharp
// inside corner of about 83° (driving gear) or 89° (pinion) on its root cone, in every section through the teeth.
// "Corners and roots" does not exempt them, so they are known findings, and the model stays on CORNER_PENDING until
// bevelgear rounds them. The test fails when the roots are rounded or the geometry moves.
func TestBevelToothRootCorners(t *testing.T) {
	_, parts, a := sharedDefault(t)
	t.Run("pinion", func(t *testing.T) {
		// 28 teeth, y -28.25 to -24.25 (the teeth span y -28.75 to -23.75), root cone r 10.61 at y -26.25.
		toothRoots(t, parts["pinion"].Body, r3.Vec{Y: 1}, a.State.Apex, 28,
			[]float64{-28.25, -27.25, -26.25, -25.25, -24.25},
			func(h float64) int { // the spline bore ends at y -26.75
				if h < -26.75 {
					return 6
				}
				return 0
			}, 10.61, -26.25, -0.404, 88, 90.5)
	})
	t.Run("driving-gear", func(t *testing.T) {
		// 56 teeth, x 14.14 to 16.14, root cone r 23.41 at x 14.14.
		toothRoots(t, parts["driving-gear"].Body, r3.Vec{X: 1}, a.State.Apex, 56,
			[]float64{14.14, 15.14, 16.14}, func(float64) int { return 6 }, 23.41, 14.14, 1.655, 82, 84)
	})
}

func TestHandCrankFanCorners(t *testing.T) {
	_, parts, _ := sharedDefault(t)
	requireCorners(t, handcrankfan.DefaultParams(), parts)
}

// minPressWall is the press-fit wall of "Corners and roots": four 0.4 mm lines.
const minPressWall = 1.6

// pointSegment is the distance from p to the segment ab.
func pointSegment(p, a, b r3.Vec) float64 {
	ab := b.Sub(a)
	length := ab.Dot(ab)
	f := 0.0
	if length > 0 {
		f = math.Max(0, math.Min(1, p.Sub(a).Dot(ab)/length))
	}
	return p.Sub(a.Add(ab.Scale(f))).Len()
}

// boreWall returns the least distance between the loop of a section that stays nearest axis (the bore) and every other
// loop, measured between the vertices of each and the segments of the other. axis is a point on the bore's axis in the
// section plane.
func boreWall(t *testing.T, loops [][]r3.Vec, axis r3.Vec) float64 {
	t.Helper()
	require.Greater(t, len(loops), 1, "a section with a bore has at least two loops")
	reach := func(l []r3.Vec) float64 {
		far := 0.0
		for _, v := range l {
			far = math.Max(far, v.Sub(axis).Len())
		}
		return far
	}
	bore := 0
	for i := range loops {
		if reach(loops[i]) < reach(loops[bore]) {
			bore = i
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

// TestPressFitWalls requires at least 1.6 mm of wall round the bore of every separate printed part pressed on a shaft
// (docs/model-pipeline.md "Corners and roots", Root width): the hub through its socket floors, the arm and the two
// spur gears. The bevel pair is not measured here.
func TestPressFitWalls(t *testing.T) {
	_, parts, a := sharedDefault(t)
	for _, c := range []struct {
		name string
		axis r3.Vec
		at   float64 // along the print axis
		x    bool    // the print axis is X, otherwise Y
	}{
		{"fan-hub", a.State.Apex, -86, false},
		{"crank-arm", a.State.Crank, 52, true},
		{"crank-gear", a.State.Crank, -6.2, true},
		{"drive-gear", a.State.Apex, -6.2, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			plane, centre := atY(t, c.at), r3.Vec{X: c.axis.X, Y: c.at, Z: c.axis.Z}
			if c.x {
				plane, centre = atX(t, c.at), r3.Vec{X: c.at, Y: c.axis.Y, Z: c.axis.Z}
			}
			section, err := parts[c.name].Body.SectionOutline(t.Context(), plane, units.Millimeters(fitTolerance))
			require.NoError(t, err)
			wall := boreWall(t, section.Loops, centre)
			require.GreaterOrEqual(t, wall, minPressWall-0.01, "%s: wall round the bore is %.3f", c.name, wall)
		})
	}
}
