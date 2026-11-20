package main

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/swisslever"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/stretchr/testify/require"
)

// swissBuildOK builds the real movement.
func swissBuildOK(t *testing.T) *swissBuild {
	t.Helper()
	b, err := buildSwiss(t.Context())
	require.NoError(t, err)
	return b
}

// swissBodyNames lists the body names of a movement without building one.
func swissBodyNames() []string {
	m := &swisslever.Movement{Bolts: make([]*decad.Body, 4), Nuts: make([]*decad.Body, 4)}
	var names []string
	for _, nb := range m.Named() {
		names = append(names, nb.Name)
	}
	return names
}

// projectedSpan returns the share of the image width and height the models' vertices span as seen by camera.
func projectedSpan(camera solidlens.Camera, aspect float64, models []solidlens.Model) (float64, float64) {
	position, target := r3.Vec(camera.Position), r3.Vec(camera.Target)
	forward, _ := target.Sub(position).Normalize()
	right, _ := forward.Cross(r3.Vec(camera.Up)).Normalize()
	up := right.Cross(forward)
	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, m := range models {
		for _, v := range m.Mesh.Vertices() {
			d := v.Sub(position)
			depth := d.Dot(forward)
			x, y := d.Dot(right)/depth, d.Dot(up)/depth
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			minY, maxY = math.Min(minY, y), math.Max(maxY, y)
		}
	}
	tanHalf := math.Tan(camera.FOV * math.Pi / 360)
	return (maxX - minX) / (2 * tanHalf * aspect), (maxY - minY) / (2 * tanHalf)
}
