package renderkit

import (
	"math"

	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
)

// Scene returns a scene of models seen from camera with the gallery's two directional lights and pale background.
func Scene(camera solidlens.Camera, models []solidlens.Model) solidlens.Scene {
	return solidlens.Scene{
		Camera: camera,
		Models: models,
		DirectionalLights: []solidlens.DirectionalLight{
			{Direction: solidlens.Vec{X: 0.5, Y: 0.4, Z: -1}, Color: solidlens.RGB(1, 1, 1), Intensity: 1.1},
			{Direction: solidlens.Vec{X: -0.8, Y: -0.2, Z: -0.4}, Color: solidlens.RGB(0.2, 0.5, 1), Intensity: 0.6},
		},
		Background: solidlens.RGB(0.88, 0.92, 0.95),
	}
}

// FitCamera slides the camera parallel to the image plane until the models' projected extents are centered,
// then narrows or widens its field of view until the extents leave margin of the image free on every side.
// The camera keeps its viewing direction and distance.
func FitCamera(camera solidlens.Camera, models []solidlens.Model, aspect, margin float64) solidlens.Camera {
	position, target := r3.Vec(camera.Position), r3.Vec(camera.Target)
	forward, ok := target.Sub(position).Normalize()
	if !ok {
		return camera
	}
	right, ok := forward.Cross(r3.Vec(camera.Up)).Normalize()
	if !ok {
		return camera
	}
	up := right.Cross(forward)
	var vertices []r3.Vec
	for _, model := range models {
		vertices = append(vertices, model.Mesh.Vertices()...)
	}
	var halfX, halfY float64
	for range 20 {
		minX, maxX := math.Inf(1), math.Inf(-1)
		minY, maxY := math.Inf(1), math.Inf(-1)
		var depthSum float64
		for _, v := range vertices {
			d := v.Sub(position)
			depth := d.Dot(forward)
			x, y := d.Dot(right)/depth, d.Dot(up)/depth
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			minY, maxY = math.Min(minY, y), math.Max(maxY, y)
			depthSum += depth
		}
		meanDepth := depthSum / float64(len(vertices))
		shiftX := (minX + maxX) / 2 * meanDepth
		shiftY := (minY + maxY) / 2 * meanDepth
		shift := right.Scale(shiftX).Add(up.Scale(shiftY))
		position, target = position.Add(shift), target.Add(shift)
		halfX, halfY = (maxX-minX)/2, (maxY-minY)/2
	}
	// The vertical field of view FOV satisfies tan(FOV/2) = half the image height in image-plane units.
	tanHalf := math.Max(halfX/aspect, halfY) / (1 - 2*margin)
	camera.Position, camera.Target = solidlens.Vec(position), solidlens.Vec(target)
	camera.FOV = 2 * math.Atan(tanHalf) * 180 / math.Pi
	return camera
}
