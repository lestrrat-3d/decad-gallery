package turbinefan_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestTurbineFanBuild(t *testing.T) {
	for _, c := range fanCases {
		t.Run(c.name, func(t *testing.T) {
			m, doc := built(t, c)
			require.Len(t, doc.Bodies(), 12)
			require.NoError(t, doc.VerifyContainers().Err())
			want := map[string][]*decad.Body{
				turbinefan.HubContainerName:    {m.Hub},
				turbinefan.BladesContainerName: {m.Blades[0], m.Blades[1], m.Blades[2]},
				turbinefan.BoltsContainerName: {
					m.BladeBolts[0].Body, m.BladeBolts[1].Body, m.BladeBolts[2].Body, m.SetScrew.Body},
				turbinefan.NutsContainerName: {
					m.BladeNuts[0].Body, m.BladeNuts[1].Body, m.BladeNuts[2].Body, m.SetScrewNut.Body},
			}
			children := doc.Root().Children()
			require.Len(t, children, len(want))
			for _, box := range children {
				bodies, ok := want[box.Name()]
				require.True(t, ok, "unexpected container %q", box.Name())
				require.Equal(t, bodies, box.Bodies(), box.Name())
				for _, b := range bodies {
					require.True(t, b.IsSolid(), box.Name())
				}
			}
		})
	}
}

// TestTurbineFanSketches requires every sketch of every case (the compact, default and large rotors), the hub's and the
// blades' and each bolt's and nut's, fully constrained: 27 from BuildParts (the hub's two adapter hole sketches and two
// adapter slot sketches among them) and 20 from Build. The clutch mount's sketches are in TestTurbineFanClutchSketches.
func TestTurbineFanSketches(t *testing.T) {
	for _, c := range fanCases {
		t.Run(c.name, func(t *testing.T) {
			m, _ := built(t, c)
			require.Len(t, m.Sketches, 27)
			for i, s := range m.Sketches {
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
			}
			for i, b := range m.BladeBolts {
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), b.Profile), "blade bolt %d profile", i+1)
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), b.Socket), "blade bolt %d socket", i+1)
			}
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), m.SetScrew.Profile))
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), m.SetScrew.Socket))
			for i, n := range append(m.BladeNuts[:], m.SetScrewNut) {
				for _, s := range []*sketch.Sketch{n.Hexagon, n.LowerChamfer, n.UpperChamfer} {
					require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "nut %d", i)
				}
			}
		})
	}
}

// TestTurbineFanBounds checks the hub's bounds at the compact (14 mm radius, 20 long), default (18.9, 30) and large
// (18.9, 35) rotors and tessellates the hub, a blade, the set screw and its nut with every verification on.
func TestTurbineFanBounds(t *testing.T) {
	eachCase(t, []string{"compact", "605ZZ", "large"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		sz := m.Dimensions.Size
		box, err := m.Hub.Bounds()
		require.NoError(t, err)
		tol := box.Bound.Base() + 1e-3
		require.InDelta(t, -sz.HubRadius, box.Min.X, tol)
		require.InDelta(t, sz.HubRadius, box.Max.X, tol)
		require.InDelta(t, -sz.HubRadius, box.Min.Y, tol)
		require.InDelta(t, sz.HubRadius, box.Max.Y, tol)
		require.InDelta(t, 0, box.Min.Z, tol)
		require.InDelta(t, sz.HubLength, box.Max.Z, tol)
		if c.name == "compact" {
			// The volumes of the fan before RotorDiameter existed (608ZZ hub, one blade): the 128 mm rotor rebuilds it.
			// Decad 7cb5dd46b60c meshes the boolean operands closer to their surfaces: b11d7e1658 (#3371) chords a
			// round-ended cutter such as a blade socket at half the chord (hub 9703.740630 before, now 0.145 mm^3 less),
			// and 3179532fc3 (#3385) chords a through-cut target no coarser than its own size (blade 1945.996807
			// before, now 0.005 mm^3 more).
			hv, err := m.Hub.Volume()
			require.NoError(t, err)
			bv, err := m.Blades[0].Volume()
			require.NoError(t, err)
			require.InEpsilon(t, 9703.596022, hv.Value.Base(), 1e-6, "hub volume")
			require.InEpsilon(t, 1946.001321, bv.Value.Base(), 1e-6, "blade volume")
		}
		for name, item := range map[string]struct {
			body *decad.Body
			tol  float64
		}{
			"hub": {m.Hub, 0.05}, "blade": {m.Blades[0], 0.1}, "set screw": {m.SetScrew.Body, 0.01},
			"set screw nut": {m.SetScrewNut.Body, 0.01},
		} {
			mesh := meshOf(t, item.body, item.tol)
			require.True(t, mesh.BoundaryVerified(), name)
		}
	})
}

