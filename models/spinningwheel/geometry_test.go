package spinningwheel_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

type surfaceTriangles struct {
	vertices  []r3.Vec
	triangles [][3]int
	bound     float64
}

// The probes avoid surfaces by at least0.2 mm. All meshes have already passed VerifyAll.
func (m surfaceTriangles) contains(p r3.Vec) bool {
	d := r3.Vec{X: 1, Y: .17320508075688773, Z: .2718281828459045}
	hits := make([]float64, 0, 8)
	for _, f := range m.triangles {
		a, b, c := m.vertices[f[0]], m.vertices[f[1]], m.vertices[f[2]]
		e1, e2 := b.Sub(a), c.Sub(a)
		h := d.Cross(e2)
		det := e1.Dot(h)
		if math.Abs(det) < 1e-11 {
			continue
		}
		inv := 1 / det
		s := p.Sub(a)
		u := inv * s.Dot(h)
		if u < -1e-10 || u > 1+1e-10 {
			continue
		}
		q := s.Cross(e1)
		v := inv * d.Dot(q)
		if v < -1e-10 || u+v > 1+1e-10 {
			continue
		}
		distance := inv * e2.Dot(q)
		if distance <= 1e-8 {
			continue
		}
		duplicate := false
		for _, previous := range hits {
			if math.Abs(previous-distance) < 1e-7 {
				duplicate = true
				break
			}
		}
		if !duplicate {
			hits = append(hits, distance)
		}
	}
	return len(hits)%2 == 1
}

func wheelPart(t *testing.T, a *spinningwheel.Assembly, name string) *decad.Body {
	t.Helper()
	for _, p := range a.Parts {
		if p.Name == name {
			return p.Body
		}
	}
	t.Fatalf("missing part %s", name)
	return nil
}

func requireOnePieceWhorl(t *testing.T, a *spinningwheel.Assembly, p spinningwheel.Params) {
	t.Helper()
	body := wheelPart(t, a, "flyer-drive-whorl")
	cones := 0
	for _, face := range body.Faces() {
		if face.Surface().Kind() == decad.KindCone {
			cones++
		}
	}
	require.Equal(t, 8, cones, "all four V grooves must retain both native conical walls")
	// Integrate the eight conical walls and cylindrical lands independently of the producer's profile.
	outerVolume := 0.0
	for i, ratio := range []float64{7, 9, 12, 17} {
		root := a.State.WheelPitch/ratio - math.Sqrt(5)
		lip := root + 6*p.Scale
		outerVolume += 2 * math.Pi * p.Scale * (root*root + root*lip + lip*lip)
		land := []float64{3, 2, 2, 1}[i] * p.Scale
		outerVolume += math.Pi * lip * lip * land
	}
	const radius, halfWidth, extent = 8.1, 1.6, 8.8
	boreArea := math.Pi*radius*radius + 2*halfWidth*extent -
		halfWidth*math.Sqrt(radius*radius-halfWidth*halfWidth) - radius*radius*math.Asin(halfWidth/radius)
	want := outerVolume - boreArea*32*p.Scale
	volume, err := body.Volume()
	require.NoError(t, err)
	require.Less(t, volume.Bound.Base(), 1e-5*want, "whorl volume must have a tight bound")
	require.InDelta(t, want, volume.Value.Base(), volume.Bound.Base()+1e-5, "one-piece rim minus its constant keyed through-bore")
}

