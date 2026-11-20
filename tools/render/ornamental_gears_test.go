package main

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestOrnamentalFaceplateWindows(t *testing.T) {
	for _, wheel := range []ornamentalWheel{{teeth: 30, petals: 7, slant: .48}, {teeth: 22, petals: 5, slant: -.46}} {
		r := float64(wheel.teeth)
		plate, err := wheel.faceplate(t.Context(), decad.New(), sketch.NewWorld(), r, 0, 1)
		require.NoError(t, err)
		for _, ref := range []decad.FeatureRef{decad.CapStart(plate), decad.CapEnd(plate)} {
			caps, err := decad.Faces(decad.FaceCreatedBy(ref)).Exactly(1).SelectFaces(plate)
			require.NoError(t, err)
			loops := caps[0].Loops()
			require.Len(t, loops, 2*wheel.petals+2)
			require.Len(t, loops[0].Edges(), wheel.petals)
			var bores, large, small int
			for _, loop := range loops[1:] {
				edges := loop.Edges()
				if len(edges) == 1 {
					bore, ok := edges[0].Curve().(decad.Circle3)
					require.True(t, ok)
					require.InDelta(t, .09*r, bore.Radius.Base(), 1e-9)
					bores++
					continue
				}
				require.Len(t, edges, 2)
				for _, edge := range edges {
					require.IsType(t, decad.Arc3{}, edge.Curve())
				}
				start, end := edges[0].Start().Position().Value, edges[0].End().Position().Value
				middle := start.Add(end).Scale(.5)
				radial := math.Hypot(middle.X, middle.Y)
				halfChord := start.Sub(end).Len() / 2
				if radial > .45*r {
					require.InDelta(t, .53*r, radial, 1e-8)
					require.InDelta(t, .199*r, halfChord, 1e-8)
					large++
				} else {
					require.InDelta(t, .354*r, radial, 1e-8)
					require.InDelta(t, .105*r, halfChord, 1e-8)
					small++
				}
			}
			require.Equal(t, 1, bores)
			require.Equal(t, wheel.petals, large)
			require.Equal(t, wheel.petals, small)
		}
	}
}
