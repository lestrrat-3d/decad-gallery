package clamparm_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

const (
	clampMeshTol  = 0.02
	clampLocTol   = 0.005
	clampDistTol  = 0.02
	clampFrameLow = -73.5
)

func clampParams(opening float64) clamparm.Params {
	p := clamparm.DefaultParams()
	p.Opening = opening
	return p
}

func clampFrame(t *testing.T) *decad.Body {
	t.Helper()
	body, sketches, err := clamparm.BuildClamp(t.Context(), clamparm.NewDocument(), clamparm.DefaultParams())
	require.NoError(t, err)
	require.True(t, body.IsSolid())
	// The profile, the J1 hole and head pocket, the clamp screw hole and the jaw nut pocket.
	require.Len(t, sketches, 5)
	requireFullyConstrained(t, sketches)
	return body
}

func clampMesh(t *testing.T, body *decad.Body) meshgap.Mesh {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(clampMeshTol))
	require.NoError(t, err)
	mg, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return mg
}

func requireLocation(t *testing.T, body *decad.Body, p r3.Vec, want decad.PointLocation) {
	t.Helper()
	got, err := body.LocatePoint(t.Context(), p, units.Millimeters(clampLocTol))
	require.NoError(t, err)
	require.Equal(t, want, got, "point %v", p)
}

func requireDistance(t *testing.T, body *decad.Body, p r3.Vec, want float64) {
	t.Helper()
	m, err := body.DistanceToPoint(t.Context(), p, units.Millimeters(clampLocTol))
	require.NoError(t, err)
	// The measurement encloses the true distance: Value is within Bound of it.
	require.InDelta(t, want, m.Value.Base(), m.Bound.Base()+1e-6, "point %v", p)
}

func TestClampArmClampFrameBounds(t *testing.T) {
	body := clampFrame(t)
	requireBounds(t, body, r3.NewVec(-14, -15, clampFrameLow), r3.NewVec(50, 15, 28.26))
}

func TestClampArmClampJ1Gap(t *testing.T) {
	body := clampFrame(t)
	require.InDelta(t, 8.52, clamparm.ForkGap, 1e-9)
	// The gap runs z 14 to 22.52 at x = 20: material 0.1 beyond each face, open just inside it.
	requireLocation(t, body, r3.NewVec(20, 0, 12), decad.PointInside)
	requireLocation(t, body, r3.NewVec(20, 0, 25), decad.PointInside)
	requireDistance(t, body, r3.NewVec(20, 0, 14.1), 0.1)
	requireDistance(t, body, r3.NewVec(20, 0, 22.42), 0.1)
	// The mid-plane of the gap is 4.26 from both faces.
	requireDistance(t, body, r3.NewVec(20, 0, 18.26), 4.26)
	// The gap ends in a full round whose deepest point is x = -5, and it is open at x = 46.
	requireDistance(t, body, r3.NewVec(-4.5, 0, 18.26), 0.5)
	requireLocation(t, body, r3.NewVec(47, 0, 18.26), decad.PointOutside)
	requireDistance(t, body, r3.NewVec(-0.74, 0, 18.26), 4.26)
}

func TestClampArmClampJawRootRounds(t *testing.T) {
	body := clampFrame(t)
	// The round of radius 7 at each jaw root is tangent to the spine and the jaw: its centre is 7 from the material.
	requireDistance(t, body, r3.NewVec(7, 0, -7), 7)
	requireDistance(t, body, r3.NewVec(7, 0, jawTopZ+7), 7)
	// The round fills the corner out to 2.9 mm from the corner along the diagonal, and no further.
	requireLocation(t, body, r3.NewVec(1.5, 0, -1.5), decad.PointInside)
	requireLocation(t, body, r3.NewVec(1.0, 0, -1.0), decad.PointInside)
	requireLocation(t, body, r3.NewVec(3.5, 0, -3.5), decad.PointOutside)
	requireLocation(t, body, r3.NewVec(1.5, 0, jawTopZ+1.5), decad.PointInside)
	requireLocation(t, body, r3.NewVec(3.5, 0, jawTopZ+3.5), decad.PointOutside)
}

