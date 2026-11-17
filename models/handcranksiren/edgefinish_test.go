package handcranksiren_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// largeFaces returns the planar faces of body that face along normal (a signed unit axis), lie at one of the offsets
// along it and have at least LargeFaceArea of net area. A chamfer is cut into the body, so a face is measured after it.
func largeFaces(t *testing.T, body *decad.Body, normal r3.Vec, offsets ...float64) []*decad.Face {
	t.Helper()
	var out []*decad.Face
	for _, f := range body.Faces() {
		plane, ok := f.Surface().(decad.Plane)
		if !ok || math.Abs(math.Abs(plane.Frame.N().Dot(normal))-1) > 1e-9 {
			continue
		}
		match := false
		for _, o := range offsets {
			match = match || math.Abs(plane.Frame.Origin().Dot(normal)-o) < 1e-6
		}
		if !match {
			continue
		}
		area, err := f.Area()
		require.NoError(t, err)
		if area.Value.Base() >= edgefinish.LargeFaceArea {
			out = append(out, f)
		}
	}
	return out
}

// bevelled reports whether the face beside an edge of f, whose normal lies along normal, is a chamfer: a plane or cone
// that is neither parallel nor perpendicular to f. Decad reports the chamfer round the frame's rounded corners as a
// faceted surface with no geometry to measure. The wall it replaces is a cylinder, which stays a cylinder, so a faceted neighbour is the
// chamfer.
func bevelled(t *testing.T, f, other *decad.Face, normal r3.Vec) bool {
	t.Helper()
	switch s := other.Surface().(type) {
	case decad.Plane:
		d := math.Abs(s.Frame.N().Dot(normal))
		return d > 0.1 && d < 0.9
	case decad.Cone:
		half, err := s.HalfAngle.In(units.Radians(1).Unit())
		require.NoError(t, err)
		// A cone that opens the other way is reported with the supplementary half angle.
		return math.Abs(half-math.Pi/4) < 1e-6 || math.Abs(half-3*math.Pi/4) < 1e-6
	case decad.Faceted:
		return true
	}
	return false
}

// rimExtent is the extent of the outer loops of a set of faces. A circle has one vertex, so radius is the largest
// vertex distance from the line through the centre along the normal, which is the radius of a disc or flange centred
// on it.
type rimExtent struct {
	min, max r3.Vec
	radius   float64
}

// requireChamfered checks the large faces facing +Y: the centre of its rim radius is the rotor axis.
func requireChamfered(t *testing.T, name string, body *decad.Body, keep edgefinish.LoopFilter, offsets ...float64) rimExtent {
	t.Helper()
	return requireChamferedAlong(t, name, body, r3.Vec{Y: 1}, r3.Vec{Z: 84}, keep, offsets...)
}

// requireChamferedAlong checks that every edge of every kept loop on the large faces facing normal at the offsets has a
// bevel beside it, and returns the outer-loop extent of those faces. The rim radius is measured from the line through
// centre along normal.
func requireChamferedAlong(t *testing.T, name string, body *decad.Body, normal, centre r3.Vec, keep edgefinish.LoopFilter,
	offsets ...float64) rimExtent {
	t.Helper()
	return requireChamferedExcept(t, name, body, normal, centre, keep, nil, offsets...)
}

