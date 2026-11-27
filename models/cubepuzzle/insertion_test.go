package cubepuzzle_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/cubepuzzle"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestSampledInsertionClearances(t *testing.T) {
	doc := cubepuzzle.NewDocument()
	a, err := cubepuzzle.Build(t.Context(), doc, cubepuzzle.DefaultParams())
	require.NoError(t, err)
	hub := checkMesh(t, t.Context(), "hub", a.Hub, .2)
	centers := checkParts(t, "center", a.Centers, .025)
	edges := checkParts(t, "edge", a.Edges, .025)
	corners := checkParts(t, "corner", a.Corners, .025)
	disks := checkParts(t, "disk", a.Disks, .05)
	flanges := checkParts(t, "flanged sleeve", a.FlangedSleeves, .025)
	sleeves := checkParts(t, "sleeve", a.Sleeves, .025)
	bolts := checkParts(t, "bolt", a.Bolts, .05)
	nuts := checkParts(t, "nut", a.Nuts, .025)
	offsets := []float64{1, 2, 3, 4, 5, 8, 12, 20, 35}
	move := func(p checkedPart, v r3.Vec) checkedPart {
		vertices := make([]r3.Vec, len(p.vertices))
		for i, point := range p.vertices {
			vertices[i] = point.Add(v)
		}
		mesh, err := meshgap.New(vertices, p.triangles, p.bound)
		require.NoError(t, err, "%s moved mesh", p.name)
		return checkedPart{name: p.name, mesh: mesh}
	}
	pairs := [][2]r3.Vec{{{X: 1}, {Y: 1}}, {{X: 1}, {Z: 1}}, {{Y: 1}, {Z: 1}}}
	for i, edge := range edges {
		x, y := -1.0, -1.0
		if i%4 >= 2 {
			x = 1
		}
		if i%2 == 1 {
			y = 1
		}
		direction := pairs[i/4][0].Scale(x).Add(pairs[i/4][1].Scale(y))
		for _, offset := range offsets {
			part := move(edge, direction.Scale(offset))
			requireGap(t, part, hub, .35)
			for j, other := range edges {
				if i != j {
					requireGap(t, part, other, .35)
				}
			}
			for _, other := range corners {
				requireGap(t, part, other, .35)
			}
		}
	}
	axes := []r3.Vec{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}, {Z: 1}, {Z: -1}}
	boltIndices := []int{1, 2, 3, 4, 0, 5}
	for i, center := range centers {
		for _, offset := range offsets {
			part := move(center, axes[i].Scale(offset))
			requireGap(t, part, hub, .35)
			for j, other := range centers {
				if i != j {
					requireGap(t, part, other, .35)
				}
			}
			for _, group := range [][]checkedPart{edges, corners, flanges, nuts} {
				for _, other := range group {
					requireGap(t, part, other, .35)
				}
			}
			for j := range axes {
				if i != j {
					requireGap(t, part, disks[j], .35)
					requireGap(t, part, sleeves[j], .35)
					requireGap(t, part, bolts[boltIndices[j]], .35)
				}
			}
		}
	}
}