const jawTopZ = -59.5

func TestClampArmClampPad(t *testing.T) {
	for _, o := range []float64{clamparm.OpeningMin, clamparm.OpeningMax} {
		t.Run("opening", func(t *testing.T) {
			p := clampParams(o)
			d, err := p.Derive()
			require.NoError(t, err)
			ctx := t.Context()
			doc := clamparm.NewDocument()
			frameBody, _, err := clamparm.BuildClamp(ctx, doc, p)
			require.NoError(t, err)
			pad, sketches, err := clamparm.BuildPad(ctx, doc, p)
			require.NoError(t, err)
			require.True(t, pad.IsSolid())
			// The outline and the socket's meridian.
			require.Len(t, sketches, 2)
			requireFullyConstrained(t, sketches)
			requireBounds(t, pad, r3.NewVec(-24.5, -15, -12), r3.NewVec(15, 15, 0))
			pad = place(ctx, t, pad, d.PadFrame)
			requireBounds(t, pad, r3.NewVec(0.5, -15, -o-12), r3.NewVec(40, 15, -o))

			// The pad stands 0.5 from the spine and clears the jaw root round at the lowest position.
			separated, ok, err := clampMesh(t, pad).Exceeds(clampMesh(t, frameBody), 0.45)
			require.NoError(t, err)
			require.True(t, ok, "pad is closer than 0.45 to the frame: gap bound %g", separated)
		})
	}
}

func TestClampArmClampScrewKnob(t *testing.T) {
	for _, o := range []float64{clamparm.OpeningMin, 25, clamparm.OpeningMax} {
		p := clampParams(o)
		d, err := p.Derive()
		require.NoError(t, err)
		ctx := t.Context()
		doc := clamparm.NewDocument()
		knob, sketches, err := clamparm.BuildScrewKnob(ctx, doc, p)
		require.NoError(t, err)
		require.True(t, knob.IsSolid())
		// The outline, the head pocket and two part marks.
		require.Len(t, sketches, 4)
		requireFullyConstrained(t, sketches)
		requireBounds(t, knob, r3.NewVec(-15, -13.526, 0), r3.NewVec(15, 13.526, 10))

		knob = place(ctx, t, knob, d.ScrewKnobFrames[0])
		kb, err := knob.Bounds()
		require.NoError(t, err)
		// The knob hangs below the jaw: its inner face is ScrewKnobGap under the bottom jaw's underside.
		gap := clampFrameLow - kb.Max.Z
		require.InDelta(t, d.ScrewKnobGap, gap, boundTol, "opening %g", o)
		require.GreaterOrEqual(t, gap, 0.5)
		require.InDelta(t, -o-77.5, kb.Min.Z, boundTol)
	}
}

// clampScrew is the clamp screw at an Opening: the frame, the jaw nut, the screw, the screw knob and the pad, posed.
type clampScrew struct {
	frame, nut, bolt, knob, pad *decad.Body
	site                        m5fastener.Site
}

func buildClampScrew(t *testing.T, o float64) clampScrew {
	t.Helper()
	ctx := t.Context()
	p := clampParams(o)
	d, err := p.Derive()
	require.NoError(t, err)
	doc := clamparm.NewDocument()
	var c clampScrew
	c.frame, _, err = clamparm.BuildClamp(ctx, doc, p)
	require.NoError(t, err)
	knob, _, err := clamparm.BuildScrewKnob(ctx, doc, p)
	require.NoError(t, err)
	c.knob = place(ctx, t, knob, d.ScrewKnobFrames[0])
	pad, _, err := clamparm.BuildPad(ctx, doc, p)
	require.NoError(t, err)
	c.pad = place(ctx, t, pad, d.PadFrame)
	c.site = d.ScrewSite
	bolt, err := m5fastener.PlaceBolt(ctx, doc, c.site)
	require.NoError(t, err)
	c.bolt = bolt.Body
	nut, err := m5fastener.PlaceNut(ctx, doc, d.JawNutFrame)
	require.NoError(t, err)
	c.nut = nut.Body
	return c
}