func requireWheelFeatures(t *testing.T, a *spinningwheel.Assembly, p spinningwheel.Params, meshes map[string]surfaceTriangles) {
	t.Helper()
	// The rear wedge is the 90 mm wide band between the rail's top line and z=525, 30 mm thick, all scaled.
	wedge, err := wheelPart(t, a, "rear-head-wedge").Volume()
	require.NoError(t, err)
	wantWedge := 90 * (525 - (500 - (-150+280)*90/520.0)) * 30 * p.Scale * p.Scale * p.Scale
	require.InDelta(t, wantWedge, wedge.Value.Base(), wedge.Bound.Base()+1e-6*wantWedge, "wedge volume")
	s, st := p.Scale, a.State
	headPoint := func(x, y, z float64) r3.Vec {
		angle := math.Remainder(st.FlyerAngle, 2*math.Pi) + st.HeadTilt
		return r3.Vec{X: st.Flyer.X + x*math.Cos(angle) - z*math.Sin(angle), Y: y + st.HeadShift, Z: st.Flyer.Z + x*math.Sin(angle) + z*math.Cos(angle)}
	}
	shaft, exists := meshes["flyer-shaft"]
	require.True(t, exists)
	for _, y := range []float64{-325, -300, -230} {
		require.False(t, shaft.contains(headPoint(0, y*s, 0)), "orifice must remain open atY%g", y)
	}
	require.True(t, shaft.contains(headPoint(6.5, -300*s, 0)), "tube wall must exist outside the10 mm bore")
	require.True(t, shaft.contains(headPoint(0, -206.5*s, 0)), "blind bore must have an end wall")
	for _, z := range []float64{4, 6.5, 8.5} {
		require.False(t, shaft.contains(headPoint(0, -227*s, z)), "radial outlet must open atZ%g", z)
	}
	require.True(t, shaft.contains(headPoint(3.5, -227*s, 6.5)), "port must preserve nearby tube wall")
	for _, y := range []float64{-215, -270} {
		require.False(t, shaft.contains(headPoint(7.5, y*s, 0)), "key seat must open atY%g", y)
		require.True(t, shaft.contains(headPoint(6.5, y*s, 0)), "key seat must preserve tube wall atY%g", y)
	}
	require.True(t, shaft.contains(headPoint(3.5, -48*s+3.5, 0)), "rear bearing journal must remain round")
	require.True(t, shaft.contains(headPoint(7.5, -300*s, 0)), "front sleeve journal must remain round")
	whorl, exists := meshes["flyer-drive-whorl"]
	require.True(t, exists)
	require.False(t, whorl.contains(headPoint(8.5, -266*s, 0)), "whorl keyway must receive the removable key")
	require.True(t, meshes["flyer-whorl-key"].contains(headPoint(8.5, -266*s, 0)), "key must project beyond the round journal")
	require.True(t, whorl.contains(headPoint(8.5, -266*s, 2)), "hub material must flank the keyway")
	require.True(t, whorl.contains(headPoint(0, -266*s, 8.5)), "a quarter-turned key must interfere with the hub")
	bobbin, exists := meshes["bobbin"]
	require.True(t, exists)
	for _, x := range []float64{0, 4, 5.75} {
		require.False(t, bobbin.contains(headPoint(x, -160*s, 0)), "bobbin passage must clear shaft and spacer")
	}
	require.True(t, bobbin.contains(headPoint(8, -160*s, 0)), "bobbin barrel wall must exist")
	require.False(t, meshes["bobbin-inner-race-sleeve"].contains(headPoint(5.75, -160*s, 0)), "11 OD spacer must clear12 bore")
	heel := meshes["heel-pin"]
	for _, x := range []float64{-260, 220} {
		require.True(t, heel.contains(st.PedalFrame.ToWorld(r3.Vec{X: x * s, Y: 3.7})), "heel journal must stay round")
	}
	require.False(t, heel.contains(st.PedalFrame.ToWorld(r3.Vec{X: 60 * s, Y: 3.5})), "heel shaft must receive its key")
	require.True(t, heel.contains(st.PedalFrame.ToWorld(r3.Vec{X: 60 * s, Y: 2.5})), "heel shaft preserves material behind its key seat")
	rail := meshes["treadle-heel-rail"]
	require.True(t, rail.contains(st.PedalFrame.ToWorld(r3.Vec{X: 60 * s, Y: 4.6, Z: -1.4})), "half-lapped heel rail must flank its keyway")
	keyPoint := st.PedalFrame.ToWorld(r3.Vec{X: 60 * s, Y: 4.5, Z: -.5})
	require.False(t, rail.contains(keyPoint), "heel rail must clear the installed key")
	require.True(t, meshes["heel-key"].contains(keyPoint), "key must occupy its passage")
	require.True(t, rail.contains(st.PedalFrame.ToWorld(r3.Vec{X: -100 * s, Z: 4.5})), "a quarter-turned key must interfere with the heel rail")
	requireBoreCaps(t, wheelPart(t, a, "footman-head-boss"), st.RodHead, st.RodFrame.V(), 7)
	stock608, stock605 := 0, 0
	for _, part := range a.Parts {
		if part.Group != "bearing" {
			continue
		}
		inner, outerRadius := math.Inf(1), 0.0
		var axis r3.Vec
		for _, face := range part.Body.Faces() {
			if cylinder, ok := face.Surface().(decad.Cylinder); ok {
				inner, outerRadius = math.Min(inner, cylinder.Radius.Base()), math.Max(outerRadius, cylinder.Radius.Base())
				axis = cylinder.Axis
			}
		}
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, edge := range part.Body.Edges() {
			if circle, ok := edge.Curve().(decad.Circle3); ok {
				coordinate := circle.Center.Dot(axis)
				lo, hi = math.Min(lo, coordinate), math.Max(hi, coordinate)
			}
		}
		width := hi - lo
		bore, outer := 8.0, 22.0
		if math.Abs(width-5) < 1e-6 {
			stock605++
			bore, outer = 5, 14
		} else {
			stock608++
			require.InDelta(t, 7, width, 1e-6, "%s stock width", part.Name)
		}
		require.InDelta(t, outer, 2*outerRadius, 1e-6, "%s stock outside diameter", part.Name)
		require.InDelta(t, bore, 2*inner, 1e-6, "%s stock bore", part.Name)
		volume, err := part.Body.Volume()
		require.NoError(t, err)
		require.InDelta(t, math.Pi*(outer*outer-bore*bore)*width/4, volume.Value.Base(), volume.Bound.Base()+1e-6, "%s stock annulus", part.Name)
	}
	require.Equal(t, 7, stock608)
	require.Equal(t, 3, stock605)
}

