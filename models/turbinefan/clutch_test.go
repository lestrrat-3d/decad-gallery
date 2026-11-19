package turbinefan_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

// TestTurbineFanClutchBuild checks the clutch mount's 14 bodies and its four containers. The adapter and the clutch are
// the caller's to build, so the fan's document holds neither.
func TestTurbineFanClutchBuild(t *testing.T) {
	eachClutchCase(t, func(t *testing.T, _ fanCase, m *turbinefan.Model, doc *decad.Document) {
		require.Len(t, doc.Bodies(), 14)
		require.NoError(t, doc.VerifyContainers().Err())
		require.Nil(t, m.SetScrew, "no shaft in the hub, so no set screw")
		require.Nil(t, m.SetScrewNut)
		want := map[string][]*decad.Body{
			turbinefan.HubContainerName:    {m.Hub},
			turbinefan.BladesContainerName: {m.Blades[0], m.Blades[1], m.Blades[2]},
			turbinefan.BoltsContainerName: {m.BladeBolts[0].Body, m.BladeBolts[1].Body, m.BladeBolts[2].Body,
				m.AdapterBolts[0].Body, m.AdapterBolts[1].Body},
			turbinefan.NutsContainerName: {m.BladeNuts[0].Body, m.BladeNuts[1].Body, m.BladeNuts[2].Body,
				m.AdapterNuts[0].Body, m.AdapterNuts[1].Body},
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

// TestTurbineFanClutchSketches requires 29 fully constrained sketches from BuildParts, the hub's own among them, and the
// adapter fasteners' own.
func TestTurbineFanClutchSketches(t *testing.T) {
	eachClutchCase(t, func(t *testing.T, _ fanCase, m *turbinefan.Model, _ *decad.Document) {
		require.Equal(t, 29, len(m.Sketches))
		for i, s := range m.Sketches {
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
		}
		for i, b := range m.AdapterBolts {
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), b.Profile), "adapter bolt %d profile", i+1)
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), b.Socket), "adapter bolt %d socket", i+1)
		}
		for i, n := range m.AdapterNuts {
			for _, s := range []*sketch.Sketch{n.Hexagon, n.LowerChamfer, n.UpperChamfer} {
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "adapter nut %d", i+1)
			}
		}
	})
}

// TestTurbineFanClutchHub checks that the clutch mount's hub is the ShaftMount hub of the same shaft: the same volume and
// bounds, and the same cuts for the set screw and the adapter bolts.
func TestTurbineFanClutchHub(t *testing.T) {
	for clutch, shaft := range map[string]string{"clutch-605ZZ": "605ZZ", "clutch-608ZZ-left": "608ZZ"} {
		t.Run(clutch, func(t *testing.T) {
			cm, _ := built(t, caseNamed(t, clutch))
			sm, _ := built(t, caseNamed(t, shaft))
			cv, err := cm.Hub.Volume()
			require.NoError(t, err)
			sv, err := sm.Hub.Volume()
			require.NoError(t, err)
			require.InDelta(t, sv.Value.Base(), cv.Value.Base(), 1e-6*sv.Value.Base())
			cb, err := cm.Hub.Bounds()
			require.NoError(t, err)
			sb, err := sm.Hub.Bounds()
			require.NoError(t, err)
			tol := cb.Bound.Base() + sb.Bound.Base() + 1e-6
			require.InDelta(t, sb.Min.X, cb.Min.X, tol)
			require.InDelta(t, sb.Max.X, cb.Max.X, tol)
			require.InDelta(t, sb.Min.Y, cb.Min.Y, tol)
			require.InDelta(t, sb.Max.Y, cb.Max.Y, tol)
			require.InDelta(t, 0, cb.Min.Z, tol)
			require.InDelta(t, cm.Dimensions.Size.HubLength, cb.Max.Z, tol)
			require.InDelta(t, cm.Dimensions.Size.HubRadius, cb.Max.X, tol, "no flange: the hub is a plain cylinder")
			require.True(t, meshOf(t, cm.Hub, 0.05).BoundaryVerified())
		})
	}
}

