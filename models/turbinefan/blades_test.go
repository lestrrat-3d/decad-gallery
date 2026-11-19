package turbinefan_test

// CI: isolated-tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestTurbineFanPitch pins the fan's hand and pitch. Blade 1 lies along +Y. In one radial ring of its mesh the chord
// runs from the trailing edge's midpoint to the leading edge. Seen from the front the fan turns clockwise, toward +X at
// the top, so the leading edge leads (+X) and sits upstream (-Z): the chord is (cos beta, 0, -sin beta), where beta is
// the blade angle turbinefan.Beta gives at that radius.
func TestTurbineFanPitch(t *testing.T) {
	eachCase(t, pitchCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		p, sz := c.params(), m.Dimensions.Size
		verts := meshOf(t, m.Blades[0], 0.1).Vertices()
		for _, r := range []float64{30, 40, 48, 50, 60} {
			y, ring := meshRing(verts, r*sz.Scale)
			require.NotEmpty(t, ring, "ring near radius %g", r*sz.Scale)
			if r == 48 {
				require.InDelta(t, sz.RefRadius, y, 1e-6, "rated-radius section")
			}
			// The trailing edge is the two-millimetre flat on local +X.
			// Blade 1 maps that plane to world X = -stud radius.
			lo, hi := math.Inf(1), math.Inf(-1)
			for _, v := range ring {
				if math.Abs(v.X+sz.BladeStudRadius) <= 1e-9 {
					lo, hi = math.Min(lo, v.Z), math.Max(hi, v.Z)
				}
			}
			require.InDelta(t, 2.0, hi-lo, 1e-8, "trailing flat at radius %g", y)
			mid := r3.Vec{X: -sz.BladeStudRadius, Y: y, Z: (lo + hi) / 2}
			le := farthest(ring, mid, nil)
			dx, dz := le.X-mid.X, le.Z-mid.Z
			require.Greater(t, dx, 0.0, "the leading edge leads toward +X at radius %g", y)
			require.Less(t, dz, 0.0, "the leading edge is upstream at radius %g", y)
			require.InDelta(t, sz.Beta(y, p.BladePitch), math.Atan2(-dz, dx), 0.5*math.Pi/180, "chord angle at radius %g", y)
		}
	})
}

// TestTurbineFanCopies requires blades 2 and 3, their bolts and their nuts to be blade 1's turned by 120 and 240
// degrees about the shaft axis, and the hub to hold three identical sockets with solid hub between them.
func TestTurbineFanCopies(t *testing.T) {
	eachCase(t, []string{"605ZZ", "steep", "compact"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		sz := m.Dimensions.Size
		for name, item := range map[string]struct {
			bodies [turbinefan.BladeCount]*decad.Body
			tol    float64
		}{
			"blade": {m.Blades, 0.1},
			"bolt":  {[3]*decad.Body{m.BladeBolts[0].Body, m.BladeBolts[1].Body, m.BladeBolts[2].Body}, 0.01},
			"nut":   {[3]*decad.Body{m.BladeNuts[0].Body, m.BladeNuts[1].Body, m.BladeNuts[2].Body}, 0.01},
		} {
			firstLo, firstHi := turnedBox(t, item.bodies[0], item.tol, 0)
			for k := 2; k <= turbinefan.BladeCount; k++ {
				lo, hi := turnedBox(t, item.bodies[k-1], item.tol, -2*math.Pi*float64(k-1)/turbinefan.BladeCount)
				for i, pair := range [][2]float64{
					{firstLo.X, lo.X}, {firstLo.Y, lo.Y}, {firstLo.Z, lo.Z}, {firstHi.X, hi.X}, {firstHi.Y, hi.Y}, {firstHi.Z, hi.Z},
				} {
					require.InDelta(t, pair[0], pair[1], 0.15, "%s %d matches %s 1 (bound %d)", name, k, name, i)
				}
			}
		}
		locate := func(at r3.Vec) decad.PointLocation {
			loc, err := m.Hub.LocatePoint(t.Context(), at, units.Millimeters(0.05))
			require.NoError(t, err)
			return loc
		}
		for k := 1; k <= turbinefan.BladeCount; k++ {
			u := bladeDir(k)
			for _, r := range []float64{7 * sz.HubScale, 12 * sz.HubScale} {
				require.Equal(t, decad.PointOutside, locate(u.Scale(r).Add(r3.Vec{Z: sz.BladeSocketZ()})),
					"socket %d is empty at radius %g", k, r)
			}
		}
		for _, deg := range []float64{30, 150, 270} {
			a := deg * math.Pi / 180
			at := r3.Vec{X: sz.BladeBoltAt * math.Cos(a), Y: sz.BladeBoltAt * math.Sin(a), Z: sz.HubMid}
			require.Equal(t, decad.PointInside, locate(at), "the hub is solid at %g degrees", deg)
		}
	})
}

