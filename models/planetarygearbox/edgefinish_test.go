package planetarygearbox_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// largeFlatArea is the net area, in square millimeters, from which a flat face is large (Surface edge finishing).
const largeFlatArea = 400

// largeFlatFaces returns the planar faces of body that face along Z and have at least largeFlatArea of area.
func largeFlatFaces(t *testing.T, body *decad.Body) []*decad.Face {
	t.Helper()
	var out []*decad.Face
	for _, f := range body.Faces() {
		plane, ok := f.Surface().(decad.Plane)
		if !ok || math.Abs(math.Abs(plane.Frame.N().Z)-1) > 1e-9 {
			continue
		}
		area, err := f.Area()
		require.NoError(t, err)
		if area.Value.Base() >= largeFlatArea {
			out = append(out, f)
		}
	}
	return out
}

// bevelAngle reports whether the face beside a rim edge is a 45° bevel: a plane tilted 45° from Z or a cone of
// 45° half angle.
func bevelAngle(t *testing.T, f *decad.Face) bool {
	t.Helper()
	switch s := f.Surface().(type) {
	case decad.Plane:
		return math.Abs(math.Abs(s.Frame.N().Z)-math.Sqrt2/2) < 1e-6
	case decad.Cone:
		half, err := s.HalfAngle.In(units.Radians(1).Unit())
		require.NoError(t, err)
		return math.Abs(half-math.Pi/4) < 1e-6
	}
	return false
}

// requireOuterEdgesChamfered checks that every large flat face of body has a 45° bevel along its whole outer
// boundary and that the body has at least faces of them.
func requireOuterEdgesChamfered(t *testing.T, name string, body *decad.Body, faces int) {
	t.Helper()
	flat := largeFlatFaces(t, body)
	require.GreaterOrEqual(t, len(flat), faces, "%s: large flat faces", name)
	for _, f := range flat {
		for _, loop := range f.Loops() {
			if !loop.IsOuter() {
				continue
			}
			for _, e := range loop.Edges() {
				beside := 0
				for _, other := range e.Faces() {
					if other != f && bevelAngle(t, other) {
						beside++
					}
				}
				require.Equal(t, 1, beside, "%s: an outer edge of a large flat face has no 45° bevel", name)
			}
		}
	}
}

// requireRimSetback checks that the circular outer edges of the large flat faces of body sit the chamfer
// setback inside the widest cylinder of the body.
func requireRimSetback(t *testing.T, name string, body *decad.Body, setback float64) {
	t.Helper()
	outer := 0.0
	for _, f := range body.Faces() {
		if cyl, ok := f.Surface().(decad.Cylinder); ok {
			outer = math.Max(outer, cyl.Radius.Base())
		}
	}
	require.Positive(t, outer, "%s: no cylinder", name)
	rims := 0
	for _, f := range largeFlatFaces(t, body) {
		for _, loop := range f.Loops() {
			if !loop.IsOuter() {
				continue
			}
			for _, e := range loop.Edges() {
				at := e.Start().Position().Value
				require.InDelta(t, outer-setback, math.Hypot(at.X, at.Y), 1e-4, "%s: rim radius", name)
				rims++
			}
		}
	}
	require.GreaterOrEqual(t, rims, 2, "%s: chamfered rim edges", name)
}

// TestEdgeFinishing checks Surface edge finishing on the ring, covers and carrier: a 45° chamfer of EdgeChamfer
// along the outer boundary of each large flat face.
func TestEdgeFinishing(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	require.InDelta(t, 0.5, p.EdgeChamfer, 0)
	require.GreaterOrEqual(t, p.EdgeChamfer, planetarygearbox.MinEdgeChamfer)
	_, box := defaultGearbox(t)

	requireOuterEdgesChamfered(t, "ring", box.Ring, 2)
	requireRimSetback(t, "ring", box.Ring, p.EdgeChamfer)
	for name, cover := range map[string]*decad.Body{"front cover": box.FrontCover, "rear cover": box.RearCover} {
		requireOuterEdgesChamfered(t, name, cover, 2)
		requireRimSetback(t, name, cover, p.EdgeChamfer)
	}
	for k, stage := range box.Stages {
		requireOuterEdgesChamfered(t, planetarygearbox.CarrierName(k), stage.Carrier, 2)
	}
}

// TestEdgeChamferLimits checks that a chamfer below the edge finishing minimum is refused.
func TestEdgeChamferLimits(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	p.EdgeChamfer = planetarygearbox.MinEdgeChamfer - 0.01
	require.ErrorContains(t, p.Validate(), "edge chamfer")
	p.EdgeChamfer = 1.01
	require.ErrorContains(t, p.Validate(), "edge chamfer")
	p.EdgeChamfer = 1
	require.NoError(t, p.Validate())
}