func TestClampArmClampJawNutAndScrew(t *testing.T) {
	for _, o := range []float64{clamparm.OpeningMin, 25, clamparm.OpeningMax} {
		c := buildClampScrew(t, o)
		fb, err := c.frame.Bounds()
		require.NoError(t, err)
		nb, err := c.nut.Bounds()
		require.NoError(t, err)
		bb, err := c.bolt.Bounds()
		require.NoError(t, err)
		kb, err := c.knob.Bounds()
		require.NoError(t, err)

		// The nut sits on the pocket floor, 4.7 tall, below the jaw's top face and inside its outline.
		require.InDelta(t, -64.6, nb.Min.Z, boundTol)
		require.InDelta(t, -64.6+4.7, nb.Max.Z, boundTol)
		require.Less(t, nb.Max.Z, jawTopZ)
		require.Greater(t, nb.Min.X, fb.Min.X)
		require.Less(t, nb.Max.X, 45.0)
		require.Greater(t, nb.Min.Y, fb.Min.Y)
		require.Less(t, nb.Max.Y, fb.Max.Y)

		// The screw runs from the knob's outer face, flush with its head, up past the nut to the pad socket floor.
		require.InDelta(t, kb.Min.Z, bb.Min.Z, boundTol, "head flush in the knob, opening %g", o)
		require.Greater(t, bb.Min.X, kb.Min.X)
		require.Less(t, bb.Max.X, kb.Max.X)
		require.Greater(t, bb.Min.Y, kb.Min.Y)
		require.Less(t, bb.Max.Y, kb.Max.Y)
		require.InDelta(t, -o-4, bb.Max.Z, boundTol)
		require.Less(t, bb.Min.Z, nb.Min.Z)
		require.Greater(t, bb.Max.Z, nb.Max.Z)
	}

	t.Run("no overlap", func(t *testing.T) {
		for _, pair := range []struct {
			name string
			a, b func(clampScrew) *decad.Body
		}{
			{"screw and frame", func(c clampScrew) *decad.Body { return c.bolt }, func(c clampScrew) *decad.Body { return c.frame }},
			{"nut and frame", func(c clampScrew) *decad.Body { return c.nut }, func(c clampScrew) *decad.Body { return c.frame }},
			{"screw and pad", func(c clampScrew) *decad.Body { return c.bolt }, func(c clampScrew) *decad.Body { return c.pad }},
		} {
			t.Run(pair.name, func(t *testing.T) {
				c := buildClampScrew(t, 25)
				requireDisjoint(t, pair.a(c), pair.b(c))
			})
		}
	})
}

func TestClampArmClampPadSocketFit(t *testing.T) {
	for _, o := range []float64{clamparm.OpeningMin, 25, clamparm.OpeningMax} {
		c := buildClampScrew(t, o)
		// A point 0.1 outside the shank, half way up the socket on the screw axis, is 0.1 from the shank and 0.25 from
		// the socket wall: the radial fit is 0.35.
		p := r3.NewVec(clamparm.ScrewX+2.6, 0, -o-8)
		socket, err := c.pad.DistanceToPoint(t.Context(), p, units.Millimeters(clampLocTol))
		require.NoError(t, err)
		shank, err := c.bolt.DistanceToPoint(t.Context(), p, units.Millimeters(clampLocTol))
		require.NoError(t, err)
		require.InDelta(t, 0.1, shank.Value.Base(), shank.Bound.Base()+1e-6, "opening %g", o)
		require.InDelta(t, 0.25, socket.Value.Base(), socket.Bound.Base()+1e-6, "opening %g", o)
	}
}
