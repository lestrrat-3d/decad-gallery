package turbinefan_test

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// clutchGapMesh tessellates body at tol with every verification on and wraps it for gap queries.
func clutchGapMesh(t *testing.T, body *decad.Body, tol float64) meshgap.Mesh {
	t.Helper()
	mesh := meshOf(t, body, tol)
	require.True(t, mesh.BoundaryVerified())
	mg, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return mg
}

// sectionAt cuts body by the plane f at the finest chord from 0.01 mm that the body allows; the faceted hub holds no
// finer mesh than about 0.017.
func sectionAt(t *testing.T, body *decad.Body, f r3.Frame) decad.SectionOutline {
	t.Helper()
	var o decad.SectionOutline
	var err error
	for _, tol := range []float64{0.01, 0.02, 0.03, 0.05} {
		if o, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol)); err == nil {
			require.NotEmpty(t, o.Loops)
			return o
		}
	}
	require.NoError(t, err)
	return o
}

// zRange returns the lowest and highest z over the section's vertices.
func zRange(o decad.SectionOutline) (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, l := range o.Loops {
		for _, v := range l {
			lo, hi = math.Min(lo, v.Z), math.Max(hi, v.Z)
		}
	}
	return lo, hi
}

// TestTurbineFanOnClutch builds the real one-way clutch and the clutch-mount fan placed on it, and checks the ears, the
// stack and every fit pair through the pair. The clutch and the fan are built in separate documents.
func TestTurbineFanOnClutch(t *testing.T) {
	cp := onewayclutch.DefaultParams()
	clutch, err := onewayclutch.Build(t.Context(), onewayclutch.NewDocument(), cp)
	require.NoError(t, err)
	fp, err := turbinefan.ParamsForClutch(cp)
	require.NoError(t, err)
	fan, err := turbinefan.Build(t.Context(), turbinefan.NewDocument(), fp)
	require.NoError(t, err)

	// Pairing and ear alignment.
	require.Equal(t, turbinefan.RightHand, fp.Hand)
	require.Equal(t, cp.Freewheel, fp.Hand.Freewheel())
	for k, site := range fan.EarSites {
		a := onewayclutch.EarAngles[k] * math.Pi / 180
		want := r3.Vec{X: onewayclutch.EarCircle * math.Cos(a), Y: onewayclutch.EarCircle * math.Sin(a), Z: turbinefan.MountZ}
		require.InDelta(t, 0, site.Head.Sub(want).Len(), 1e-9, "ear %d head", k+1)
		require.InDelta(t, 0, site.Dir.Sub(r3.Vec{Z: -1}).Len(), 1e-9, "ear %d direction", k+1)
	}

	// Overlap pass: proves "clear" only. Meshes of printed parts at 0.03 and hardware at 0.05.
	type part struct {
		name string
		body *decad.Body
		tol  float64
	}
	parts := []part{
		{"clutch hub", clutch.Hub, 0.03}, {"clutch ring", clutch.Ring, 0.03}, {"back plate", clutch.BackPlate, 0.03},
		{"cover", clutch.Cover, 0.03}, {"fan hub", fan.Hub, 0.03}, {"clamp", fan.Clamp, 0.03},
	}
	for i := range 3 {
		parts = append(parts,
			part{fmt.Sprintf("clutch bolt %d", i+1), clutch.Bolts[i].Body, 0.05},
			part{fmt.Sprintf("clutch nut %d", i+1), clutch.Nuts[i].Body, 0.05},
			part{fmt.Sprintf("ear bolt %d", i+1), fan.EarBolts[i].Body, 0.05},
			part{fmt.Sprintf("ear nut %d", i+1), fan.EarNuts[i].Body, 0.05})
	}
	slices.SortFunc(parts, func(a, b part) int { return cmp.Compare(a.name, b.name) })
	meshes := map[string]meshgap.Mesh{}
	for _, p := range parts {
		meshes[p.name] = clutchGapMesh(t, p.body, p.tol)
	}
	skip := map[[2]string]struct{}{}
	for i := 1; i <= 3; i++ {
		skip[[2]string{fmt.Sprintf("clutch bolt %d", i), fmt.Sprintf("clutch nut %d", i)}] = struct{}{}
		skip[[2]string{fmt.Sprintf("ear bolt %d", i), fmt.Sprintf("ear nut %d", i)}] = struct{}{}
		skip[[2]string{fmt.Sprintf("ear bolt %d", i), "fan hub"}] = struct{}{}
		skip[[2]string{fmt.Sprintf("clutch bolt %d", i), "cover"}] = struct{}{}
	}
	for i, a := range parts {
		for _, b := range parts[i+1:] {
			_, hit1 := skip[[2]string{a.name, b.name}]
			_, hit2 := skip[[2]string{b.name, a.name}]
			if hit1 || hit2 {
				continue
			}
			_, clear, err := meshes[a.name].Exceeds(meshes[b.name], 0)
			require.NoError(t, err)
			require.True(t, clear, "%s and %s overlap or touch", a.name, b.name)
		}
	}

	// Axial gaps in the plane x = 0, which holds ear 1 and the clutch's 270 degree bolt.
	axial, err := r3.NewFrame(r3.Vec{}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	require.NoError(t, err)
	hub, ring := sectionAt(t, fan.Hub, axial), sectionAt(t, clutch.Ring, axial)
	clamp, cover := sectionAt(t, fan.Clamp, axial), sectionAt(t, clutch.Cover, axial)
	back, bolt := sectionAt(t, clutch.BackPlate, axial), sectionAt(t, clutch.Bolts[2].Body, axial)
	hubLo, _ := zRange(hub)
	ringLo, ringHi := zRange(ring)
	_, clampHi := zRange(clamp)
	_, coverHi := zRange(cover)
	backLo, _ := zRange(back)
	_, boltHi := zRange(bolt)
	off := func(a, b decad.SectionOutline) float64 { return a.OffSurface.Base() + b.OffSurface.Base() }
	for _, g := range []struct {
		name  string
		gap   float64
		least float64
	}{
		{"flange back face / ring front end", hubLo - ringHi - off(hub, ring), 0.20},
		{"clamp front face / ring back end", ringLo - clampHi - off(ring, clamp), 0.20},
		{"flange back face / cover outer face", hubLo - coverHi - off(hub, cover), 1.45},
		{"flange back face / cover bolt head", hubLo - boltHi - off(hub, bolt), 1.45},
		{"clamp front face / back plate outer face", backLo - clampHi - off(back, clamp), 1.45},
	} {
		t.Logf("gap %s: %.3f", g.name, g.gap)
		require.GreaterOrEqual(t, g.gap, g.least, g.name)
	}

	// Ear bolt clearance in sections normal to the bolt, and no overlap with the hub where the head rests.
	for k, eb := range fan.EarBolts {
		for _, c := range []struct {
			name string
			host *decad.Body
			z    float64
		}{
			{"ring", clutch.Ring, 0}, {"ring", clutch.Ring, 9.25}, {"ring", clutch.Ring, 18.5},
			{"clamp", fan.Clamp, -2.0}, {"fan hub", fan.Hub, 20.5},
		} {
			f, err := r3.NewFrame(r3.Vec{Z: c.z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
			require.NoError(t, err)
			ob, oh := sectionAt(t, eb.Body, f), sectionAt(t, c.host, f)
			cl, err := ob.Clearance(oh)
			require.NoError(t, err)
			require.False(t, cl.Overlaps, "ear bolt %d overlaps the %s at z %g", k+1, c.name, c.z)
			require.GreaterOrEqual(t, cl.Distance.Value.Base(), 0.28, "ear bolt %d to the %s at z %g", k+1, c.name, c.z)
		}
		u := fan.EarSites[k].Corner
		planes := []r3.Frame{}
		pf, err := r3.NewFrame(r3.Vec{}, u, r3.Vec{Z: 1})
		require.NoError(t, err)
		planes = append(planes, pf)
		for _, z := range []float64{22, 21.5} {
			f, err := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
			require.NoError(t, err)
			planes = append(planes, f)
		}
		for _, f := range planes {
			requireNoOverlap(t, fan.Hub, f, fmt.Sprintf("ear bolt %d with the fan hub at %v", k+1, f.Origin()), eb.Body)
		}
	}

	// Radial and axial limits.
	for _, v := range meshOf(t, fan.Clamp, 0.05).Vertices() {
		require.GreaterOrEqual(t, math.Hypot(v.X, v.Y), turbinefan.ClampInnerRadius-0.05, "clamp vertex %v", v)
	}
	for _, v := range meshOf(t, fan.Hub, 0.05).Vertices() {
		if math.Hypot(v.X, v.Y) < onewayclutch.RingCounter {
			require.GreaterOrEqual(t, v.Z, onewayclutch.RingTo+turbinefan.ClutchSeam-1e-3, "hub vertex %v", v)
		}
	}
	level, err := r3.NewFrame(r3.Vec{Z: 21.5}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	o := sectionAt(t, fan.Hub, level)
	for _, v := range boreLoop(t, o.Loops, 4) {
		require.GreaterOrEqual(t, math.Hypot(v.X, v.Y), 2.5+0.5-o.OffSurface.Base(), "bore point %v", v)
	}
}
