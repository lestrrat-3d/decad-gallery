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
// |u| <= 9.9 at elevations 90 / 45 / 30 degrees and yellow at least 65 % of those; unlit, no ray hits a yellow body. Azimuths
// along u and the whole-window share are reported, not asserted.
func TestSevenSegmentVisibility(t *testing.T) {
	startDigit0()
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
	poses := []pose{
		{"lit", true, []float64{29.39, 45, 63.12}},
		{"unlit", false, []float64{-66.7, -55, -41.22, -37.90}},
	}
	elevs := []float64{90, 45, 30}
	litFlag := map[float64]float64{90: 0.90, 45: 0.55, 30: 0.20}
	for _, s := range sevensegment.Segments() {
		yMesh, _ := meshOf(t, parts.Flags[s].Yellow, 0.05)
		bMesh, _ := meshOf(t, parts.Flags[s].Black, 0.05)
		point, axis := segmentAxis(s)
		sAxis := r3.Vec{Z: 1}.Cross(axis) // s = Z x u
		uAxis := axis
		for _, ps := range poses {
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
// shoulders of the pointed ends. Both are under 0.8 and listed as an exception in instructions.md.
func TestSevenSegmentWindowLip(t *testing.T) {
	startDigit0()
	t.Parallel()
	parts := builtDigit0(t)
	// 0.015: the pivot pockets' rounds meet the bore's faceted cone, and decad holds that body's mesh to 0.0148 at best.
	topMesh, _ := meshOf(t, parts.TopPlate, 0.015)
	plate := trisOf(topMesh, nil, r3.Vec{}, kindPlate)
	var near []rayTri
	for _, tr := range plate {
		if tr.hi.X > -14 && tr.lo.X < 14 && tr.hi.Y > -6 && tr.lo.Y < 6 && tr.hi.Z > 82 {
			near = append(near, tr)
		}
	}
	const windowHalf, windowChamfer, windowEnd = 2.95, 12.65, 11.696
	long, shoulder := math.Inf(1), math.Inf(1)
	for u := 0.0; u <= windowEnd; u += 0.01 {
		sv := math.Min(windowHalf, windowChamfer-u) + 0.01
		dist, kind := firstHitDist(near, r3.Vec{X: u, Y: sv, Z: 83}, r3.Vec{X: 1e-9, Y: 1e-9, Z: 1})
		require.Equal(t, kindPlate, kind, "u %.2f", u)
		lip := sevensegment.FaceZ - (83 + dist)
		if u < 9.0 {
			long = math.Min(long, lip)
		} else if u >= 9.7 && u <= 10.7 {
			shoulder = math.Min(shoulder, lip)
		}
	}
	t.Logf("lip along the long sides %.3f, at the shoulders %.3f", long, shoulder)
	require.InDelta(t, 0.704, long, 0.02)
	require.InDelta(t, 0.49, shoulder, 0.02)
}

// TestSevenSegmentWindowYellow casts rays through the window opening of a type A rotor (segment g) and a type B rotor
// (segment b) in the model pose (the flag and the plate move together in the physical geometry). The opening is read from
// the plate mesh on a 0.05 grid over |s| <= 4, offset by half a step, so a wider window shows more.
//
// Lit, the yellow fills at least 0.95 of the opening at 90 degrees at phi 45, 0.82 at the lit band ends 34.47 (a, g, d) and
// 33.41 (b to f) and 0.76 at the b-f worst-case end 29.39 (computed 0.98, 0.85 to 0.87, 0.80).
//
// Unlit, no ray hits yellow at the design-play band end (a, g, d -40.33, b to f -41.22), the worst-case end (-39.23, -37.90)
// and a 1 degree margin beyond it (-38.23, -36.90), with the rotor centred, 0.16 or 0.25 to either side and 0.30 along the
// axis either way, from elevations 90, 45 and 30 at four azimuths. At the margin poses with the 0.25 side shift the views
// also include elevations 60 and 20 at 24 azimuths, grazing views included: a yellow face seen edge-on still counts.
func TestSevenSegmentWindowYellow(t *testing.T) {
	startDigit0()
	t.Parallel()
	parts := builtDigit0(t)
	dm := parts.Dimensions
	topMesh := rawMeshOf(t, parts.TopPlate, 0.05)
	plate := trisOf(topMesh, nil, r3.Vec{}, kindPlate)
	for _, c := range []struct {
		seg    sevensegment.Segment
		unlit  [3]float64 // design end, worst-case end, 1 degree margin
		lit    map[float64]float64
		litEnd string
	}{
		{sevensegment.SegG, [3]float64{-40.33, -39.23, -38.23}, map[float64]float64{45: 0.95, 34.47: 0.82, 33.20: 0.82}, "a, g, d"},
		{sevensegment.SegB, [3]float64{-41.22, -37.90, -36.90}, map[float64]float64{45: 0.95, 33.41: 0.82, 29.39: 0.76}, "b to f"},
	} {
		s := c.seg
		yMesh := rawMeshOf(t, parts.Flags[s].Yellow, 0.02)
		bMesh := rawMeshOf(t, parts.Flags[s].Black, 0.02)
		point, axis := segmentAxis(s)
		sAxis := r3.Vec{Z: 1}.Cross(axis)
		cx, cy := point.X, point.Y
		var near []rayTri
		for _, tr := range plate {
			if tr.hi.X < cx-26 || tr.lo.X > cx+26 || tr.hi.Y < cy-26 || tr.lo.Y > cy+26 || tr.hi.Z < 82 {
				continue
			}
			near = append(near, tr)
		}
		entries := openingEntries(newRayTree(near), s, sevensegment.FaceZ)
		// The opening, read from the mesh, is the designed window: 5.9 wide, 130.0 mm2.
		maxS := 0.0
		for _, q := range entries {
			maxS = math.Max(maxS, math.Abs((r3.Vec{X: q[0], Y: q[1]}).Sub(r3.Vec{X: cx, Y: cy}).Dot(sAxis)))
		}
		require.InDelta(t, 2.95, maxS, 0.03, "segment %s: the window's half-width on the mesh", s)
		require.InDelta(t, 130.0, float64(len(entries))*0.0025, 1.0, "segment %s: the window's area on the mesh", s)
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
			for _, ds := range []float64{0, 0.16, -0.16, 0.25, -0.25} {
				for _, du := range []float64{0, 0.30, -0.30} {
					jobs = append(jobs, job{phi, ds, du, pi == 2 && math.Abs(ds) == 0.25})
				}
			}
		}
		counts := make([]int, len(jobs))
		var wg sync.WaitGroup
		sem := make(chan struct{}, 20)
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
