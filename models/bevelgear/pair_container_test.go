package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/stretchr/testify/require"
)

// children returns the names of the child containers of c, in creation order.
func children(c *decad.Container) []string {
	var names []string
	for _, child := range c.Children() {
		names = append(names, child.Name())
	}
	return names
}

// containerPath names the container holding body.
func containerPath(t *testing.T, doc *decad.Document, body *decad.Body) string {
	t.Helper()
	c, err := doc.ContainerOf(body)
	require.NoError(t, err)
	if c.Parent() == nil {
		return "/"
	}
	return c.Parent().Name() + "/" + c.Name()
}

func volumeOf(t *testing.T, body *decad.Body) (float64, float64) {
	t.Helper()
	v, err := body.Volume()
	require.NoError(t, err)
	return v.Value.Base(), v.Bound.Base()
}

// The blank pair draws both blanks in one sketch. The volumes are what each blank had when it was revolved from
// a sketch of its own and the pinion was rotated about the apex. The pinion's section sits on the grid its
// shaft needs (shaft.go), which moves a volume by about 2e-8 of itself.
func TestBlankPairSharesOneSketch(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		mod                         func(p *bevelgear.Params)
		drivingVolume, pinionVolume float64
	}{
		{"equal", func(p *bevelgear.Params) {}, 3379.546545682, 3379.546545682},
		{"unequal", func(p *bevelgear.Params) { p.Teeth, p.MateTeeth, p.BaseHeight = 16, 12, 3 },
			458.776762366, 209.149699394},
		{"oblique", func(p *bevelgear.Params) {
			p.Teeth, p.MateTeeth, p.ShaftAngle = 16, 12, 70*math.Pi/180
		}, 335.847817576, 169.656265650},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := spiralDefaults()
			tc.mod(&p)
			doc := decad.New()
			pair, err := bevelgear.BuildBlankPair(t.Context(), doc, p)
			require.NoError(t, err)

			require.Equal(t, []string{"Pair"}, children(doc.Root()))
			pairContainer := doc.Root().Children()[0]
			require.Equal(t, []string{"Driving Gear", "Pinion Gear"}, children(pairContainer))
			require.Len(t, pairContainer.Sketches(), 1, "both sections are in one sketch")
			require.Equal(t, "Pair/Driving Gear", containerPath(t, doc, pair.Driving))
			require.Equal(t, "Pair/Pinion Gear", containerPath(t, doc, pair.Pinion))
			require.Empty(t, pairContainer.Bodies())
			require.NoError(t, doc.VerifyContainers().Err(), "no body in the root, no empty container")

			driving, drivingBound := volumeOf(t, pair.Driving)
			pinion, pinionBound := volumeOf(t, pair.Pinion)
			require.InDelta(t, tc.drivingVolume, driving, 1e-7*tc.drivingVolume+drivingBound)
			require.InDelta(t, tc.pinionVolume, pinion, 1e-7*tc.pinionVolume+pinionBound)
		})
	}
}

// A second pair in one document gets its own containers.
func TestSecondPairGetsOwnContainers(t *testing.T) {
	p := spiralDefaults()
	doc := decad.New()
	_, err := bevelgear.BuildBlankPair(t.Context(), doc, p)
	require.NoError(t, err)
	_, err = bevelgear.BuildBlankPair(t.Context(), doc, p)
	require.NoError(t, err)
	require.Equal(t, []string{"Pair", "Pair 2"}, children(doc.Root()))
}
