package spinningwheel_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

type surfaceTriangles struct {
	vertices  []r3.Vec
	triangles [][3]int
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
	const radius, flat = 4.1, 3.1
	capArea := radius*radius*math.Acos(flat/radius) - flat*math.Sqrt(radius*radius-flat*flat)
	dArea := math.Pi*radius*radius - capArea
	want := outerVolume - dArea*32*p.Scale
	volume, err := body.Volume()
	require.NoError(t, err)
	require.Less(t, volume.Bound.Base(), 1e-5*want, "whorl volume must have a tight bound")
	require.InDelta(t, want, volume.Value.Base(), volume.Bound.Base()+1e-5, "one-piece rim minus its constant D through-bore")
}

func requireWheelFeatures(t *testing.T, a *spinningwheel.Assembly, p spinningwheel.Params, meshes map[string]surfaceTriangles) {
	t.Helper()
	s, st := p.Scale, a.State
	headPoint := func(x, y, z float64) r3.Vec {
		angle := math.Remainder(st.FlyerAngle, 2*math.Pi)
		return r3.Vec{X: st.Flyer.X + x*math.Cos(angle) - z*math.Sin(angle), Y: y + st.HeadShift, Z: st.Flyer.Z + x*math.Sin(angle) + z*math.Cos(angle)}
	}
	shaft, exists := meshes["flyer-shaft"]
	require.True(t, exists)
	for _, y := range []float64{-270, -250, -230} {
		require.False(t, shaft.contains(headPoint(0, y*s, 0)), "orifice must remain open atY%g", y)
	}
	require.True(t, shaft.contains(headPoint(6.5, -250*s, 0)), "tube wall must exist outside the10 mm bore")
	require.True(t, shaft.contains(headPoint(0, -206.5*s, 0)), "blind bore must have an end wall")
	for _, z := range []float64{4, 6.5, 8.5} {
		require.False(t, shaft.contains(headPoint(0, -227*s, z)), "radial outlet must open atZ%g", z)
	}
	require.True(t, shaft.contains(headPoint(3.5, -227*s, 6.5)), "port must preserve nearby tube wall")
	require.False(t, shaft.contains(headPoint(6.5, -215*s, 0)), "front positive coupling needs its D flat")
	require.True(t, shaft.contains(headPoint(5.5, -215*s, 0)), "front D shaft needs material behind its flat")
	require.False(t, shaft.contains(headPoint(3.5, -15*s, 0)), "rear positive coupling needs its D flat")
	require.True(t, shaft.contains(headPoint(2.5, -15*s, 0)), "rear D shaft must remain solid")
	require.True(t, shaft.contains(headPoint(3.5, -48*s+3.5, 0)), "rear bearing journal must remain round")
	require.True(t, shaft.contains(headPoint(7.5, -245*s, 0)), "front sleeve journal must remain round")
	whorl, exists := meshes["flyer-drive-whorl"]
	require.True(t, exists)
	require.True(t, whorl.contains(headPoint(3.5, -st.HeadShift, 0)), "D bore's flat must leave torque-transmitting material")
	require.False(t, whorl.contains(headPoint(2.5, -st.HeadShift, 0)), "D bore must remain open")
	// A quarter-turn error would put shaft material into the whorl's flat.
	require.True(t, shaft.contains(headPoint(0, -15*s, -3.5)), "the noncircular interface must prevent free relative rotation")
	bobbin, exists := meshes["bobbin"]
	require.True(t, exists)
	for _, x := range []float64{0, 4, 5.75} {
		require.False(t, bobbin.contains(headPoint(x, -160*s, 0)), "bobbin passage must clear shaft and spacer")
	}
	require.True(t, bobbin.contains(headPoint(8, -160*s, 0)), "bobbin barrel wall must exist")
	require.False(t, meshes["bobbin-inner-race-sleeve"].contains(headPoint(5.75, -160*s, 0)), "11 OD spacer must clear12 bore")
	requireBoreCaps(t, wheelPart(t, a, "treadle"), st.Heel, 11)
	requireBoreCaps(t, wheelPart(t, a, "footman"), st.CrankPin, 7)
	requireBoreCaps(t, wheelPart(t, a, "footman"), st.Toe, 7)
	stock608, stock605 := 0, 0
	for _, part := range a.Parts {
		if part.Group != "bearing" {
			continue
		}
		box, err := part.Body.Bounds()
		require.NoError(t, err)
		width, diameter := box.Max.Y-box.Min.Y, box.Max.X-box.Min.X
		bore, outer := 8.0, 22.0
		if math.Abs(width-5) < 1e-6 {
			stock605++
			bore, outer = 5, 14
		} else {
			stock608++
			require.InDelta(t, 7, width, 1e-6, "%s stock width", part.Name)
		}
		require.InDelta(t, outer, diameter, 1e-6, "%s stock outside diameter", part.Name)
		volume, err := part.Body.Volume()
		require.NoError(t, err)
		require.InDelta(t, math.Pi*(outer*outer-bore*bore)*width/4, volume.Value.Base(), volume.Bound.Base()+1e-6, "%s stock annulus", part.Name)
	}
	require.Equal(t, 7, stock608)
	require.Equal(t, 2, stock605)
}

func requireBoreCaps(t *testing.T, body *decad.Body, centre r3.Vec, radius float64) {
	t.Helper()
	found := 0
	for _, face := range body.Faces() {
		if face.Surface().Kind() != decad.KindPlane {
			continue
		}
		for _, loop := range face.Loops() {
			for _, edge := range loop.Edges() {
				c, ok := edge.Curve().(decad.Circle3)
				if ok && math.Hypot(c.Center.X-centre.X, c.Center.Z-centre.Z) < 1e-6 && math.Abs(c.Radius.Base()-radius) < 1e-6 {
					found++
				}
			}
		}
	}
	require.Equal(t, 2, found, "real pivot bore must reach both caps")
}