func requireHeadAdjustment(t *testing.T, a *spinningwheel.Assembly, p spinningwheel.Params) {
	t.Helper()
	tip, err := wheelPart(t, a, "mother-jack-screw").Bounds()
	require.NoError(t, err)
	require.InDelta(t, 540*p.Scale, tip.Min.Z, tip.Bound.Base()+1e-6, "pointed jack must touch rather than penetrate its fixed shelf")
	base := wheelPart(t, a, "mother-base")
	for _, x := range []float64{-180, -120} {
		location, err := base.LocatePoint(t.Context(), r3.Vec{X: x * p.Scale, Y: -70 * p.Scale, Z: 532 * p.Scale}, units.Millimeters(.025))
		require.NoError(t, err)
		require.True(t, location == decad.PointOutside, "alignment bolt must pass through the real slot at all ratios, got %s", location)
	}
	cords := 0
	for _, part := range a.Parts {
		if part.Group != "brake-band" {
			continue
		}
		cords++
		require.Equal(t, "brake-band", part.Name, "main cord and its spliced eyes must be one physical part")
	}
	require.Equal(t, 1, cords)
	radius := 42*p.Scale + .6*math.Sqrt(5) + .05
	// Both tails shorten by the springs' shift, max(0, 15-15*Scale)+0.01.
	wantLength := math.Pi*radius + 612*p.Scale - 36.8 + 14.6*math.Pi - 2*(math.Max(0, 15-15*p.Scale)+.01)
	require.InDelta(t, wantLength, a.State.BrakeLength, 1e-9, "nominal centreline length includes both terminal eyes")
}

// requireBoreCaps requires an exact bore cylinder of the radius on the axis through centre, and a rim of it on
// two planar caps. A rim is an exact Circle3 on that axis, or a FacetedCurve within 0.005 mm of the bore
// cylinder: a bore cut by a tool on another lift than its host (the footman boss's, on the cross bearings'
// gimbal lift) leaves the Cut's faceted path, where Decad correctly refuses to call the rim a circle (rule E12);
// the cylinder face itself stays exact.
func requireBoreCaps(t *testing.T, body *decad.Body, centre, axis r3.Vec, radius float64) {
	t.Helper()
	onAxis := func(p r3.Vec) bool {
		delta := p.Sub(centre)
		return delta.Sub(axis.Scale(delta.Dot(axis))).Len() < 1e-6
	}
	isBore := func(face *decad.Face) bool {
		c, ok := face.Surface().(decad.Cylinder)
		return ok && onAxis(c.Origin) && c.Axis.Cross(axis).Len() < 1e-6 && math.Abs(c.Radius.Base()-radius) < 1e-6
	}
	bores, found := 0, 0
	for _, face := range body.Faces() {
		if isBore(face) {
			bores++
		}
		if face.Surface().Kind() != decad.KindPlane {
			continue
		}
		rim := false
		for _, loop := range face.Loops() {
			for _, edge := range loop.Edges() {
				switch c := edge.Curve().(type) {
				case decad.Circle3:
					rim = rim || onAxis(c.Center) && math.Abs(c.Radius.Base()-radius) < 1e-6
				case decad.FacetedCurve:
					if c.Bound.Base() <= 0.005 {
						for _, other := range edge.Faces() {
							rim = rim || isBore(other)
						}
					}
				}
			}
		}
		if rim {
			found++
		}
	}
	require.Positive(t, bores, "real pivot bore must be an exact cylinder")
	require.Equal(t, 2, found, "real pivot bore must reach both caps")
}
