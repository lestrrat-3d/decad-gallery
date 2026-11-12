package handcranksiren_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// outward returns the extent of box in the direction n, which is a signed unit axis.
func outward(box decad.Box, n r3.Vec) float64 {
	switch {
	case n.X > 0.5:
		return box.Max.X
	case n.X < -0.5:
		return -box.Min.X
	case n.Y > 0.5:
		return box.Max.Y
	case n.Y < -0.5:
		return -box.Min.Y
	case n.Z > 0.5:
		return box.Max.Z
	}
	return -box.Min.Z
}

// along returns the extent of box along the axis that the unit vector d lies on.
func along(box decad.Box, d r3.Vec) (lo, hi float64) {
	switch {
	case d.X > 0.5 || d.X < -0.5:
		return box.Min.X, box.Max.X
	case d.Y > 0.5 || d.Y < -0.5:
		return box.Min.Y, box.Max.Y
	}
	return box.Min.Z, box.Max.Z
}

// headPocketMid is the distance from the head top to the middle of the head pocket.
const headPocketMid = m2bolt.HeadHeight / 2

// acrossPlane is the plane at distance t from the head across the shaft, with the corner direction as its U axis.
func acrossPlane(t *testing.T, f handcranksiren.Fastener, d float64) r3.Frame {
	return frameAt(t, f.Head.Add(f.Dir.Scale(d)), f.Corner, f.Dir.Cross(f.Corner))
}

// sectionAt is a body's section across a bolt's axis. contains reports whether a point, given as an offset from the
// axis in the plane, lies in the section's material: the loops are polygons of the mesh, so the test is exact for them
// and does not depend on the body's point location near its boundaries.
type sectionAt struct {
	outline      decad.SectionOutline
	origin, u, v r3.Vec
}

func (s sectionAt) contains(offset r3.Vec) bool {
	px, py := offset.Dot(s.u), offset.Dot(s.v)
	crossings := 0
	for _, loop := range s.outline.Loops {
		for i, a := range loop {
			b := loop[(i+1)%len(loop)]
			ax, ay := a.Sub(s.origin).Dot(s.u), a.Sub(s.origin).Dot(s.v)
			bx, by := b.Sub(s.origin).Dot(s.u), b.Sub(s.origin).Dot(s.v)
			if (ay > py) != (by > py) && px < ax+(py-ay)/(by-ay)*(bx-ax) {
				crossings++
			}
		}
	}
	return crossings%2 == 1
}

