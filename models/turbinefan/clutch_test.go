package turbinefan_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func eachClutchCase(t *testing.T, fn func(t *testing.T, c fanCase, m *turbinefan.Model, doc *decad.Document)) {
	t.Helper()
	for _, c := range clutchCases {
		t.Run(c.name, func(t *testing.T) {
			m, doc := built(t, c)
			fn(t, c, m, doc)
		})
	}
}

// TestTurbineFanClutchBuild checks the clutch mount's 17 bodies and its five containers in order.
func TestTurbineFanClutchBuild(t *testing.T) {
	eachClutchCase(t, func(t *testing.T, _ fanCase, m *turbinefan.Model, doc *decad.Document) {
		require.Len(t, doc.Bodies(), 17)
		require.NoError(t, doc.VerifyContainers().Err())
		require.Nil(t, m.SetScrew)
		require.Nil(t, m.SetScrewNut)
		want := map[string][]*decad.Body{
			turbinefan.HubContainerName:    {m.Hub},
			turbinefan.BladesContainerName: {m.Blades[0], m.Blades[1], m.Blades[2]},
			turbinefan.ClampContainerName:  {m.Clamp},
			turbinefan.BoltsContainerName: {m.BladeBolts[0].Body, m.BladeBolts[1].Body, m.BladeBolts[2].Body,
				m.EarBolts[0].Body, m.EarBolts[1].Body, m.EarBolts[2].Body},
			turbinefan.NutsContainerName: {m.BladeNuts[0].Body, m.BladeNuts[1].Body, m.BladeNuts[2].Body,
				m.EarNuts[0].Body, m.EarNuts[1].Body, m.EarNuts[2].Body},
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

// TestTurbineFanClutchSketches requires 33 fully constrained sketches from BuildParts and the ear fasteners' own.
func TestTurbineFanClutchSketches(t *testing.T) {
	eachClutchCase(t, func(t *testing.T, _ fanCase, m *turbinefan.Model, _ *decad.Document) {
		require.Len(t, m.Sketches, 33)
		for i, s := range m.Sketches {
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
		}
		for i, b := range m.EarBolts {
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), b.Profile), "ear bolt %d profile", i+1)
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), b.Socket), "ear bolt %d socket", i+1)
		}
		for i, n := range m.EarNuts {
			for _, s := range []*sketch.Sketch{n.Hexagon, n.LowerChamfer, n.UpperChamfer} {
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "ear nut %d", i+1)
			}
		}
	})
}

// TestTurbineFanClutchBounds checks the hub's and the clamp ring's bounds and tessellates four bodies with every
// verification on.
func TestTurbineFanClutchBounds(t *testing.T) {
	m, _ := built(t, caseNamed(t, "clutch-605ZZ"))
	hub, err := m.Hub.Bounds()
	require.NoError(t, err)
	tol := hub.Bound.Base() + 1e-3
	require.InDelta(t, -turbinefan.FlangeRadius, hub.Min.X, tol)
	require.InDelta(t, turbinefan.FlangeRadius, hub.Max.X, tol)
	require.InDelta(t, -turbinefan.FlangeRadius, hub.Min.Y, tol)
	require.InDelta(t, turbinefan.FlangeRadius, hub.Max.Y, tol)
	require.InDelta(t, -3, hub.Min.Z, tol)
	require.InDelta(t, 20, hub.Max.Z, tol)
	clamp, err := m.Clamp.Bounds()
	require.NoError(t, err)
	tol = clamp.Bound.Base() + 0.01
	require.InDelta(t, -turbinefan.ClampOuterRadius, clamp.Min.X, tol)
	require.InDelta(t, turbinefan.ClampOuterRadius, clamp.Max.X, tol)
	require.InDelta(t, turbinefan.ClampOuterRadius, clamp.Max.Y, tol)
	// The 270 degree slot spans only z -27.5 to -25.5, so the box still reaches the outer radius above and below it;
	// the slot's own section reaches only the wall beside it.
	require.InDelta(t, -turbinefan.ClampOuterRadius, clamp.Min.Y, tol)
	lowest := math.Inf(1)
	for _, l := range section(t, m.Clamp, atZ(t, -26.5)).Loops {
		for _, v := range l {
			lowest = math.Min(lowest, v.Y)
		}
	}
	require.InDelta(t, -math.Sqrt(turbinefan.ClampOuterRadius*turbinefan.ClampOuterRadius-2.17*2.17), lowest, 0.01)
	require.InDelta(t, turbinefan.ClampBack, clamp.Min.Z, tol)
	require.InDelta(t, turbinefan.ClampBack+turbinefan.ClampThickness, clamp.Max.Z, tol)
	for name, item := range map[string]struct {
		body *decad.Body
		tol  float64
	}{
		"hub": {m.Hub, 0.05}, "clamp": {m.Clamp, 0.05}, "ear bolt": {m.EarBolts[0].Body, 0.01},
		"ear nut": {m.EarNuts[0].Body, 0.01},
	} {
		require.True(t, meshOf(t, item.body, item.tol).BoundaryVerified(), name)
	}
}

