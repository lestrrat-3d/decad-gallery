package spinningwheel_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSpinningWheelConnectorDefault(t *testing.T) {
	p := spinningwheel.DefaultParams()
	doc := spinningwheel.NewDocument()
	connector, err := spinningwheel.BuildConnector(t.Context(), doc, p)
	require.NoError(t, err)
	require.Equal(t, 1, len(doc.Bodies()), "only the one-piece connector may remain live")
	require.True(t, connector.Body.IsSolid())
	mesh, err := connector.Body.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	vertices, triangles := mesh.Vertices(), mesh.Triangles()
	require.Equal(t, 1, meshComponents(len(vertices), triangles), "the two legs must form one physical piece")
	require.Equal(t, -2, meshEulerCharacteristic(triangles), "both transverse clamp passages must remain open")
	require.Greater(t, len(connector.Sketches), 0)
	for i, section := range connector.Sketches {
		report := section.Verify(t.Context())
		require.True(t, report.Status == sketch.FullyConstrained, "connector sketch %d status %s", i, report.Status)
		require.Zero(t, report.DOF, "connector sketch %d", i)
		require.Zero(t, len(report.Redundant), "connector sketch %d", i)
		require.Zero(t, len(report.Conflicts), "connector sketch %d", i)
		require.NoError(t, report.Check(), "connector sketch %d", i)
	}
	state, err := spinningwheel.Motion(p)
	require.NoError(t, err)
	material := surfaceTriangles{vertices: vertices, triangles: triangles, bound: mesh.Bound().Base()}
	for _, end := range []struct {
		name              string
		centre, leg, bore r3.Vec
	}{
		{"lower", state.Toe.Sub(state.PedalFrame.N().Scale(12 * p.Scale)), state.PedalFrame.N(), state.PedalFrame.U()},
		{"upper", state.Toe.Add(state.RodFrame.U().Scale(26 * p.Scale)), state.RodFrame.U(), state.RodFrame.V()},
	} {
		for _, offset := range []float64{-2.6, 0, 2.6} {
			require.False(t, material.contains(end.centre.Add(end.bore.Scale(offset))),
				"%s clamp passage must be open along its actual transverse axis at %g mm", end.name, offset)
		}
		require.True(t, material.contains(end.centre.Add(end.leg.Scale(2.6))),
			"%s leg needs material outside its clamp bore", end.name)
	}
}