// TestTurbineFanClutchAccess checks the blades against the clutch mount: the blades keep clear of the adapter's front face
// and of each ear bolt's line, so a 1.5 mm hex key reaches every ear bolt head from the front between the blades, and the
// clutch mount reuses the ShaftMount blades. The hub's rim keeps HeadPocketRadius + 0.5 clear of every ear bolt line too,
// so a head and a key pass the hub into the adapter's ear pockets.
func TestTurbineFanClutchAccess(t *testing.T) {
	earLo, earHi := math.Inf(1), math.Inf(-1)
	for _, shaft := range onewayclutch.Shafts() {
		for _, c := range []float64{0.05, 0.5} {
			p := onewayclutch.DefaultParams()
			p.Shaft, p.BoreClearance = shaft.Name, c
			dm, err := p.Derive()
			require.NoError(t, err)
			earLo, earHi = math.Min(earLo, dm.EarCircle), math.Max(earHi, dm.EarCircle)
		}
	}
	// earGap returns the distance from v to the nearest ear bolt line, on the plane, over every ear circle in earLo to
	// earHi.
	earGap := func(v r3.Vec, e int) float64 {
		a := turbinefan.EarBoltAngle(e)
		lo := r3.Vec{X: earLo * math.Cos(a), Y: earLo * math.Sin(a)}
		hi := r3.Vec{X: earHi * math.Cos(a), Y: earHi * math.Sin(a)}
		seg := hi.Sub(lo)
		u := math.Max(0, math.Min(1, v.Sub(lo).Dot(seg)/seg.Dot(seg)))
		line := lo.Add(seg.Scale(u))
		return math.Hypot(v.X-line.X, v.Y-line.Y)
	}
	eachCase(t, pitchCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		sz := m.Dimensions.Size
		hubGap := math.Inf(1)
		for _, v := range meshOf(t, m.Hub, 0.05).Vertices() {
			for e := 1; e <= onewayclutch.Ears; e++ {
				hubGap = math.Min(hubGap, earGap(v, e))
			}
		}
		t.Logf("%s: the hub's least gap to an ear bolt line is %.3f", c.name, hubGap)
		require.GreaterOrEqual(t, hubGap, m2fastener.HeadPocketRadius+0.5-1e-6, "hub rim to the ear bolt path")
		for k, blade := range m.Blades {
			for _, v := range meshOf(t, blade, 0.1).Vertices() {
				// The adapter's front face lies at z -0.25 in the fan frame, so a blade at z 0 or more is 0.25 clear of it.
				require.GreaterOrEqual(t, v.Z, 0.0, "blade %d at %v: behind the hub's back face", k+1, v)
				for e := 1; e <= onewayclutch.Ears; e++ {
					// The ear bolt sits at EarCircle, which grows with the 8 mm clutch's bore clearance: keep clear of the
					// whole path from the 605ZZ clutch's ear circle to the largest one.
					require.False(t, earGap(v, e) < m2fastener.HeadPocketRadius+0.5 && v.Z >= 0 && v.Z <= sz.HubLength+5,
						"blade %d vertex %v blocks ear bolt %d's path", k+1, v, e)
				}
			}
		}
	})
	fan, _ := built(t, caseNamed(t, "clutch-605ZZ"))
	ref, _ := built(t, caseNamed(t, "605ZZ"))
	for k := range fan.Blades {
		a, err := fan.Blades[k].Volume()
		require.NoError(t, err)
		b, err := ref.Blades[k].Volume()
		require.NoError(t, err)
		require.InDelta(t, b.Value.Base(), a.Value.Base(), 1e-6*b.Value.Base(), "blade %d", k+1)
		requireAccess(t, fan.Dimensions.Size, meshOf(t, fan.Blades[k], 0.1).Vertices(), k+1)
	}
}
