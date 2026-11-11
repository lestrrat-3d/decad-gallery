package spinningwheel_test

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

type bearingEnvelope struct {
	axis, centre       r3.Vec
	bore, outer, width float64
	name               string
}

// readBearingEnvelope reads a purchased bearing's native stock envelope from its circles and cylinders.
func readBearingEnvelope(t *testing.T, part spinningwheel.Part) bearingEnvelope {
	t.Helper()
	e := bearingEnvelope{name: part.Name}
	inner, outerRadius := math.Inf(1), 0.0
	for _, face := range part.Body.Faces() {
		if cylinder, ok := face.Surface().(decad.Cylinder); ok {
			inner, outerRadius = math.Min(inner, cylinder.Radius.Base()), math.Max(outerRadius, cylinder.Radius.Base())
			e.axis = cylinder.Axis
		}
	}
	require.Greater(t, outerRadius, 0.0, "%s exposes no native cylinder", part.Name)
	lo, hi := math.Inf(1), math.Inf(-1)
	var sum r3.Vec
	circles := 0
	for _, edge := range part.Body.Edges() {
		if circle, ok := edge.Curve().(decad.Circle3); ok {
			coordinate := circle.Center.Dot(e.axis)
			lo, hi = math.Min(lo, coordinate), math.Max(hi, coordinate)
			sum = sum.Add(circle.Center)
			circles++
		}
	}
	require.Greater(t, circles, 0, "%s exposes no native circle", part.Name)
	e.bore, e.outer, e.width = 2*inner, 2*outerRadius, hi-lo
	e.centre = sum.Scale(1 / float64(circles))
	return e
}

func (e bearingEnvelope) ringPoints(radius, offset float64) []r3.Vec {
	u, ok := e.axis.Cross(r3.Vec{X: 1}).Normalize()
	if !ok {
		u, _ = e.axis.Cross(r3.Vec{Y: 1}).Normalize()
	}
	v := e.axis.Cross(u)
	var points []r3.Vec
	for i := range 12 {
		angle := (float64(i) + .5) * math.Pi / 6
		points = append(points, e.centre.Add(e.axis.Scale(offset)).
			Add(u.Scale(radius*math.Cos(angle))).Add(v.Scale(radius*math.Sin(angle))))
	}
	return points
}

// Only the radial seat is asserted. Axial capture of the 608-2RS press fits is not modelled by a
// shoulder, collar or plate at every end face, so it is not claimed here.
func requireBearingSeats(t *testing.T, a *spinningwheel.Assembly) {
	t.Helper()
	mounts := make(map[string]spinningwheel.Mount, len(a.Mounts))
	for _, m := range a.Mounts {
		mounts[m.Part] = m
	}
	inside := func(part spinningwheel.Part, p r3.Vec) bool {
		box, err := part.Body.Bounds()
		require.NoError(t, err)
		margin := box.Bound.Base() + .05
		if p.X < box.Min.X-margin || p.X > box.Max.X+margin || p.Y < box.Min.Y-margin || p.Y > box.Max.Y+margin ||
			p.Z < box.Min.Z-margin || p.Z > box.Max.Z+margin {
			return false
		}
		location, err := part.Body.LocatePoint(t.Context(), p, units.Millimeters(.05))
		require.NoError(t, err)
		return location == decad.PointInside
	}
	checked := 0
	for _, part := range a.Parts {
		if part.Group != "bearing" {
			continue
		}
		checked++
		e := readBearingEnvelope(t, part)
		mount := mounts[part.Name]
		held := func(radius, offset float64, names []string) int {
			count := 0
			for _, p := range e.ringPoints(radius, offset) {
				for _, other := range a.Parts {
					if other.Name == part.Name || (names != nil && !strings.Contains(" "+strings.Join(names, " ")+" ", " "+other.Name+" ")) {
						continue
					}
					if inside(other, p) {
						count++
						break
					}
				}
			}
			return count
		}
		refs := append(append([]string{}, mount.Supports...), mount.Fasteners...)
		// Both races lie inside the stock envelope, the seat wraps the outer race and a journal fills the bore.
		race := (e.bore/2 + e.outer/2) / 2
		for _, p := range e.ringPoints(race, 0) {
			require.True(t, inside(part, p), "%s race must fill its stock envelope", part.Name)
		}
		require.Equal(t, 12, held(e.outer/2+.15, 0, refs), "%s seat must wrap the outer race", part.Name)
		require.Equal(t, 12, held(e.bore/2-.15, 0, nil), "%s bore must hold a journal", part.Name)
	}
	require.Equal(t, 10, checked, "seven 608-2RS and three 605ZZ bearings")
}
