package sevensegment_test

// CI: isolated-tests

import (
	goreflect "reflect"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentFlagReuse checks that the flags of the placed pose at (10, 20, 30) are the baseline flags turned and
// placed, and that the build files and constrains as a plain one does. The placed parts come from the shared posed
// build. Their bodies sit in no container, and the filing and the sketches do not depend on the placement, so the
// containers, the body count and the sketches are checked on the default display of the same build.
func TestSevenSegmentFlagReuse(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	p := sevensegment.DefaultParams()
	placeCase := 1
	require.Equal(t, r3.Vec{X: 10, Y: 20, Z: 30}, placementCases[placeCase].origin)
	frame, err := placementCases[placeCase].frame()
	require.NoError(t, err)
	p.Placement = &frame
	dm, err := p.Derive()
	require.NoError(t, err)

	doc, model := builtDefault(t)
	placedParts := builtPlaced(t, placeCase)
	require.Equal(t, 78, len(doc.Bodies()))
	require.NoError(t, doc.VerifyContainers().Err())
	for i, s := range model.Sketches {
		report := s.Verify(ctx)
		require.Equal(t, sketch.FullyConstrained, report.Status, "sketch %d", i)
		require.Equal(t, 0, report.DOF, "sketch %d", i)
		require.Empty(t, report.Redundant, "sketch %d", i)
		require.Empty(t, report.Conflicts, "sketch %d", i)
	}
	require.Equal(t, len(model.Sketches), len(placedParts.Sketches))

	for _, seg := range sevensegment.Segments() {
		want, err := sevensegment.FlagBaseline(ctx, p, seg)
		require.NoError(t, err)
		cx, cy, horizontal := segmentCentre(seg)
		axis := r3.Vec{Y: 1}
		if horizontal {
			axis = r3.Vec{X: 1}
		}
		turn, err := r3.RotationAround(r3.Vec{X: cx, Y: cy, Z: sevensegment.RotorAxisZ}, axis,
			units.Degrees(dm.FlagAngle(seg, dm.Theta)))
		require.NoError(t, err)
		for _, part := range []struct {
			name             string
			want, plain, got *decad.Body
		}{
			{sevensegment.YellowFaceContainerName, want.Yellow, model.Flags[seg].Yellow, placedParts.Flags[seg].Yellow},
			{sevensegment.BlackBodyContainerName, want.Black, model.Flags[seg].Black, placedParts.Flags[seg].Black},
		} {
			c, err := doc.ContainerOf(part.plain)
			require.NoError(t, err)
			require.Equal(t, part.name, c.Name())
			require.Equal(t, sevensegment.FlagsContainerName, c.Parent().Parent().Name())
			posed, err := decad.Transform(part.want, turn).Do(ctx)
			require.NoError(t, err)
			placed, err := placement.Place(ctx, posed, p.Placement)
			require.NoError(t, err)
			a, err := placed.Tessellate(ctx, units.Millimeters(0.05), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			b, err := part.got.Tessellate(ctx, units.Millimeters(0.05), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, goreflect.DeepEqual(a.Vertices(), b.Vertices()), "vertices %s %s", seg, part.name)
			require.True(t, goreflect.DeepEqual(a.Triangles(), b.Triangles()), "triangles %s %s", seg, part.name)
		}
	}
}
