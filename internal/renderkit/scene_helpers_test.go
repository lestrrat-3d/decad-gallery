package renderkit_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
)

// cubeSource is the eight corners of a unit cube centred at (1, 2, 0).
type cubeSource struct{}

func (cubeSource) Vertices() []r3.Vec {
	var out []r3.Vec
	for _, x := range []float64{0.5, 1.5} {
		for _, y := range []float64{1.5, 2.5} {
			for _, z := range []float64{-0.5, 0.5} {
				out = append(out, r3.Vec{X: x, Y: y, Z: z})
			}
		}
	}
	return out
}

func (cubeSource) Triangles() [][3]int { return nil }

// offCentreCubeModels returns one unit cube whose centre is away from the origin.
func offCentreCubeModels(t *testing.T) []solidlens.Model {
	t.Helper()
	return []solidlens.Model{{Mesh: cubeSource{}}}
}

// lookingDownNegativeZ views the cube from +Z with a target away from the cube.
func lookingDownNegativeZ() solidlens.Camera {
	return solidlens.Camera{
		Position: solidlens.Vec{X: 3, Y: -2, Z: 20},
		Target:   solidlens.Vec{X: 3, Y: -2, Z: 0},
		Up:       solidlens.Vec{Y: 1},
		FOV:      40,
	}
}

// projectedExtents returns the extents of the models' vertices in the camera's image plane.
func projectedExtents(camera solidlens.Camera, models []solidlens.Model) (minX, maxX, minY, maxY float64) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, model := range models {
		for _, v := range model.Mesh.Vertices() {
			depth := camera.Position.Z - v.Z
			x, y := (v.X-camera.Position.X)/depth, (v.Y-camera.Position.Y)/depth
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			minY, maxY = math.Min(minY, y), math.Max(maxY, y)
		}
	}
	return minX, maxX, minY, maxY
}
