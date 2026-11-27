package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentSliders builds the three flag slider types at the unlit pose and checks their volumes, their
// drive cone, blade and head, and their corners along the print axis (a, g, d: the -X face down; b-f: the -Y face down).
func TestSevenSegmentSliders(t *testing.T) {
	t.Parallel()
	r := newTestRun(t)
	// The volumes with the 0.5 chamfers of "Surface edge finishing" on the stem faces (the tongue's end outline, now one 27.75 mm
	// tangent run, takes them too). The buttressed cone adds its boss to the leaning frustum it replaces: +5.41 mm3 on a, g
	// and d, +2.49 on b, c, e and f (3185.28 and 3320.84, 3368.79 before). The boss is 5.95 mm3 more than the frustum
	// before the chamfers, and the longer chamfer run takes the rest away.
	want := map[Segment]float64{SegA: 3190.69, SegG: 3190.69, SegD: 3190.69, SegB: 3323.33, SegC: 3323.33, SegE: 3371.26, SegF: 3371.26}
	vols := map[Segment]float64{}
	for _, seg := range Segments() {
		t.Run(seg.String(), func(t *testing.T) {
			_, body := sharedSlider(t, seg)
			require.True(t, body.IsSolid())
			vol, err := body.Volume()
			require.NoError(t, err)
			vols[seg] = vol.Value.Base()
			require.InDelta(t, want[seg], vol.Value.Base(), 2)
			require.LessOrEqual(t, vol.Value.Base(), 3607.875)
			yi := r.dm.Station(seg)
			_, _, horizontal := segmentCentre(seg)
			ef := seg == SegE || seg == SegF
			// Head or arm top 72.8: material just under it and none just over it, away from the blade and tongue.
			// The a, g and d bodies are built raise higher than the others (the sag of the separate pin in its hole).
			dz := 0.0
			if horizontal {
				dz = raise(seg)
			}
			headX, headY := 0.0, yi-6.0
			if !horizontal {
				headX, headY = 21.0, yi
				if ef {
					headX = -21.0
				}
			}
			require.True(t, inBody(t, body, r3.Vec{X: headX, Y: headY, Z: sliderHeadTop + dz - 0.1}), "head top")
			require.False(t, inBody(t, body, r3.Vec{X: headX, Y: headY, Z: sliderHeadTop + dz + 0.1}), "head top")
			// Blade: 1.65 thick (its rotor-side face 0.1 back from the old 1.2) and 10.5 (a, g, d) or 7.2 (b-f) long, 10.8 over the head top.
			bz := sliderHeadTop + dz + 10.8
			if horizontal {
				require.True(t, inBody(t, body, r3.Vec{X: 5.25 - 0.1, Y: yi - 10.8, Z: bz}))
				require.False(t, inBody(t, body, r3.Vec{X: 5.25 + 0.1, Y: yi - 10.8, Z: bz}))
				require.False(t, inBody(t, body, r3.Vec{X: 0, Y: yi - 11.7 - 0.1, Z: bz}))
				require.False(t, inBody(t, body, r3.Vec{X: 0, Y: yi - 10.05 + 0.05, Z: bz}))
				require.True(t, inBody(t, body, r3.Vec{X: 0, Y: yi - 10.05 - 0.05, Z: bz}))
			} else {
				bx := 13.95
				if ef {
					bx = -13.95
				}
				require.True(t, inBody(t, body, r3.Vec{X: bx, Y: yi + 4.5 - 0.1, Z: bz}))
				require.False(t, inBody(t, body, r3.Vec{X: bx, Y: yi + 4.5 + 0.1, Z: bz}))
				require.True(t, inBody(t, body, r3.Vec{X: bx, Y: yi - StemThickness/2 + 0.1, Z: bz}))
				require.False(t, inBody(t, body, r3.Vec{X: bx, Y: yi - StemThickness/2 - 0.1, Z: bz}))
			}
			// The drive cone: its drive axis is (sc, zp); probes on and off its base (Ø4.0 on the tongue face, off the axis), its
			// section at the pocket mouth, its section at two heights between, and its tip.
			var centre, along, wdir r3.Vec
			var ue float64
			switch {
			case horizontal:
				centre, along, wdir, ue = r3.Vec{Y: yi + pinStationAGD, Z: pinAxisZ + raise(seg)}, r3.Vec{X: 1}, r3.Vec{Y: 1}, -2.85
			case ef:
				centre, along, wdir, ue = r3.Vec{X: -pinStationEF, Z: pinAxisZ + raise(seg)}, r3.Vec{Y: 1}, r3.Vec{X: 1}, yi-0.3
			default:
				centre, along, wdir, ue = r3.Vec{X: pinStationBC, Z: pinAxisZ + raise(seg)}, r3.Vec{Y: 1}, r3.Vec{X: 1}, yi-0.3
			}
			perp := r3.Vec{Z: 1}
			at := func(u float64, off r3.Vec) r3.Vec { return centre.Add(along.Scale(u)).Add(off) }
			// The buttress: its base is the hull of the end disc (Ø4.2, off the axis) and the junction circle, flush with the
			// tongue's end outline. Its back is the disc's far side; its front is a vertical wall, the junction circle carried
			// down to the tongue face, with no lean; its back slopes in to the junction plane.
			base := wdir.Scale(toAxis(horizontal) * coneOffsetW).Add(perp.Scale(coneOffsetZ))
			back := base.Scale(1 / coneOffset)
			require.True(t, inBody(t, body, at(ue+0.05, base.Add(back.Scale(coneDiscR-0.15)))), "base back")
			require.False(t, inBody(t, body, at(ue+0.05, base.Add(back.Scale(coneDiscR+0.05)))), "base back radius")
			require.True(t, inBody(t, body, at(ue+0.05, base)), "base centre")
			for _, h := range []float64{0.05, 0.6, coneSpan - coneJoin - 0.05} {
				require.True(t, inBody(t, body, at(ue+h, back.Scale(-(coneJoinR-0.05)))), "front wall at %.2f", h)
				require.False(t, inBody(t, body, at(ue+h, back.Scale(-(coneJoinR+0.05)))), "front wall at %.2f", h)
			}
			require.True(t, inBody(t, body, at(ue+0.05, back.Scale(3.7))), "back slope low")
			require.False(t, inBody(t, body, at(ue+coneSpan-coneJoin-0.05, back.Scale(3.7))), "back slope high")
			// The section shrinks from the junction to the tip: radius coneMouthR at the mouth, coneTipR at the tip's chamfer.
			for _, du := range []float64{coneSpan, coneSpan + 1.0, coneSpan + 2.0, coneSpan + 2.5} {
				rad := coneMouthR - (du-coneSpan)*coneSlope
				require.True(t, inBody(t, body, at(ue+du, perp.Scale(rad-0.05))), "cone section at %.2f", du)
				require.False(t, inBody(t, body, at(ue+du, perp.Scale(rad+0.05))), "cone section at %.2f", du)
				require.True(t, inBody(t, body, at(ue+du, wdir.Scale(rad-0.05))), "cone section at %.2f", du)
				require.False(t, inBody(t, body, at(ue+du, wdir.Scale(rad+0.05))), "cone section at %.2f", du)
			}
			tip := coneSpan - coneJoin + coneLength
			require.True(t, inBody(t, body, at(ue+tip-0.05, r3.Vec{})), "tip")
			require.False(t, inBody(t, body, at(ue+tip+0.05, r3.Vec{})), "tip")
			require.False(t, inBody(t, body, at(ue+tip-0.05, perp.Scale(coneTipR))), "tip chamfer")
			if !horizontal {
				require.InDelta(t, pinAxisZ+map[bool]float64{false: 0.20, true: 0.25}[ef], pinAxisZ+raise(seg), 1e-9)
			}
			up := r3.Vec{Y: 1}
			if horizontal {
				up = r3.Vec{X: 1}
			}
			// The integral groove pin's root and tip of b to f are listed (groovePinExempt); a, g and d have no exemption.
			var ex []cornercheck.Exempt
			if !horizontal {
				ex = groovePinExempt(yi)
			}
			fs, err := cornercheck.Check(t.Context(), body, up, ex)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
	require.InDelta(t, vols[SegA], vols[SegG], 0.01)
	require.InDelta(t, vols[SegA], vols[SegD], 0.01)
	require.InDelta(t, vols[SegB], vols[SegC], 0.01)
	require.InDelta(t, vols[SegE], vols[SegF], 0.01)
}

// TestSevenSegmentBuiltKinematics reads the drive cone of every built slider and checks where it stands against the
// design's values: its drive axis at pinAxisZ (121.375) plus the slider's raise, and at its station beside the stem (2.25 along y for a, g,
// d from the slider's station, x 22.5 for b and c, x -27 for e and f). These are the contract numbers of instructions.md
// "Kinematics and play", read from the mesh: the right cone's rings, on the drive axis from the junction to the tip (the buttress's
// base lies off the axis); a cone built higher or off its line moves the rotor's angles.
func TestSevenSegmentBuiltKinematics(t *testing.T) {
	t.Parallel()
	r := newTestRun(t)
	for _, seg := range Segments() {
		_, body := sharedSlider(t, seg)
		mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.05))
		require.NoError(t, err)
		yi := r.dm.Station(seg)
		_, _, horizontal := segmentCentre(seg)
		lo, hi := r3.Vec{X: 1e9, Y: 1e9, Z: 1e9}, r3.Vec{X: -1e9, Y: -1e9, Z: -1e9}
		for _, v := range mesh.Vertices() {
			var on bool
			w, sc := v.X, pinStationBC
			if seg == SegE || seg == SegF {
				sc = -pinStationEF
			}
			if horizontal {
				on = v.X > -2.85+coneSpan-coneJoin-0.01 && v.Z > pinAxisZ-2.775 && v.Y > yi-1.5
				w, sc = v.Y, yi+pinStationAGD
			} else {
				on = v.Y > yi-0.3+coneSpan-coneJoin-0.01 && v.Z > pinAxisZ-2.775 && math.Abs(v.X) > 18.0
			}
			// The buttress's top section at the junction plane has a 0.09 crescent behind the junction circle: only the
			// right cone's vertices, within the circle, give the axis.
			if !on || math.Hypot(w-sc, v.Z-(pinAxisZ+raise(seg))) > coneJoinR+0.02 {
				continue
			}
			lo = r3.Vec{X: math.Min(lo.X, v.X), Y: math.Min(lo.Y, v.Y), Z: math.Min(lo.Z, v.Z)}
			hi = r3.Vec{X: math.Max(hi.X, v.X), Y: math.Max(hi.Y, v.Y), Z: math.Max(hi.Z, v.Z)}
		}
		require.Less(t, lo.Z, hi.Z, "no cone vertices on slider %s", seg)
		axisZ := (lo.Z + hi.Z) / 2
		require.InDelta(t, pinAxisZ+raise(seg), axisZ, 0.02, "drive axis height of slider %s", seg)
		require.InDelta(t, 2*coneJoinR, hi.Z-lo.Z, 0.05, "cone diameter at the junction of slider %s", seg)
		if horizontal {
			require.InDelta(t, yi+pinStationAGD, (lo.Y+hi.Y)/2, 0.02, "drive axis station of slider %s", seg)
		} else {
			want := pinStationBC
			if seg == SegE || seg == SegF {
				want = -pinStationEF
			}
			require.InDelta(t, want, (lo.X+hi.X)/2, 0.02, "drive axis station of slider %s", seg)
		}
	}
}

// TestSevenSegmentSliderConeOverhang checks that no face of the drive cone points down. With the tongue's cone face on the bed,
// the buttress is a vertical wall in front and a slope behind, so every layer lies inside the one below it: the least upward
// component of a face normal must stay at -0.01 or more (the leaning cone it replaces reached -0.622, 38.4 degrees). The union
// of the buttress and the right cone leaves slivers of the junction circle (under 0.01 mm2 each, a ring about 0.006 wide)
// that face along -u; they are skipped, and the total area of faces past 45 degrees must stay under 0.05 mm2.
func TestSevenSegmentSliderConeOverhang(t *testing.T) {
	t.Parallel()
	r := newTestRun(t)
	for _, seg := range Segments() {
		_, body := sharedSlider(t, seg)
		mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.05))
		require.NoError(t, err)
		_, _, horizontal := segmentCentre(seg)
		uval := func(v r3.Vec) float64 { return v.Y }
		wz := func(v r3.Vec) (float64, float64) { return v.X, v.Z }
		up, ue, sc, zp := r3.Vec{Y: 1}, r.dm.Station(seg)-0.3, pinStationBC, pinAxisZ+raise(seg)
		if seg == SegE || seg == SegF {
			sc = -pinStationEF
		}
		if horizontal {
			uval, wz = func(v r3.Vec) float64 { return v.X }, func(v r3.Vec) (float64, float64) { return v.Y, v.Z }
			up, ue, sc, zp = r3.Vec{X: 1}, -2.85, r.dm.Station(seg)+pinStationAGD, pinAxisZ+raise(seg)
		}
		worst, seen, steep := 1.0, 0, 0.0
		vs := mesh.Vertices()
		for _, tri := range mesh.Triangles() {
			ok := true
			for _, vi := range tri {
				w, z := wz(vs[vi])
				if uval(vs[vi]) < ue-0.01 || math.Hypot(w-sc, z-zp) > coneDiscR+coneOffset+0.1 {
					ok = false
				}
			}
			if !ok {
				continue
			}
			a, b, c := vs[tri[0]], vs[tri[1]], vs[tri[2]]
			n := b.Sub(a).Cross(c.Sub(a))
			if n.Len() < 1e-12 {
				continue
			}
			area, nu := n.Len()/2, n.Scale(1/n.Len()).Dot(up)
			if nu < -0.707 {
				steep += area
			}
			if area < 0.01 {
				continue
			}
			worst = math.Min(worst, nu)
			seen++
		}
		require.Greater(t, seen, 100, "slider %s: cone triangles", seg)
		require.Less(t, steep, 0.05, "slider %s: area of faces past 45 degrees", seg)
		require.GreaterOrEqual(t, worst, -0.01, "slider %s: the cone's steepest face leans %.1f degrees from the print axis", seg,
			math.Asin(-worst)*180/math.Pi)
		t.Logf("slider %s: the cone's most downward normal . up = %.3f (the 45 degree limit is -0.707); %.4f mm2 past it", seg, worst, steep)
	}
}
