package main

import (
	"image/gif"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"github.com/stretchr/testify/require"
)

func TestPlanetaryMotionUsesBuiltGearbox(t *testing.T) {
	box, params, err := buildPlanetaryGearbox(t.Context(), 1)
	require.NoError(t, err)
	parts := box.Named()
	type geometry struct {
		min, max r3.Vec
		volume   float64
	}
	before := make([]geometry, len(parts))
	for i, part := range parts {
		bounds, err := part.Body.Bounds()
		require.NoError(t, err, part.Name)
		volume, err := part.Body.Volume()
		require.NoError(t, err, part.Name)
		before[i] = geometry{min: bounds.Min, max: bounds.Max, volume: volume.Value.Base()}
	}
	_, err = planetaryHeroScene(t.Context(), box)
	require.NoError(t, err)
	output := filepath.Join(t.TempDir(), "motion.gif")
	require.NoError(t, renderPlanetaryMotionFrom(t.Context(), output, 120, 80, 2, box, params))
	f, err := os.Open(output)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	animation, err := gif.DecodeAll(f)
	require.NoError(t, err)
	require.Len(t, animation.Image, 2)
	standalone := filepath.Join(t.TempDir(), "standalone.gif")
	require.NoError(t, renderPlanetaryMotion(t.Context(), standalone, 120, 80, 2))
	g, err := os.Open(standalone)
	require.NoError(t, err)
	defer func() { _ = g.Close() }()
	twoStage, err := gif.DecodeAll(g)
	require.NoError(t, err)
	different := false
	for y := range 80 {
		for x := range 120 {
			r1, g1, b1, a1 := animation.Image[0].At(x, y).RGBA()
			r2, g2, b2, a2 := twoStage.Image[0].At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				different = true
				break
			}
		}
		if different {
			break
		}
	}
	require.True(t, different, "one-stage gearbox must produce a different first frame from the standalone two-stage render")
	for i, part := range parts {
		bounds, err := part.Body.Bounds()
		require.NoError(t, err, part.Name)
		volume, err := part.Body.Volume()
		require.NoError(t, err, part.Name)
		require.Equal(t, before[i].min, bounds.Min, "%s min", part.Name)
		require.Equal(t, before[i].max, bounds.Max, "%s max", part.Name)
		require.Equal(t, before[i].volume, volume.Value.Base(), "%s volume", part.Name)
	}
}

