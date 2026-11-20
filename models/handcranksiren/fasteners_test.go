package handcranksiren_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// extent returns the least and the greatest offset along the unit vector d of body's mesh vertices, and the mesh's
// bound. A bolt's head top, its tip and a nut's faces and corners are planar faces or edges of the body, so their
// vertices are exact; the crank's fasteners turn with it, so their axes need not lie on a world axis.
func extent(t *testing.T, body *decad.Body, d r3.Vec) (lo, hi, bound float64) {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.01), decad.WithVerification(decad.VerifyNone))
	require.NoError(t, err)
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, v := range mesh.Vertices() {
		lo, hi = math.Min(lo, v.Dot(d)), math.Max(hi, v.Dot(d))
	}
	return lo, hi, mesh.Bound().Base()
}

// headPocketMid is the distance from the head top to the middle of the head pocket.
const headPocketMid = m2fastener.HeadPocketDepth / 2

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

// openPocketNut names the nuts that sit in a hexagonal pocket open to their part's face instead of an enclosed cavity:
// the crank arm's two clamp nuts and the crank pin's two cross nuts, which their bolts hold in (models/crankhandle). A
// pocket runs from CavityNear to the face at CavityFar, and the bolt's tip ends flush with that face. Each clamp nut's
// pocket runs deeper: it reaches the face 2 mm past CavityFar (pocketFace), and the bolt's tip ends at CavityFar,
// inside the pocket.
var openPocketNut = map[string]struct{}{
	"crank-clamp-nut-1": {}, "crank-clamp-nut-2": {}, "crank-cross-nut-1": {}, "crank-cross-nut-2": {},
}

// pocketFace gives the distance from Head to the face an openPocketNut's pocket opens on, where it is not CavityFar.
var pocketFace = map[string]float64{"crank-clamp-nut-1": 20, "crank-clamp-nut-2": 20}

// requireFasteners proves for each fastener: the head ends flush with its part's face, the nut sits inside the cavity
// its part encloses (or, for an openPocketNut, inside its hexagonal pocket below the face), the part keeps 1 mm of material
// round the pocket and the cavity, the bolt's tip stays in the part, and no fastener shares material with the parts it
// passes through.
func requireFasteners(t *testing.T, doc *decad.Document, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	all, err := handcranksiren.Fasteners(p)
	require.NoError(t, err)
	require.Len(t, all, 28)
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
		top, _, bound := extent(t, bolt, f.Dir)
		require.InDelta(t, f.Head.Dot(f.Dir), top, 2e-3+bound, "%s head is flush", f.Bolt)
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
			require.True(t, pocket.contains(ring.Scale(m2fastener.HeadPocketRadius+material)),
				"%s: the part keeps 1 mm round the head pocket at %d degrees", f.Bolt, k*30)
		}

		// The nut floats in its cavity, and the part encloses the cavity on every side but the bolt hole.
		flat := f.Dir.Cross(f.Corner)
		mid := (f.CavityNear + f.CavityFar) / 2
		centre := f.Head.Add(f.Dir.Scale(mid))
		var off [3]float64
		for i, d := range []r3.Vec{f.Dir, f.Corner, flat} {
			lo, hi, bound := extent(t, nut, d)
			if i == 0 {
				require.InDelta(t, m2fastener.NutSize.Thickness, hi-lo, 2e-3+bound, "%s thickness", f.Nut)
			}
			off[i] = (lo+hi)/2 - centre.Dot(d)
			require.Less(t, math.Abs(off[i]), 0.01+bound, "%s is centred in its cavity", f.Nut)
		}
		// The cavity's section is a hole loop with the part's material round it. The floor and the roof beside the
		// cavity are solid except for the bolt hole, so the cavity is closed on every side but the hole.
		_, open := openPocketNut[f.Nut]
		wall := host(f, nutHost, mid)
		require.False(t, wall.contains(r3.Vec{}), "%s: the cavity is empty", f.Nut)
		require.True(t, wall.contains(f.Corner.Scale(3.0)), "%s: the cavity's corner wall is solid", f.Nut)
		require.True(t, wall.contains(flat.Scale(2.6)), "%s: the cavity's flat wall is solid", f.Nut)
		require.False(t, wall.contains(f.Corner.Scale(2.4)), "%s: the cavity has the nut's room", f.Nut)
		// Round the cavity the part keeps 1 mm beyond its radius (the circle through the nut's corners plus 0.2 mm).
		cavityRadius := m2fastener.NutCavityRadius
		for k := range 12 {
			angle := float64(k) * math.Pi / 6
			ring := f.Corner.Scale(math.Cos(angle)).Add(flat.Scale(math.Sin(angle)))
			require.True(t, wall.contains(ring.Scale(cavityRadius+material)),
				"%s: the part keeps 1 mm round the cavity at %d degrees", f.Nut, k*30)
		}
		layers := map[string]float64{"roof": f.CavityFar + 0.5, "floor": f.CavityNear - 0.5}
		if open {
			delete(layers, "roof")
		}
		for name, d := range layers {
			layer := host(f, nutHost, d)
			require.True(t, layer.contains(f.Corner.Scale(1.6)), "%s: the %s of the cavity is solid", f.Nut, name)
			require.False(t, layer.contains(r3.Vec{}), "%s: the %s has the bolt hole", f.Nut, name)
		}
		tip := m2fastener.BoltSize.HeadHeight + f.Length
		if open {
			// The pocket opens to the face at CavityFar: the part ends there, the nut ends below it, and the bolt's tip
			// ends flush with it, through the whole nut.
			require.InDelta(t, f.CavityFar, tip, 1e-9, "%s: the tip reaches the nut cavity's far face", f.Bolt)
			face, ok := pocketFace[f.Nut]
			if !ok {
				face = f.CavityFar
			}
			require.True(t, host(f, nutHost, face-0.05).contains(f.Corner.Scale(3.0)), "%s: the part runs to the face", f.Nut)
			require.False(t, host(f, nutHost, face-0.05).contains(f.Corner.Scale(1.6)), "%s: the pocket reaches the face", f.Nut)
			require.False(t, host(f, nutHost, face+0.05).contains(f.Corner.Scale(3.0)), "%s: the part ends at the face", f.Nut)
			_, nutEnd, nb := extent(t, nut, f.Dir)
			require.LessOrEqual(t, nutEnd, f.Head.Dot(f.Dir)+f.CavityFar+nb, "%s ends below the face", f.Nut)
			_, end, bb := extent(t, bolt, f.Dir)
			require.InDelta(t, f.Head.Dot(f.Dir)+tip, end, 2e-3+bb, "%s: tip ends at the nut cavity's far face", f.Bolt)
			require.Greater(t, end, nutEnd, "%s runs through the whole nut", f.Bolt)
		} else {
			require.True(t, host(f, nutHost, tip-0.3).contains(f.Corner.Scale(1.6)), "%s: the tip is held by the part", f.Bolt)
		}
		if !open {
			require.True(t, host(f, nutHost, tip+0.3).contains(f.Corner.Scale(1.6)), "%s: the tip stays inside the part", f.Bolt)
		}

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
		if open {
			// Past the open pocket's face there is no part: read the shank in the hole before the pocket's floor.
			gap("shank in the nut host's hole", bolt, nutHost, f.CavityNear-0.5)
		} else {
			gap("shank in the nut host's hole", bolt, nutHost, f.CavityFar+0.3)
		}
	}
}