// requireChamferedExcept is requireChamferedAlong that leaves out the edges of a kept loop that sharp reports as listed
// sharp edges. A nil sharp leaves out none.
func requireChamferedExcept(t *testing.T, name string, body *decad.Body, normal, centre r3.Vec, keep edgefinish.LoopFilter,
	sharp func(*decad.Edge) bool, offsets ...float64) rimExtent {
	t.Helper()
	faces := largeFaces(t, body, normal, offsets...)
	require.NotEmpty(t, faces, "%s: large faces", name)
	inf := math.Inf(1)
	rim := rimExtent{min: r3.Vec{X: inf, Y: inf, Z: inf}, max: r3.Vec{X: -inf, Y: -inf, Z: -inf}}
	for _, f := range faces {
		for _, loop := range f.Loops() {
			edges := loop.Edges()
			if loop.IsOuter() {
				for _, e := range edges {
					for _, v := range []r3.Vec{e.Start().Position().Value, e.End().Position().Value} {
						rim.min = r3.Vec{X: math.Min(rim.min.X, v.X), Y: math.Min(rim.min.Y, v.Y), Z: math.Min(rim.min.Z, v.Z)}
						rim.max = r3.Vec{X: math.Max(rim.max.X, v.X), Y: math.Max(rim.max.Y, v.Y), Z: math.Max(rim.max.Z, v.Z)}
						d := v.Sub(centre)
						rim.radius = math.Max(rim.radius, d.Sub(normal.Scale(d.Dot(normal))).Len())
					}
				}
			}
			if !keep(loop, edges) {
				continue
			}
			for _, e := range edges {
				if sharp != nil && sharp(e) {
					continue
				}
				beside := 0
				for _, other := range e.Faces() {
					if other != f && bevelled(t, f, other, normal) {
						beside++
					}
				}
				require.Equal(t, 1, beside, "%s: an edge of a kept loop has no bevel", name)
			}
		}
	}
	return rim
}

