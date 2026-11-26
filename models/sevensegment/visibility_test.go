package sevensegment_test

// CI: isolated-tests

import (
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentVisibility casts rays at the real meshes in the physical geometry (clamped and settled: the cradle and
// the flags 0.25 lower and the flags a further 0.25 lower, the top plate 0.50 lower) with every flag turned to the band
// ends and to the nominal rests. Lit, across the flag, the flag fills at least 90 / 55 / 20 % of the rays that enter at
// |u| <= 14.85 at elevations 90 / 45 / 30 degrees and yellow at least 65 % of those; unlit, no ray hits a yellow body. Azimuths
// along u and the whole-window share are reported, not asserted.
func TestSevenSegmentVisibility(t *testing.T) {
	t.Parallel()
	parts := builtDigit0(t)
	dm := parts.Dimensions
	topMesh, _ := meshOf(t, parts.TopPlate, 0.05)
	// The top plate and the flags sit 0.50 lower than modeled: the flags' 0.25 clamp and 0.25 settle against the plate's 0.50.
	shift := r3.Vec{Z: -0.5}
	faceZ := sevensegment.FaceZ - 0.5
	plate := trisOf(topMesh, nil, shift, kindPlate)
	type pose struct {
		name string
		lit  bool
		phis []float64
	}
	// The outer band ends are those of the cone's greatest slot clearance (0.481, its full withdrawal): 62.05 and -65.41 for every
	// class. The inner ends are those of the least (0.200). Every end comes from bands, as the
	// rest band test reads them.
	c := dwellOffset
	_, uwDesign, _, _, _ := bands(sevensegment.PinLinePlay, 0.24, slotClearMin, c)
	posesOf := func(s sevensegment.Segment) []pose {
		z := zPlayBF
		if _, _, horizontal := segmentCentreOf(s); horizontal {
			z = zPlayAGD
		}
		_, uwWorst, lwWorst, _, _ := bands(sevensegment.PinLinePlay, z, slotClearMin, c)
		unlit, _, _, lit, _ := bands(sevensegment.PinLinePlay, 0.24, slotClearMax, c)
		return []pose{
			{"lit", true, []float64{lwWorst, 45, lit}},
			{"unlit", false, []float64{unlit, -55, uwDesign, uwWorst}},
		}
	}
	elevs := []float64{90, 45, 30}
	litFlag := map[float64]float64{90: 0.90, 45: 0.55, 30: 0.20}
	for _, s := range sevensegment.Segments() {
		yMesh, _ := meshOf(t, parts.Flags[s].Yellow, 0.05)
		bMesh, _ := meshOf(t, parts.Flags[s].Black, 0.05)
		point, axis := segmentAxis(s)
		sAxis := r3.Vec{Z: 1}.Cross(axis) // s = Z x u
		uAxis := axis
		for _, ps := range posesOf(s) {
			for _, phi := range ps.phis {
				turn, err := r3.RotationAround(point, axis, units.Degrees(phi-dm.FlagAngle(s, 0)))
				require.NoError(t, err)
				tris := append(append([]rayTri{}, plate...), trisOf(yMesh, &turn, shift, kindYellow)...)
				tris = append(tris, trisOf(bMesh, &turn, shift, kindBlack)...)
				tree := newRayTree(windowNear(tris, s))
				for _, elev := range elevs {
					for _, dir := range []r3.Vec{sAxis, sAxis.Scale(-1)} {
						flag, yellow := windowShares(tree, s, faceZ, elev, dir)
						if ps.lit {
							require.GreaterOrEqual(t, flag, litFlag[elev], "segment %s lit phi %.1f elevation %.0f flag share", s, phi, elev)
							require.GreaterOrEqual(t, yellow, 0.65, "segment %s lit phi %.1f elevation %.0f yellow share", s, phi, elev)
						} else {
							require.Zero(t, yellow, "segment %s unlit phi %.1f elevation %.0f shows yellow", s, phi, elev)
						}
					}
					if s == sevensegment.SegG && ps.lit && phi == 45 {
						flag, yellow := windowShares(tree, s, faceZ, elev, uAxis)
						t.Logf("segment g lit phi 45 elevation %.0f along +u: flag %.2f yellow %.2f (reported, not asserted)", elev, flag, yellow)
					}
				}
			}
		}
	}
}

// TestSevenSegmentWindowLip measures the plate's lip under the window edge of segment g, on the plate mesh, by a ray
// going up from inside the bore 0.01 outside the edge: 0.704 along the long sides (half-width 2.95), and 0.49 at the
// shoulders of the pointed ends. Both pass the 0.8 minimum.
func TestSevenSegmentWindowLip(t *testing.T) {
	t.Parallel()
	parts := builtDigit0(t)
	// 0.025: the pivot pockets' rounds meet the bore's faceted cone, and decad holds that body's mesh to 0.0208 at best.
	topMesh, _ := meshOf(t, parts.TopPlate, 0.025)
	plate := trisOf(topMesh, nil, r3.Vec{}, kindPlate)
	var near []rayTri
	for _, tr := range plate {
		if tr.hi.X > -21.0 && tr.lo.X < 21.0 && tr.hi.Y > -9.0 && tr.lo.Y < 9.0 && tr.hi.Z > sevensegment.TopFrom-0.85 {
			near = append(near, tr)
		}
	}
	windowHalf, windowChamfer, windowEnd := sevensegment.WindowHalfWidth, sevensegment.WindowChamfer, sevensegment.WindowEnd
	rayZ := sevensegment.TopFrom + 0.15 // inside the bore
	long, shoulder := math.Inf(1), math.Inf(1)
	for u := 0.0; u <= windowEnd; u += 0.01 {
		sv := math.Min(windowHalf, windowChamfer-u) + 0.01
		dist, kind := firstHitDist(near, r3.Vec{X: u, Y: sv, Z: rayZ}, r3.Vec{X: 1e-9, Y: 1e-9, Z: 1})
		require.Equal(t, kindPlate, kind, "u %.2f", u)
		lip := sevensegment.FaceZ - (rayZ + dist)
		if u < 13.5 {
			long = math.Min(long, lip)
		} else if u >= 14.55 && u <= 16.05 {
			shoulder = math.Min(shoulder, lip)
		}
	}
	t.Logf("lip along the long sides %.3f, at the shoulders %.3f", long, shoulder)
	// The lips are 1.512 and 1.248, above the 0.8 minimum.
	require.InDelta(t, 1.512, long, 0.03)
	require.InDelta(t, 1.248, shoulder, 0.03)
}

// TestSevenSegmentWindowYellow casts rays through the window opening of a type A rotor (segment g) and a type B rotor
// (segment b) in the model pose (the flag and the plate move together in the physical geometry). The opening is read from
// the plate mesh on a 0.05 grid over |s| <= 4, offset by half a step, so a wider window shows more.
//
// Lit, the yellow fills at least 0.95 of the opening at 90 degrees at phi 45, 0.82 at the lit design end 36.94 and 0.76 at the worst-case
// ends 34.17 (a, g, d) and 34.34 (b to f).
//
// Unlit, no ray hits yellow at the design-play band end (-42.14), the worst-case end (-39.74 for a, g, d, -39.89 for b to f)
// and a 1 degree margin beyond it (-38.74, -38.89), with the rotor centred, 0.186 (the rotor fully across its pins, plus 0.05)
// or 0.25 to either side and 0.25 (the pin tips' play, plus 0.05) along the axis either way, from elevations 90, 45 and 30 at four azimuths. At the margin poses with the 0.25 side shift the views
// also include elevations 60 and 20 at 24 azimuths, grazing views included: a yellow face seen edge-on still counts.
func TestSevenSegmentWindowYellow(t *testing.T) {
	t.Parallel()
	parts := builtDigit0(t)
	dm := parts.Dimensions
	topMesh := rawMeshOf(t, parts.TopPlate, 0.05)
	plate := trisOf(topMesh, nil, r3.Vec{}, kindPlate)
	c := dwellOffset
	// ends returns the unlit design end, the unlit worst-case end and the lit ends, as the rest band test reads them.
	ends := func(z, c float64) (unlit [3]float64, lit [2]float64) {
		_, uwDesign, lwDesign, _, _ := bands(sevensegment.PinLinePlay, 0.24, slotClearMin, c)
		_, uwWorst, lwWorst, _, _ := bands(sevensegment.PinLinePlay, z, slotClearMin, c)
		return [3]float64{uwDesign, uwWorst, uwWorst + 1}, [2]float64{lwDesign, lwWorst}
	}
	unlitG, litG := ends(zPlayAGD, c)
	unlitB, litB := ends(zPlayBF, c)
	sideShift, axialShift := sevensegment.JournalAcross+0.05, sevensegment.PivotEndPlay+0.05
	for _, c := range []struct {
		seg    sevensegment.Segment
		unlit  [3]float64 // design end, worst-case end, 1 degree margin
		lit    map[float64]float64
		litEnd string
	}{
		{sevensegment.SegG, unlitG, map[float64]float64{45: 0.95, litG[0]: 0.82, litG[1]: 0.76}, "a, g, d"},
		{sevensegment.SegB, unlitB, map[float64]float64{45: 0.95, litB[0]: 0.82, litB[1]: 0.76}, "b to f"},
	} {
		s := c.seg
		yMesh := rawMeshOf(t, parts.Flags[s].Yellow, 0.02)
		bMesh := rawMeshOf(t, parts.Flags[s].Black, 0.02)
		point, axis := segmentAxis(s)
		sAxis := r3.Vec{Z: 1}.Cross(axis)
		cx, cy := point.X, point.Y
		var near []rayTri
		for _, tr := range plate {
			if tr.hi.X < cx-39.0 || tr.lo.X > cx+39.0 || tr.hi.Y < cy-39.0 || tr.lo.Y > cy+39.0 || tr.hi.Z < sevensegment.TopFrom-0.85 {
				continue
			}
			near = append(near, tr)
		}
		entries := openingEntries(newRayTree(near), s, sevensegment.FaceZ)
		// The opening, read from the mesh, is the designed window: 9.65 wide, 322.5 mm2.
		maxS := 0.0
		for _, q := range entries {
			maxS = math.Max(maxS, math.Abs((r3.Vec{X: q[0], Y: q[1]}).Sub(r3.Vec{X: cx, Y: cy}).Dot(sAxis)))
		}
		require.InDelta(t, sevensegment.WindowHalfWidth, maxS, 0.03, "segment %s: the window's half-width on the mesh", s)
		require.InDelta(t, 322.5, float64(len(entries))*0.0025, 1.5, "segment %s: the window's area on the mesh", s)
		t.Logf("segment %s (%s): the opening holds %d entry points (%.1f mm2)", s, c.litEnd, len(entries), float64(len(entries))*0.0025)
		pose := func(phi, ds, du float64) []rayTri {
			turn, err := r3.RotationAround(point, axis, units.Degrees(phi-dm.FlagAngle(s, 0)))
			require.NoError(t, err)
			shift := sAxis.Scale(ds).Add(axis.Scale(du))
			tris := append([]rayTri{}, near...)
			tris = append(tris, trisOf(yMesh, &turn, shift, kindYellow)...)
			return indexTris(append(tris, trisOf(bMesh, &turn, shift, kindBlack)...))
		}
		for phi, want := range c.lit {
			tris := pose(phi, 0, 0)
			share := 0.0
			for _, dir := range []r3.Vec{sAxis, sAxis.Scale(-1)} {
				share = math.Max(share, float64(windowRays(tris, entries, sevensegment.FaceZ, 90, dir))/float64(len(entries)))
			}
			t.Logf("segment %s lit phi %.2f: yellow share of the opening %.3f", s, phi, share)
			require.GreaterOrEqual(t, share, want, "segment %s lit phi %.2f", s, phi)
		}
		type job struct {
			phi, ds, du float64
			wide        bool // elevations 90, 60, 45, 30 and 20 at 24 azimuths
		}
		var jobs []job
		for pi, phi := range c.unlit {
			for _, ds := range []float64{0, sideShift, -sideShift, 0.25, -0.25} {
				for _, du := range []float64{0, axialShift, -axialShift} {
					jobs = append(jobs, job{phi, ds, du, pi == 2 && math.Abs(ds) == 0.25})
				}
			}
		}
		counts := make([]int, len(jobs))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 4)
		for i, j := range jobs {
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				tris := pose(j.phi, j.ds, j.du)
				elevs, step := []float64{90, 45, 30}, 90.0
				if j.wide {
					elevs, step = []float64{90, 60, 45, 30, 20}, 15.0
				}
				for _, elev := range elevs {
					for az := 0.0; az < 360; az += step {
						a := az * math.Pi / 180
						dir := sAxis.Scale(math.Cos(a)).Add(axis.Scale(math.Sin(a)))
						counts[i] += windowRays(tris, entries, sevensegment.FaceZ, elev, dir)
					}
				}
			}()
		}
		wg.Wait()
		for i, j := range jobs {
			require.Zero(t, counts[i], "segment %s unlit phi %.2f with side shift %+.2f and axial shift %+.2f: yellow rays", s, j.phi,
				j.ds, j.du)
		}
	}
}