// TestTurbineFanPlacement checks that the placed case's bodies and sites are the default's moved so that the hub's
// front face (z = Size.HubLength) lies on the frame's origin.
func TestTurbineFanPlacement(t *testing.T) {
	local, _ := built(t, caseNamed(t, "605ZZ"))
	placed, _ := built(t, caseNamed(t, "placed"))
	frame := *caseNamed(t, "placed").params().Placement
	xf := anchoredTransform(t, frame, local.Dimensions.Size.HubLength)
	pairs := map[string][2]*decad.Body{
		"hub": {local.Hub, placed.Hub}, "blade 1": {local.Blades[0], placed.Blades[0]},
		"blade 3": {local.Blades[2], placed.Blades[2]}, "blade bolt": {local.BladeBolts[1].Body, placed.BladeBolts[1].Body},
		"blade nut": {local.BladeNuts[2].Body, placed.BladeNuts[2].Body},
		"set screw": {local.SetScrew.Body, placed.SetScrew.Body}, "set screw nut": {local.SetScrewNut.Body, placed.SetScrewNut.Body},
	}
	for name, pair := range pairs {
		lv, err := pair[0].Volume()
		require.NoError(t, err)
		pv, err := pair[1].Volume()
		require.NoError(t, err)
		require.InDelta(t, lv.Value.Base(), pv.Value.Base(), 1e-6*lv.Value.Base(), name)
		lb, err := pair[0].Bounds()
		require.NoError(t, err)
		pb, err := pair[1].Bounds()
		require.NoError(t, err)
		// The frame maps local X to world Y, Y to Z and Z to X.
		tol := lb.Bound.Base() + pb.Bound.Base() + 1e-6
		lo, hi := xf.Apply(lb.Min), xf.Apply(lb.Max)
		require.InDelta(t, math.Min(lo.X, hi.X), pb.Min.X, tol, name)
		require.InDelta(t, math.Max(lo.X, hi.X), pb.Max.X, tol, name)
		require.InDelta(t, math.Min(lo.Y, hi.Y), pb.Min.Y, tol, name)
		require.InDelta(t, math.Max(lo.Z, hi.Z), pb.Max.Z, tol, name)
	}
	require.InDelta(t, 0, placed.SetScrewSite.Head.Sub(xf.Apply(local.SetScrewSite.Head)).Len(), 1e-9)
	require.InDelta(t, 0, placed.SetScrewSite.Dir.Sub(xf.ApplyDir(local.SetScrewSite.Dir)).Len(), 1e-9)
	for i := range local.BladeSites {
		require.InDelta(t, 0, placed.BladeSites[i].Head.Sub(xf.Apply(local.BladeSites[i].Head)).Len(), 1e-9)
		require.InDelta(t, 0, placed.BladeSites[i].Dir.Sub(xf.ApplyDir(local.BladeSites[i].Dir)).Len(), 1e-9)
	}
}

// TestTurbineFanPlacementPoint checks that a ShaftMount hub's front face lies on the frame's origin plane and the hub
// extends Size.HubLength back along the frame's axis.
func TestTurbineFanPlacementPoint(t *testing.T) {
	p := turbinefan.Params605ZZ()
	f, err := r3.NewFrame(r3.Vec{X: 3, Y: 4, Z: 5}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	p.Placement = &f
	m, err := turbinefan.Build(t.Context(), turbinefan.NewDocument(), p)
	require.NoError(t, err)
	b, err := m.Hub.Bounds()
	require.NoError(t, err)
	tol := b.Bound.Base() + 1e-6
	require.InDelta(t, 5, b.Max.Z, tol, "front face on the frame origin")
	require.InDelta(t, 5-m.Dimensions.Size.HubLength, b.Min.Z, tol, "back face HubLength behind it")
	require.InDelta(t, 3, (b.Min.X+b.Max.X)/2, tol, "hub on the axis through the origin")
}

// TestTurbineFanEnvelope requires every blade vertex to lie between the hub's faces (0 <= z <= HubLength) and inside the
// tip radius, at both ends of the pitch range, for the compact, default and large rotors.
func TestTurbineFanEnvelope(t *testing.T) {
	eachCase(t, []string{"steep", "flat", "605ZZ", "compact", "large"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		sz := m.Dimensions.Size
		for k, blade := range m.Blades {
			lo, hi, far := math.Inf(1), math.Inf(-1), 0.0
			for _, v := range meshOf(t, blade, 0.1).Vertices() {
				lo, hi, far = math.Min(lo, v.Z), math.Max(hi, v.Z), math.Max(far, math.Hypot(v.X, v.Y))
			}
			require.GreaterOrEqual(t, lo, 0.0, "blade %d: lowest z", k+1)
			require.LessOrEqual(t, hi, sz.HubLength, "blade %d: highest z", k+1)
			require.LessOrEqual(t, far, sz.TipRadius+0.5, "blade %d: radius", k+1)
			t.Logf("%s blade %d: z from %.2f to %.2f, radius %.2f", c.name, k+1, lo, hi, far)
		}
	})
}