// requireEdgeFinishing checks the Surface edge finishing table of instructions.md.
func requireEdgeFinishing(t *testing.T, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	longAll, longInner := edgefinish.Long(edgefinish.All), edgefinish.Long(edgefinish.Inner)
	// The plates and the frame: the face outlines sit 0.5 inside the bounds in X and Z.
	for name, offsets := range map[string][]float64{
		"gearbox-front": {0, 10}, "gearbox-rear": {27.6, 37.6}, "gearbox-frame": {10.25, 27.35},
	} {
		rim := requireChamfered(t, name, parts[name].Body, longAll, offsets...)
		box := bounds(t, parts[name])
		require.InDelta(t, box.Min.X+0.5, rim.min.X, 1e-3+box.Bound.Base(), name)
		require.InDelta(t, box.Max.X-0.5, rim.max.X, 1e-3+box.Bound.Base(), name)
		require.InDelta(t, box.Min.Z+0.5, rim.min.Z, 1e-3+box.Bound.Base(), name)
		require.InDelta(t, box.Max.Z-0.5, rim.max.Z, 1e-3+box.Bound.Base(), name)
	}
	// The drum: the windows and the shaft hole of the back plate, and the chamfered corners of the flange.
	// The posts' junction with the back plate is concave and stays sharp, so only the windows are kept; the shaft
	// hole's rim is chamfered in the meridian and its single circle is checked with them.
	windows := func(loop *decad.Loop, edges []*decad.Edge) bool {
		return longInner(loop, edges) && (len(edges) > 1 || edges[0].Curve() != nil && isShaftHole(edges[0]))
	}
	requireChamfered(t, "drum", parts["drum"].Body, windows, -11.50, -15.50)
	// The drum's outer rim on the back plate and its flange faces take their chamfers in the meridian. The flange's
	// front face at -28.40 has no concave junction, so every long loop of it is kept (the bolt holes are short). Its rear face at -21.40 meets the wall
	// at a concave junction, so only the outer loop is kept.
	requireChamfered(t, "drum rim", parts["drum"].Body, edgefinish.Outer, -11.50, -21.40)
	requireChamfered(t, "drum flange", parts["drum"].Body, longAll, -28.40)
	// The discs: the outer loops of both caps, 0.5 inside the radius.
	for name, row := range map[string]struct {
		offsets []float64
		radius  float64
	}{"stator": {[]float64{-31.65, -28.65}, 58}, "rotor": {[]float64{-27.5, -24.5}, 45}} {
		rim := requireChamfered(t, name, parts[name].Body, edgefinish.Outer, row.offsets...)
		require.InDelta(t, row.radius-0.5, rim.radius, 1e-3+bounds(t, parts[name]).Bound.Base(), name)
	}
	// The collector: the outer loops of the rear and end faces sit 0.5 inside the bounds in X and Z, the bore rim on the
	// ceiling and the throat rim on both wall faces are chamfered.
	collectorBox := bounds(t, parts["collector"])
	rim := requireChamferedAlong(t, "collector", parts["collector"].Body, r3.Vec{Y: 1}, r3.Vec{Z: 84}, longAll, -31.90, -96.90)
	tol := 1e-3 + collectorBox.Bound.Base()
	require.InDelta(t, collectorBox.Min.X+0.5, rim.min.X, tol, "collector rim")
	require.InDelta(t, collectorBox.Max.X-0.5, rim.max.X, tol, "collector rim")
	require.InDelta(t, collectorBox.Min.Z+0.5, rim.min.Z, tol, "collector rim")
	require.InDelta(t, collectorBox.Max.Z-0.5, rim.max.Z, tol, "collector rim")
	requireChamferedAlong(t, "collector ceiling", parts["collector"].Body, r3.Vec{Y: 1}, r3.Vec{Z: 84}, longInner, -39.90)
	throat := r3.Vec{Y: -65.90, Z: 84}
	requireChamferedAlong(t, "collector throat", parts["collector"].Body, r3.Vec{X: 1}, throat, longInner, -51, -46)
	// The horn's mouth lip (vertex G) is the widest point: mouth + 2.5 - 0.5 sin 15 degrees = 51.1655 from the axis.
	hornBox := bounds(t, parts["horn"])
	require.InDelta(t, 51.1655, (hornBox.Max.Y-hornBox.Min.Y)/2, 1e-3+hornBox.Bound.Base())
	hornRim := requireChamferedAlong(t, "horn flange", parts["horn"].Body, r3.Vec{X: 1}, throat, edgefinish.Outer, -51.25, -59.25)
	require.InDelta(t, 29.5, hornRim.radius, 1e-3+hornBox.Bound.Base(), "horn flange rim")
	// The mating face's bore edge and the mouth lip face have no concave junction.
	requireChamferedAlong(t, "horn mating face and lip", parts["horn"].Body, r3.Vec{X: 1}, throat, longAll, -51.25, -159.25)

	// The grip's caps take the 2.0 chamfer of deviation A7: the rim sits 2.0 inside the bounds in Y and Z.
	gripBox := bounds(t, parts["grip"])
	gripRim := requireChamferedAlong(t, "grip", parts["grip"].Body, r3.Vec{X: 1}, r3.Vec{}, longAll, 79, 105)
	tol = 1e-3 + gripBox.Bound.Base()
	require.InDelta(t, -74.25, gripRim.min.Y, tol, "grip rim")
	require.InDelta(t, -2.25, gripRim.max.Y, tol, "grip rim")
	require.InDelta(t, 7, gripRim.min.Z, tol, "grip rim")
	require.InDelta(t, 123, gripRim.max.Z, tol, "grip rim")

	// The crank arm's caps, posed by the crank. The slit opens both faces' outlines: its walls and its end round stay
	// sharp (models/crankhandle instructions.md, "Retained sharp corners and edges"). The bore's rim in the back face is
	// chamfered too, but the slit joins it to the outline, so the check skips it with the slit's edges. Every other
	// edge of the outer loops is chamfered.
	crankAxis, err := handcranksiren.PosePoint(p, "crank", r3.Vec{X: 60, Z: 84})
	require.NoError(t, err)
	requireChamferedExcept(t, "crank-arm", parts["crank-arm"].Body, r3.Vec{Y: 1}, crankAxis, edgefinish.Outer,
		armSharpEdge(t, p), 38.6, 46.6, 55.6)

	// 48T and 50T gears: a 0.2 cap chamfer pulls the tips in 0.1 inside each cap.
	state, err := handcranksiren.Motion(p)
	require.NoError(t, err)
	for _, g := range [...]struct {
		name  string
		front float64
		axis  r3.Vec
	}{{"crank-gear", 20.6, state.Crank}, {"intermediate-gear", 12.6, state.Intermediate}} {
		axis, err := handcranksiren.PosePoint(p, map[string]string{"crank-gear": "crank", "intermediate-gear": "intermediate"}[g.name], g.axis)
		require.NoError(t, err)
		body := parts[g.name].Body
		reach := func(y float64) float64 {
			outline := outerOutline(t, body, frameAt(t, r3.Vec{Y: y}, r3.Vec{X: 1}, r3.Vec{Z: 1}), axis)
			_, far := loopDistances(outline.Loops[0], axis)
			return far
		}
		mid := reach(g.front + 3)
		require.GreaterOrEqual(t, mid-reach(g.front+0.1), 0.05, "%s front cap", g.name)
		require.GreaterOrEqual(t, mid-reach(g.front+5.9), 0.05, "%s back cap", g.name)
	}
	// 15T and 16T gears: the net cap face is below the large-face threshold, so no chamfer is required.
	for _, name := range []string{"intermediate-pinion", "rotor-pinion"} {
		for _, f := range parts[name].Body.Faces() {
			plane, ok := f.Surface().(decad.Plane)
			if !ok || math.Abs(math.Abs(plane.Frame.N().Y)-1) > 1e-9 {
				continue
			}
			area, err := f.Area()
			require.NoError(t, err)
			require.Less(t, area.Value.Base(), edgefinish.LargeFaceArea, "%s cap face", name)
		}
	}
}