// outerLoop returns the loop of the section with the largest reach from the shaft axis.
func outerLoop(t *testing.T, loops [][]r3.Vec) []r3.Vec {
	t.Helper()
	var best []r3.Vec
	far := 0.0
	for _, l := range loops {
		for _, v := range l {
			if r := math.Hypot(v.X, v.Y); r > far {
				far, best = r, l
			}
		}
	}
	require.NotNil(t, best)
	return best
}

// requireRadius requires every vertex of l to lie within tol of radius r from the shaft axis.
func requireRadius(t *testing.T, l []r3.Vec, r, tol float64, name string) {
	t.Helper()
	for _, v := range l {
		require.InDelta(t, r, math.Hypot(v.X, v.Y), tol, name)
	}
}

// TestTurbineFanClutchHub checks the revolved hub: the round bore clear of the shaft, the chamfers, the root round on the
// flange, the wall between the bore and the sockets, and the blade nuts' path above the flange.
func TestTurbineFanClutchHub(t *testing.T) {
	eachClutchCase(t, func(t *testing.T, c fanCase, m *turbinefan.Model, _ *decad.Document) {
		p, dm := c.params(), m.Dimensions
		rc := dm.ClutchBoreRadius
		require.GreaterOrEqual(t, rc-p.ShaftDiameter/2, 0.5)
		for _, z := range []float64{-1.5, 10, 19.0} {
			o := section(t, m.Hub, atZ(t, z))
			requireRadius(t, boreLoop(t, o.Loops, rc+0.6), rc, 0.02, fmt.Sprintf("bore at z %g", z))
		}
		grow := 0.0
		if 2*math.Pi*rc >= edgefinish.LongEdgeRun {
			grow = 0.25
		}
		for _, z := range []float64{-2.75, 19.75} {
			o := section(t, m.Hub, atZ(t, z))
			requireRadius(t, boreLoop(t, o.Loops, rc+0.6), rc+grow, 0.03, fmt.Sprintf("bore chamfer at z %g", z))
		}
		for z, want := range map[float64]float64{-2.75: turbinefan.FlangeRadius - 0.25, -0.25: turbinefan.FlangeRadius - 0.25, -1.5: turbinefan.FlangeRadius, 19.75: 13.75, 0.25: 15.5 - math.Sqrt(1.5*1.5-1.25*1.25), 1.6: 14} {
			requireRadius(t, outerLoop(t, section(t, m.Hub, atZ(t, z)).Loops), want, 0.03, fmt.Sprintf("outer at z %g", z))
		}
		o := section(t, m.Hub, atZ(t, 10))
		bore := boreLoop(t, o.Loops, rc+0.6)
		wall := math.Inf(1)
		for _, l := range o.Loops {
			if len(l) == len(bore) && l[0] == bore[0] {
				continue
			}
			for _, v := range bore {
				wall = math.Min(wall, distanceToLoop(v, l))
			}
		}
		require.GreaterOrEqual(t, wall, float64(turbinefan.MinWall)-o.OffSurface.Base())
		xy := func(v r3.Vec) (float64, float64) { return v.X, v.Y }
		for _, z := range []float64{1.6, 2.2} {
			level := section(t, m.Hub, atZ(t, z))
			for k := 1; k <= turbinefan.BladeCount; k++ {
				for _, r := range []float64{15.6, 20, 25} {
					d := bladeDir(k).Scale(r)
					require.False(t, inSection(level, xy, d.X, d.Y), "blade %d path at r %g, z %g", k, r, z)
				}
			}
		}
	})
}