// requireFasteners proves for each fastener: the head ends flush with its part's face, the nut sits inside the cavity
// its part encloses, the part keeps 1 mm of material round the pocket and the cavity, the bolt's tip stays in the part,
// and no fastener shares material with the parts it passes through.
func requireFasteners(t *testing.T, doc *decad.Document, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	all, err := handcranksiren.Fasteners(p)
	require.NoError(t, err)
	require.Len(t, all, 24)
	request := decad.ContactRequest{
		PointResolution:  units.Millimeters(0.01),
		NormalResolution: units.Radians(0.01),
		HeldChord:        units.Millimeters(0.01),
	}
	for _, f := range all {
		bolt, nut := parts[f.Bolt].Body, parts[f.Nut].Body
		headHost, nutHost := parts[f.HeadHost].Body, parts[f.NutHost].Body
		host := func(f handcranksiren.Fastener, body *decad.Body, d float64) sectionAt {
			plane := acrossPlane(t, f, d)
			outline, err := body.SectionOutline(t.Context(), plane, units.Millimeters(fitTolerance))
			require.NoError(t, err, f.Bolt)
			return sectionAt{outline: outline, origin: plane.Origin(), u: f.Corner, v: f.Dir.Cross(f.Corner)}
		}

		// Flush: the head's top face lies on the part's face.
		box := bounds(t, parts[f.Bolt])
		require.InDelta(t, f.Head.Dot(f.Dir.Scale(-1)), outward(box, f.Dir.Scale(-1)), 2e-3+box.Bound.Base(), "%s head is flush", f.Bolt)
		// The part's section just under the face has material beside the head's pocket and none around it on the
		// face's other side, so the face is the head's top plane. The pocket itself is empty.
		under, over := host(f, headHost, 0.3), host(f, headHost, -0.3)
		require.True(t, under.contains(f.Corner.Scale(2.6)), "%s: the part is solid just under the head's face", f.Bolt)
		require.False(t, under.contains(r3.Vec{}), "%s: the head pocket is empty", f.Bolt)
		require.False(t, over.contains(f.Corner.Scale(2.6)), "%s: nothing stands proud of the face", f.Bolt)

		// The part keeps 1 mm of material round the pocket, measured at the section across its middle. The test samples
		// 0.05 short of the millimetre.
		const material = 0.95
		pocket := host(f, headHost, headPocketMid)
		for k := range 12 {
			angle := float64(k) * math.Pi / 6
			ring := f.Corner.Scale(math.Cos(angle)).Add(f.Dir.Cross(f.Corner).Scale(math.Sin(angle)))
			require.True(t, pocket.contains(ring.Scale(m2bolt.HeadDiameter/2+0.3+material)),
				"%s: the part keeps 1 mm round the head pocket at %d degrees", f.Bolt, k*30)
		}

		// The nut floats in its cavity, and the part encloses the cavity on every side but the bolt hole.
		nutBox := bounds(t, parts[f.Nut])
		lo, hi := along(nutBox, f.Dir)
		require.InDelta(t, m2nut.Thickness, hi-lo, 2e-3+nutBox.Bound.Base(), "%s thickness", f.Nut)
		mid := (f.CavityNear + f.CavityFar) / 2
		centre := f.Head.Add(f.Dir.Scale(mid))
		got := r3.Vec{X: (nutBox.Min.X + nutBox.Max.X) / 2, Y: (nutBox.Min.Y + nutBox.Max.Y) / 2, Z: (nutBox.Min.Z + nutBox.Max.Z) / 2}
		require.Less(t, got.Sub(centre).Len(), 0.01+nutBox.Bound.Base(), "%s is centred in its cavity", f.Nut)
		flat := f.Dir.Cross(f.Corner)
		// The cavity's section is a hole loop with the part's material round it. The floor and the roof beside the
		// cavity are solid except for the bolt hole, so the cavity is closed on every side but the hole.
		wall := host(f, nutHost, mid)
		require.False(t, wall.contains(r3.Vec{}), "%s: the cavity is empty", f.Nut)
		require.True(t, wall.contains(f.Corner.Scale(3.0)), "%s: the cavity's corner wall is solid", f.Nut)
		require.True(t, wall.contains(flat.Scale(2.6)), "%s: the cavity's flat wall is solid", f.Nut)
		require.False(t, wall.contains(f.Corner.Scale(2.4)), "%s: the cavity has the nut's room", f.Nut)
		// Round the cavity the part keeps 1 mm beyond its radius (the circle through the nut's corners plus 0.2 mm).
		cavityRadius := m2nut.AcrossFlats/math.Sqrt(3) + 0.2
		for k := range 12 {
			angle := float64(k) * math.Pi / 6
			ring := f.Corner.Scale(math.Cos(angle)).Add(flat.Scale(math.Sin(angle)))
			require.True(t, wall.contains(ring.Scale(cavityRadius+material)),
				"%s: the part keeps 1 mm round the cavity at %d degrees", f.Nut, k*30)
		}
		for name, d := range map[string]float64{"roof": f.CavityFar + 0.5, "floor": f.CavityNear - 0.5} {
			layer := host(f, nutHost, d)
			require.True(t, layer.contains(f.Corner.Scale(1.6)), "%s: the %s of the cavity is solid", f.Nut, name)
			require.False(t, layer.contains(r3.Vec{}), "%s: the %s has the bolt hole", f.Nut, name)
		}
		tip := m2bolt.HeadHeight + f.Length
		require.True(t, host(f, nutHost, tip-0.3).contains(f.Corner.Scale(1.6)), "%s: the tip is held by the part", f.Bolt)
		require.True(t, host(f, nutHost, tip+0.3).contains(f.Corner.Scale(1.6)), "%s: the tip stays inside the part", f.Bolt)

		// No overlap with the parts the fastener passes through.
		contact := func(a, b *decad.Body) *decad.ContactReport {
			report, err := doc.ContactPair(t.Context(), a, b, r3.Identity(), r3.Identity(), request)
			require.NoError(t, err)
			return report
		}
		seat := contact(bolt, headHost)
		switch seat.Relation {
		case decad.ContactSeparated, decad.ContactTouching:
		case decad.ContactBand:
			require.NotNil(t, seat.Gap, "%s head seat", f.Bolt)
			require.LessOrEqual(t, seat.Gap.Bound.Base(), 0.05, "%s head seat", f.Bolt)
		default:
			require.Failf(t, "head seat", "%s in %s: relation %v, reason %v", f.Bolt, f.HeadHost, seat.Relation, seat.Reason)
		}
		for _, relationPair := range []struct {
			a, b *decad.Body
		}{{bolt, nutHost}, {nut, nutHost}} {
			separated, err := doc.ProvePairSeparated(t.Context(), relationPair.a, relationPair.b,
				r3.Identity(), r3.Identity(), request)
			require.NoError(t, err)
			if separated {
				continue
			}
			relation := contact(relationPair.a, relationPair.b)
			require.NotEqual(t, decad.ContactOverlapping, relation.Relation, "%s: %v", f.Bolt, relation.Reason)
			require.NotEqual(t, decad.ContactUndecided, relation.Relation, "%s: %v", f.Bolt, relation.Reason)
		}
		gap := func(id string, a, b *decad.Body, d float64) {
			got := sectionGap(t, a, b, acrossPlane(t, f, d))
			require.GreaterOrEqual(t, got, fastenerGap, "%s %s", f.Bolt, id)
		}
		gap("head in its pocket", bolt, headHost, 1.0)
		gap("shank in the head host's hole", bolt, headHost, 3.0)
		gap("nut in its cavity", nut, nutHost, f.CavityNear+0.9)
		gap("shank in the nut host's hole", bolt, nutHost, f.CavityFar+0.3)
	}
}
