package main

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestOrnamentalFaceplateWindows(t *testing.T) {
	wheel := ornamentalWheel{teeth: 60, ornament: rosette}
	r := float64(wheel.teeth)
	plate, err := wheel.faceplate(t.Context(), decad.New(), sketch.NewWorld(), r, 0, 1)
	require.NoError(t, err)
	for _, ref := range []decad.FeatureRef{decad.CapStart(plate), decad.CapEnd(plate)} {
		caps, err := decad.Faces(decad.FaceCreatedBy(ref)).Exactly(1).SelectFaces(plate)
		require.NoError(t, err)
		loops := caps[0].Loops()
		require.Equal(t, 2*rosettePetals+2, len(loops))
		require.Equal(t, rosettePetals, len(loops[0].Edges()))
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
			require.Equal(t, 2, len(edges))
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
		require.Equal(t, rosettePetals, large)
		require.Equal(t, rosettePetals, small)
	}
}

func TestCompassVaneAndHoneycombWindows(t *testing.T) {
	for _, tc := range []struct {
		name                                    string
		ornament                                wheelOrnament
		teeth, windows, outerEdges, windowEdges int
	}{
		{"compass", compass, 36, 8, 16, 3},
		{"vanes", vanes, 22, 6, 1, 4},
		{"honeycomb", honeycomb, 17, 18, 6, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := ornamentalWheel{teeth: tc.teeth, ornament: tc.ornament}
			r := float64(w.teeth)
			body, err := w.faceplate(t.Context(), decad.New(), sketch.NewWorld(), r, 0, 1)
			require.NoError(t, err)
			mesh, err := body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.VolumeVerified())
			require.True(t, mesh.BoundaryVerified())
			for _, cap := range []decad.FeatureRef{decad.CapStart(body), decad.CapEnd(body)} {
				faces, err := decad.Faces(decad.FaceCreatedBy(cap)).Exactly(1).SelectFaces(body)
				require.NoError(t, err)
				loops := faces[0].Loops()
				require.Equal(t, tc.windows+2, len(loops))
				require.Equal(t, tc.outerEdges, len(loops[0].Edges()))
				var count int
				for _, loop := range loops[1:] {
					edges := loop.Edges()
					if len(edges) == 1 {
						circle, ok := edges[0].Curve().(decad.Circle3)
						require.True(t, ok)
						require.InDelta(t, .09*r, circle.Radius.Base(), 1e-8)
						continue
					}
					count++
					require.Equal(t, tc.windowEdges, len(edges))
					for _, edge := range edges {
						start, end := edge.Start().Position().Value, edge.End().Position().Value
						switch tc.ornament {
						case vanes:
							require.IsType(t, decad.Arc3{}, edge.Curve())
							radius := math.Hypot(start.X, start.Y) / r
							require.True(t, math.Abs(radius-.257) < 1e-8 || math.Abs(radius-.695) < 1e-8)
						case honeycomb:
							require.IsType(t, decad.Line3{}, edge.Curve())
							require.InDelta(t, .136*r, start.Sub(end).Len(), 1e-8)
						case compass:
							require.IsType(t, decad.Line3{}, edge.Curve())
						}
					}
				}
				require.Equal(t, tc.windows, count)
			}
		})
	}
}

func TestBorderOrnamentWindows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ornament wheelOrnament
		windows  int
	}{
		{"perforated", perforated, 8},
		{"diamonds", diamonds, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wheel := ornamentalWheel{teeth: 17, ornament: tc.ornament}
			plate, err := wheel.faceplate(t.Context(), decad.New(), sketch.NewWorld(), 17, 0, 1)
			require.NoError(t, err)
			for _, cap := range []decad.FeatureRef{decad.CapStart(plate), decad.CapEnd(plate)} {
				faces, err := decad.Faces(decad.FaceCreatedBy(cap)).Exactly(1).SelectFaces(plate)
				require.NoError(t, err)
				require.Len(t, faces[0].Loops(), tc.windows+2)
			}
			mesh, err := plate.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
		})
	}
}