// TestTurbineFanClutchAccess checks the blades against the clutch mount: the shaft-mount blades keep clear of the ear
// bolts' lines and the flange, and the clutch mount reuses them.
func TestTurbineFanClutchAccess(t *testing.T) {
	ears, err := func() (turbinefan.Dimensions, error) {
		p, err := turbinefan.ParamsForClutch(onewayclutch.DefaultParams())
		if err != nil {
			return turbinefan.Dimensions{}, err
		}
		p.Placement = nil
		return p.Derive()
	}()
	require.NoError(t, err)
	eachCase(t, pitchCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		for k, blade := range m.Blades {
			for _, v := range meshOf(t, blade, 0.1).Vertices() {
				for _, site := range ears.EarSites {
					require.False(t, math.Hypot(v.X-site.Head.X, v.Y-site.Head.Y) < m2fastener.HeadPocketRadius+0.5 && v.Z >= 0 && v.Z <= 25,
						"blade %d vertex %v blocks an ear bolt's path", k+1, v)
				}
				if r := math.Hypot(v.X, v.Y); r <= turbinefan.FlangeRadius {
					require.GreaterOrEqual(t, v.Z, 0.35, "blade %d at %v: gap to the flange front face", k+1, v)
					if r <= turbinefan.HubRadius+turbinefan.FlangeRound {
						require.GreaterOrEqual(t, v.Z, turbinefan.FlangeRound+0.35, "blade %d at %v: gap to the root round", k+1, v)
					}
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
		requireAccess(t, meshOf(t, fan.Blades[k], 0.1).Vertices(), k+1)
	}
}

// TestTurbineFanClutchCorners applies "Corners and roots" to the clutch hub and the clamp ring, both printed with the axis
// +Z.
func TestTurbineFanClutchCorners(t *testing.T) {
	eachClutchCase(t, func(t *testing.T, _ fanCase, m *turbinefan.Model, _ *decad.Document) {
		up := r3.Vec{Z: 1}
		// The mid-height z 11.2 cuts each blade socket's floor round at a slant, where it runs into the socket's wall,
		// 6.9 mm from the axis. With the 25.7 mm flange the section's chords there read a turn of 50.13 degrees, 0.12 over
		// the limit (the 28 mm flange's chords read it under). The round is 0.5 mm and tangent to the wall, so the part
		// has no corner there; only the chord sampling reads one.
		fs, err := cornercheck.Check(t.Context(), m.Hub, up, []cornercheck.Exempt{{
			Reason: "blade socket floor round meets its wall, chords read 50.13 degrees at z 11.2",
			Near: func(v r3.Vec) bool {
				r := math.Hypot(v.X, v.Y)
				return math.Abs(v.Z-11.2) < 1e-6 && r >= turbinefan.BladeSocketStart+1.1 && r <= turbinefan.BladeSocketStart+1.3
			},
		}})
		require.NoError(t, err)
		require.Empty(t, fs, cornercheck.Report(fs, 8))
		fs, err = cornercheck.CheckAt(t.Context(), m.Hub, up, []float64{-2.5, -1, 0.5, 3.4, 7, 10, 13},
			[]cornercheck.Exempt{socketFloorRound()})
		require.NoError(t, err)
		require.Empty(t, fs, cornercheck.Report(fs, 8))
		fs, err = cornercheck.Check(t.Context(), m.Clamp, up, nil)
		require.NoError(t, err)
		require.Empty(t, fs, cornercheck.Report(fs, 8))
		fs, err = cornercheck.CheckAt(t.Context(), m.Clamp, up, []float64{-28, -26.5, -25}, nil)
		require.NoError(t, err)
		require.Empty(t, fs, cornercheck.Report(fs, 8))
	})
}

// TestTurbineFanClutchEdgeFinishing checks the clamp ring's 0.5 chamfers: a section 0.25 inside each face is 0.25 inside
// the outer circle and 0.25 outside the inner one, and the ear holes keep their radius.
func TestTurbineFanClutchEdgeFinishing(t *testing.T) {
	m, _ := built(t, caseNamed(t, "clutch-605ZZ"))
	for _, z := range []float64{-28.25, -24.75} {
		o := section(t, m.Clamp, atZ(t, z))
		requireRadius(t, outerLoop(t, o.Loops), turbinefan.ClampOuterRadius-0.25, 0.03, fmt.Sprintf("outer at z %g", z))
		inner, holes := 0, 0
		for _, l := range o.Loops {
			far, near := 0.0, math.Inf(1)
			for _, v := range l {
				far, near = math.Max(far, math.Hypot(v.X, v.Y)), math.Min(near, math.Hypot(v.X, v.Y))
			}
			switch {
			case far > turbinefan.ClampOuterRadius-1:
			case far < 20:
				inner++
				requireRadius(t, l, turbinefan.ClampInnerRadius+0.25, 0.03, fmt.Sprintf("inner at z %g", z))
			default:
				holes++
				// The hole's centre is the nearest ear bolt's line.
				centre := m.EarSites[0].Head
				for _, site := range m.EarSites {
					if l[0].Sub(site.Head).Len() < l[0].Sub(centre).Len() {
						centre = site.Head
					}
				}
				for _, v := range l {
					require.InDelta(t, m2fastener.HoleRadius, math.Hypot(v.X-centre.X, v.Y-centre.Y), 0.02, "ear hole at z %g", z)
				}
			}
		}
		require.Equal(t, 1, inner, "z %g", z)
		require.Equal(t, onewayclutch.Ears, holes, "z %g", z)
	}
}
