package cubepuzzle_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/cubepuzzle"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestMountedAssemblyClearances(t *testing.T) {
	doc := cubepuzzle.NewDocument()
	a, err := cubepuzzle.Build(t.Context(), doc, cubepuzzle.DefaultParams())
	require.NoError(t, err)
	hub := checkMesh(t, t.Context(), "hub", a.Hub, .2)
	blocks := append(checkParts(t, "center", a.Centers, .025), checkParts(t, "edge", a.Edges, .025)...)
	blocks = append(blocks, checkParts(t, "corner", a.Corners, .025)...)
	hardware := append(checkParts(t, "disk", a.Disks, .05), checkParts(t, "flanged sleeve", a.FlangedSleeves, .025)...)
	hardware = append(hardware, checkParts(t, "sleeve", a.Sleeves, .025)...)
	hardware = append(hardware, checkParts(t, "bolt", a.Bolts, .05)...)
	hardware = append(hardware, checkParts(t, "nut", a.Nuts, .025)...)
	for i, part := range blocks {
		requireGap(t, part, hub, .35)
		for _, other := range blocks[i+1:] {
			requireGap(t, part, other, .35)
		}
		for _, fixed := range hardware {
			requireGap(t, part, fixed, .35)
		}
	}
	for _, part := range append(checkParts(t, "bolt", a.Bolts, .025), checkParts(t, "nut", a.Nuts, .025)...) {
		requireGap(t, hub, part, .05)
	}
}

func TestPositiveZLayerSampledTurns(t *testing.T) {
	doc := cubepuzzle.NewDocument()
	a, err := cubepuzzle.Build(t.Context(), doc, cubepuzzle.DefaultParams())
	require.NoError(t, err)
	hub := checkMesh(t, t.Context(), "hub", a.Hub, .2)
	blocks := append(checkParts(t, "center", a.Centers, .025), checkParts(t, "edge", a.Edges, .025)...)
	blocks = append(blocks, checkParts(t, "corner", a.Corners, .025)...)
	hardware := append(checkParts(t, "disk", a.Disks, .05), checkParts(t, "flanged sleeve", a.FlangedSleeves, .025)...)
	hardware = append(hardware, checkParts(t, "sleeve", a.Sleeves, .025)...)
	hardware = append(hardware, checkParts(t, "bolt", a.Bolts, .05)...)
	hardware = append(hardware, checkParts(t, "nut", a.Nuts, .025)...)
	var turning, stationary []checkedPart
	for _, p := range blocks {
		maxZ := math.Inf(-1)
		for _, v := range p.vertices {
			maxZ = max(maxZ, v.Z)
		}
		if maxZ > 29.9 {
			turning = append(turning, p)
		} else {
			stationary = append(stationary, p)
		}
	}
	require.Equal(t, 9, len(turning))
	for _, degrees := range []float64{15, 30, 45, 60, 75, 90} {
		cos, sin := math.Cos(degrees*math.Pi/180), math.Sin(degrees*math.Pi/180)
		for _, part := range turning {
			vertices := make([]r3.Vec, len(part.vertices))
			for i, v := range part.vertices {
				vertices[i] = r3.Vec{X: cos*v.X - sin*v.Y, Y: sin*v.X + cos*v.Y, Z: v.Z}
			}
			mesh, err := meshgap.New(vertices, part.triangles, part.bound)
			require.NoError(t, err)
			moved := checkedPart{name: part.name, mesh: mesh}
			requireGap(t, moved, hub, .35)
			for _, fixed := range stationary {
				requireGap(t, moved, fixed, .35)
			}
			for _, fixed := range hardware {
				requireGap(t, moved, fixed, .35)
			}
		}
	}
}
