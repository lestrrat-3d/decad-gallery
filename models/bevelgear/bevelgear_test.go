package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestEightToothStraightBevelGear(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	doc := decad.New()
	body, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, body.IsSolid())
	require.Len(t, doc.Bodies(), 1)

	box, err := body.Bounds()
	require.NoError(t, err)
	width := math.Hypot(8, 8) / 6
	toeZ := width * math.Cos(math.Pi/4)
	require.InDelta(t, 0, box.Min.Z, box.Bound.Base()+1e-8)
	require.InDelta(t, toeZ, box.Max.Z, box.Bound.Base()+1e-8)
	require.Greater(t, box.Max.X, 4.5)
	require.Less(t, box.Max.X, 5.01)

	start, err := decad.Faces(decad.FaceCreatedBy(decad.CapStart(body))).Exactly(1).SelectFaces(body)
	require.NoError(t, err)
	end, err := decad.Faces(decad.FaceCreatedBy(decad.CapEnd(body))).Exactly(1).SelectFaces(body)
	require.NoError(t, err)
	startArea, err := start[0].Area()
	require.NoError(t, err)
	endArea, err := end[0].Area()
	require.NoError(t, err)
	require.Greater(t, startArea.Value.Base(), endArea.Value.Base()+startArea.Bound.Base()+endArea.Bound.Base())
	// The loft splits fitted flanks into many BRep edges. Check the actual
	// heel outline at each tooth and valley instead of counting those edges.
	heelEdges := start[0].Loops()[0].Edges()
	for tooth := range p.Teeth {
		center := 2 * math.Pi * float64(tooth) / float64(p.Teeth)
		valley := center + math.Pi/float64(p.Teeth)
		foundTip, foundRoot := false, false
		for _, edge := range heelEdges {
			v := edge.Start().Position().Value
			angle := math.Atan2(v.Y, v.X)
			radius := math.Hypot(v.X, v.Y)
			if angularDistance(angle, center) < math.Pi/(4*float64(p.Teeth)) && radius > 4.9 {
				foundTip = true
			}
			if angularDistance(angle, valley) < math.Pi/(4*float64(p.Teeth)) && radius < 2.85 {
				foundRoot = true
			}
		}
		require.True(t, foundTip, "tooth %d has no tip", tooth)
		require.True(t, foundRoot, "tooth %d has no root valley", tooth)
	}

	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1),
		decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func angularDistance(a, b float64) float64 {
	return math.Abs(math.Atan2(math.Sin(a-b), math.Cos(a-b)))
}