// TestPlanetaryGearboxMotion builds the real two-stage gearbox once and checks the animation's kinematics:
// at an input angle that turns the first carrier a third of a turn, so planet 0 has been carried
// to where planet 1 started; the fixed ring and the covers have not moved; the input shaft has turned by the
// input angle.
func TestPlanetaryGearboxMotion(t *testing.T) {
	box, params, err := buildPlanetaryGearbox(t.Context(), 2)
	require.NoError(t, err)
	movers, err := planetaryMovers(t.Context(), box, params)
	require.NoError(t, err)
	uncutA, err := planetaryPanelMovers(movers, false)
	require.NoError(t, err)
	uncutB, err := planetaryPanelMovers(movers, false)
	require.NoError(t, err)
	cut, err := planetaryPanelMovers(movers, true)
	require.NoError(t, err)
	restModels, err := planetaryPoseModels(uncutA, 0, false)
	require.NoError(t, err)
	later, err := planetaryPoseModels(uncutA, 0.2, false)
	require.NoError(t, err)
	housingCount := 0
	for i, mover := range movers {
		if cut[i].mesh != mover.mesh {
			t.Fatalf("cut panel changed mesh for %s", mover.body.name)
		}
		if !mover.body.housing || mover.move != nil {
			if uncutA[i].mesh != mover.mesh || uncutB[i].mesh != mover.mesh {
				t.Fatalf("uncut panel changed non-housing mesh for %s", mover.body.name)
			}
			continue
		}
		housingCount++
		meshA, ok := uncutA[i].mesh.(*solidlens.Mesh)
		require.True(t, ok, "%s uses a reusable mesh", mover.body.name)
		meshB, ok := uncutB[i].mesh.(*solidlens.Mesh)
		require.True(t, ok, "%s uses a reusable mesh", mover.body.name)
		if meshA == meshB || restModels[i].Mesh != meshA || later[i].Mesh != meshA {
			t.Fatalf("panel mesh reuse differs for %s", mover.body.name)
		}
		vertices, copyVertices := mover.mesh.Vertices(), meshA.Vertices()
		triangles, copyTriangles := mover.mesh.Triangles(), meshA.Triangles()
		require.Equal(t, len(vertices), len(copyVertices), "%s vertex count", mover.body.name)
		require.Equal(t, len(triangles), len(copyTriangles), "%s triangle count", mover.body.name)
		for j := range vertices {
			if vertices[j] != copyVertices[j] {
				t.Fatalf("%s vertex %d changed", mover.body.name, j)
			}
		}
		for j := range triangles {
			if triangles[j] != copyTriangles[j] {
				t.Fatalf("%s triangle %d changed", mover.body.name, j)
			}
		}
	}
	require.Equal(t, 3, housingCount)
	input := 2 * math.Pi / float64(params.PlanetCount) * float64(params.SunTeeth+params.RingTeeth) / float64(params.SunTeeth)
	posed, err := planetaryPoseModels(movers, input, true)
	require.NoError(t, err)
	require.Len(t, posed, len(movers))

	centroid := func(source solidlens.TriangleSource) r3.Vec {
		var sum r3.Vec
		vertices := source.Vertices()
		for _, v := range vertices {
			sum = sum.Add(v)
		}
		return sum.Scale(1 / float64(len(vertices)))
	}
	byName := make(map[string]int)
	for i, mover := range movers {
		byName[mover.body.name] = i
	}
	rest0 := centroid(movers[byName[planetarygearbox.PlanetName(0, 0)]].mesh)
	rest1 := centroid(movers[byName[planetarygearbox.PlanetName(0, 1)]].mesh)
	moved0 := centroid(posed[byName[planetarygearbox.PlanetName(0, 0)]].Mesh)
	require.InDelta(t, 0, moved0.Sub(rest1).Len(), 0.25, "planet 0 is carried to planet 1's place (vertex centroids, so a quarter millimeter of slack)")
	require.Greater(t, rest0.Sub(rest1).Len(), 10.0, "the planets start apart")

	for _, name := range []string{planetarygearbox.RingName, planetarygearbox.FrontCoverName, planetarygearbox.RearCoverName} {
		before, after := centroid(movers[byName[name]].mesh), centroid(posed[byName[name]].Mesh)
		require.InDelta(t, 0, after.Sub(before).Len(), 1e-9, "%s is fixed", name)
	}

	// The input shaft turns by the input angle itself, not by a ratio of it.
	shaft := byName[planetarygearbox.InputShaftName]
	rest, turned := movers[shaft].mesh.Vertices(), posed[shaft].Mesh.Vertices()
	sin, cos := math.Sincos(input)
	for _, i := range []int{0, len(rest) / 2, len(rest) - 1} {
		require.InDelta(t, rest[i].X*cos-rest[i].Y*sin, turned[i].X, 1e-9)
		require.InDelta(t, rest[i].X*sin+rest[i].Y*cos, turned[i].Y, 1e-9)
		require.InDelta(t, rest[i].Z, turned[i].Z, 1e-9)
	}

	// The last carrier, which carries the output shaft, turns by the input angle times the stage ratio squared.
	last := byName[planetarygearbox.CarrierName(1)]
	restCarrier, turnedCarrier := movers[last].mesh.Vertices(), posed[last].Mesh.Vertices()
	q := float64(params.SunTeeth) / float64(params.SunTeeth+params.RingTeeth)
	sin, cos = math.Sincos(input * q * q)
	for _, i := range []int{0, len(restCarrier) / 2, len(restCarrier) - 1} {
		require.InDelta(t, restCarrier[i].X*cos-restCarrier[i].Y*sin, turnedCarrier[i].X, 1e-9)
		require.InDelta(t, restCarrier[i].X*sin+restCarrier[i].Y*cos, turnedCarrier[i].Y, 1e-9)
	}

	// The output shaft is a separate body that turns with the last carrier.
	shaftIndex := byName[planetarygearbox.OutputShaftName]
	restShaft, turnedShaft := movers[shaftIndex].mesh.Vertices(), posed[shaftIndex].Mesh.Vertices()
	for _, i := range []int{0, len(restShaft) / 2, len(restShaft) - 1} {
		require.InDelta(t, restShaft[i].X*cos-restShaft[i].Y*sin, turnedShaft[i].X, 1e-9)
		require.InDelta(t, restShaft[i].X*sin+restShaft[i].Y*cos, turnedShaft[i].Y, 1e-9)
	}
}
