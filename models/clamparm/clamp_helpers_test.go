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

const jawTopZ = -59.5

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

func screwBody(c clampScrew) *decad.Body { return c.bolt }

func nutBody(c clampScrew) *decad.Body { return c.nut }

func frameBody(c clampScrew) *decad.Body { return c.frame }

func padBody(c clampScrew) *decad.Body { return c.pad }
