package sevensegment

// CI: isolated-tests

import (
	goreflect "reflect"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSevenSegmentFlagReuse(t *testing.T) {
	ctx := t.Context()
	p := DefaultParams()
	frame, err := r3.NewFrame(r3.Vec{X: 10, Y: 20, Z: 30}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	p.Placement = &frame
	dm, err := p.Derive()
	require.NoError(t, err)

	doc := NewDocument()
	model, err := Build(ctx, doc, p)
	require.NoError(t, err)
	require.Equal(t, 75, len(doc.Bodies()))
	require.NoError(t, doc.VerifyContainers().Err())
	for i, s := range model.Sketches {
		report := s.Verify(ctx)
		require.Equal(t, sketch.FullyConstrained, report.Status, "sketch %d", i)
		require.Equal(t, 0, report.DOF, "sketch %d", i)
		require.Empty(t, report.Redundant, "sketch %d", i)
		require.Empty(t, report.Conflicts, "sketch %d", i)
	}

	baseline := &run{ctx: ctx, doc: NewDocument(), p: p, dm: dm}
	for _, seg := range Segments() {
		want, err := baseline.flag(seg)
		require.NoError(t, err)
		cx, cy, horizontal := segmentCentre(seg)
		axis := r3.Vec{Y: 1}
		if horizontal {
			axis = r3.Vec{X: 1}
		}
		turn, err := r3.RotationAround(r3.Vec{X: cx, Y: cy, Z: RotorAxisZ}, axis,
			units.Degrees(dm.FlagAngle(seg, dm.Theta)))
		require.NoError(t, err)
		for _, part := range []struct {
			name      string
			want, got *decad.Body
		}{
			{YellowFaceContainerName, want.Yellow, model.Flags[seg].Yellow},
			{BlackBodyContainerName, want.Black, model.Flags[seg].Black},
		} {
			c, err := doc.ContainerOf(part.got)
			require.NoError(t, err)
			require.Equal(t, part.name, c.Name())
			require.Equal(t, FlagsContainerName, c.Parent().Parent().Name())
			posed, err := decad.Transform(ctx, part.want, turn)
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