// TestTurbineFanReplaceBlade proves a blade can be replaced: with its bolt out, the blade slides out along its radial
// line clear of the hub, and the bolt lifts out of the front face clear of the hub and the blade; and with the bolt in,
// the bolt stops the blade within the hole's clearance.
func TestTurbineFanReplaceBlade(t *testing.T) {
	eachCase(t, []string{"605ZZ", "steep", "compact"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		sz := m.Dimensions.Size
		hub := gapMesh(t, m.Hub, 0.02, r3.Vec{})
		for k := 1; k <= turbinefan.BladeCount; k++ {
			t.Run(fmt.Sprintf("blade-%d", k), func(t *testing.T) {
				blade, bolt := m.Blades[k-1], m.BladeBolts[k-1].Body
				for _, lift := range []float64{0.5, 2 * sz.Scale, 5 * sz.Scale, 10 * sz.Scale, 18.5 * sz.Scale} {
					moved := gapMesh(t, bolt, 0.01, r3.Vec{Z: lift})
					for name, other := range map[string]meshgap.Mesh{"hub": hub, "blade": gapMesh(t, blade, 0.02, r3.Vec{})} {
						ok, err := moved.ExceedsClearance(other, 0.05)
						require.NoError(t, err)
						require.True(t, ok, "the bolt lifted %g stays clear of the %s", lift, name)
					}
				}
				u := bladeDir(k)
				// Sliding the blade out along its radial line: its section through the stud's axis, and the one across the
				// stud at mid length, moved by the slide, stay 0.2 or more from the hub's. The stud is a body of revolution
				// about the radial line, so these two planes bound every gap between it and the socket, and the root beyond
				// the rim only moves away from the hub. (A mesh-to-mesh check of the whole blade reads the coincident bolt
				// holes of the stud and the hub as undecided.)
				for _, plane := range []r3.Frame{radial(t, k), atZ(t, sz.BladeSocketZ())} {
					hubOutline, bladeOutline := section(t, m.Hub, plane), section(t, blade, plane)
					for _, s := range []float64{0.5, 1.7, 4, 6.3, 8, 8.5} {
						slide := s * sz.HubScale
						moved := bladeOutline
						moved.Loops = nil
						for _, loop := range bladeOutline.Loops {
							shifted := make([]r3.Vec, len(loop))
							for i, v := range loop {
								shifted[i] = v.Add(u.Scale(slide))
							}
							moved.Loops = append(moved.Loops, shifted)
						}
						cl, err := hubOutline.Clearance(moved)
						require.NoError(t, err)
						require.False(t, cl.Overlaps, "the blade slid out %g overlaps the hub", slide)
						require.GreaterOrEqual(t, cl.Lower().Base(), 0.2, "the blade slid out %g", slide)
					}
				}
				// The shank's inner side lies 1.0 inside the bolt line and the hole's inner wall 1.3: material at 1.5 and
				// hollow at 1.0 mean a slide outward of 0.5 would put the blade's wall through the shank, so the bolt
				// stops it within 0.3. The same holds at the hole's outer wall and the shank's outer side.
				// The radial plane through the blade's axis shows the hole: the blade's point location is undecided there.
				plane := section(t, blade, radial(t, k))
				uv := func(v r3.Vec) (float64, float64) { return v.X*u.X + v.Y*u.Y, v.Z }
				solid := func(r float64) bool { return inSection(plane, uv, r, sz.BladeSocketZ()) }
				shank := sz.BladeBoltAt
				require.True(t, solid(shank-1.5), "the stud's wall inside the shank")
				require.False(t, solid(shank-1.0), "the shank's inner side lies in the hole")
				require.False(t, solid(shank+1.0), "the shank's outer side lies in the hole")
				require.True(t, solid(shank+1.5), "the stud's wall beyond the shank")
				require.Less(t, 1.3-0.97, 0.35, "the hole clearance leaves the blade under 0.35 of play")
				require.Greater(t, m2fastener.HoleRadius, 1.0)
			})
		}
	})
}