// armSharpEdge reports whether an edge of the crank arm's faces is one crankhandle keeps sharp: both its ends, read in
// crankhandle's local frame, lie on the slit's walls or end round (within SlitWidth/2 of x = 0, behind the shaft axis),
// on the bore's outline (within r_b of the axis), or on the ramp's foot (y = RampFoot, z = ArmThickness: a concave edge).
func armSharpEdge(t *testing.T, p handcranksiren.Params) func(*decad.Edge) bool {
	t.Helper()
	toLocal, err := handcranksiren.CrankLocal(p)
	require.NoError(t, err)
	dm, err := crankhandle.Params6082RS().Derive()
	require.NoError(t, err)
	const slack = 1e-6
	sharp := func(v r3.Vec) bool {
		l := toLocal.Apply(v)
		inSlit := math.Abs(l.X) <= crankhandle.SlitWidth/2+slack && l.Y < 0
		return inSlit || math.Hypot(l.X, l.Y) <= dm.BoreRadius+slack
	}
	foot := func(v r3.Vec) bool {
		l := toLocal.Apply(v)
		return math.Abs(l.Y-dm.RampFoot) <= slack && math.Abs(l.Z-crankhandle.ArmThickness) <= slack
	}
	// The boss top's edge with the ramp is chamfered, but its 15 degree chamfer face is nearly parallel to the top, past
	// bevelled's 0.9 limit on the cosine, so the check cannot read it. crankhandle's TestCrankHandleEdgeFinishing proves
	// that chamfer in a section 0.1 mm under the top.
	rampTop := func(v r3.Vec) bool {
		l := toLocal.Apply(v)
		return math.Abs(l.Y-(crankhandle.ArmRadius-crankhandle.ChamferSize)) <= slack && math.Abs(l.Z-crankhandle.HubLength) <= slack
	}
	return func(e *decad.Edge) bool {
		a, b := e.Start().Position().Value, e.End().Position().Value
		return sharp(a) && sharp(b) || foot(a) && foot(b) || rampTop(a) && rampTop(b)
	}
}

// isShaftHole reports whether e is the circle round the rotor axis (radius 4.7 after the chamfer).
func isShaftHole(e *decad.Edge) bool {
	c, ok := e.Curve().(decad.Circle3)
	return ok && math.Abs(c.Radius.Base()-4.7) < 1e-6
}