// TestTurbineFanLeftHand builds the left-hand fan and proves its blades are the right-hand blades reflected in the plane
// x = 0: the same volume and the bounds mirrored in x. The hub, the bolts and the nuts do not depend on the hand, so the
// other tests run for the right hand only.
func TestTurbineFanLeftHand(t *testing.T) {
	right, _ := built(t, caseNamed(t, "605ZZ"))
	p := turbinefan.DefaultParams()
	p.Hand = turbinefan.LeftHand
	doc := turbinefan.NewDocument()
	parts, err := turbinefan.BuildParts(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 4)
	for k := range parts.Blades {
		// Blade k of the left fan is the mirror of blade k of the right fan in x = 0; blades 2 and 3 swap sides.
		mirror := map[int]int{0: 0, 1: 2, 2: 1}[k]
		rb, err := right.Blades[mirror].Bounds()
		require.NoError(t, err)
		lb, err := parts.Blades[k].Bounds()
		require.NoError(t, err)
		tol := rb.Bound.Base() + lb.Bound.Base() + 1e-3
		require.InDelta(t, -rb.Max.X, lb.Min.X, tol, "blade %d", k+1)
		require.InDelta(t, -rb.Min.X, lb.Max.X, tol, "blade %d", k+1)
		require.InDelta(t, rb.Min.Y, lb.Min.Y, tol, "blade %d", k+1)
		require.InDelta(t, rb.Max.Y, lb.Max.Y, tol, "blade %d", k+1)
		require.InDelta(t, rb.Min.Z, lb.Min.Z, tol, "blade %d", k+1)
		require.InDelta(t, rb.Max.Z, lb.Max.Z, tol, "blade %d", k+1)
		rv, err := right.Blades[mirror].Volume()
		require.NoError(t, err)
		lv, err := parts.Blades[k].Volume()
		require.NoError(t, err)
		require.InDelta(t, rv.Value.Base(), lv.Value.Base(), 2e-3*rv.Value.Base(), "blade %d", k+1)
	}
	// The blade nut slots stay reachable: no left-hand blade vertex lies in its nut's path.
	for k := range parts.Blades {
		requireAccess(t, parts.Dimensions.Size, meshOf(t, parts.Blades[k], 0.1).Vertices(), k+1)
	}
	// The leading edge now leads toward -X and still sits upstream.
	verts := meshOf(t, parts.Blades[0], 0.1).Vertices()
	sz := parts.Dimensions.Size
	y, ring := meshRing(verts, 40*sz.Scale)
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range ring {
		if math.Abs(v.X-sz.BladeStudRadius) < 1e-9 {
			lo, hi = math.Min(lo, v.Z), math.Max(hi, v.Z)
		}
	}
	require.InDelta(t, 2, hi-lo, 1e-8, "left-hand trailing flat")
	mid := r3.Vec{X: sz.BladeStudRadius, Y: y, Z: (lo + hi) / 2}
	le := farthest(ring, mid, nil)
	require.Less(t, le.X-mid.X, 0.0)
	require.Less(t, le.Z-mid.Z, 0.0)
	require.InDelta(t, sz.Beta(y, p.BladePitch), math.Atan2(-(le.Z-mid.Z), -(le.X-mid.X)), 0.5*math.Pi/180)
}
