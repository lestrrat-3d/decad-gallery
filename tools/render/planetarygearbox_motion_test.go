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

// TestPlanetaryGearboxMotion builds the real two-stage gearbox once and checks the animation's kinematics:
// at an input angle that turns the first carrier a third of a turn, so planet 0 has been carried
// to where planet 1 started; the fixed ring and the covers have not moved; the input shaft has turned by the
// input angle. It then renders three small frames into a looping GIF.
func TestPlanetaryGearboxMotion(t *testing.T) {
	box, params, err := buildPlanetaryGearbox(t.Context(), 2)
	require.NoError(t, err)
	movers, err := planetaryMovers(t.Context(), box, params)
	require.NoError(t, err)
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

	output := filepath.Join(t.TempDir(), "motion.gif")
	require.NoError(t, renderPlanetaryMotion(t.Context(), output, 480, 160, 3))
	file, err := os.Open(output)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	animation, err := gif.DecodeAll(file)
	require.NoError(t, err)
	// Three input angles play forward and then back without repeating the ends: 0, 1, 2, 1.
	require.Len(t, animation.Image, 4)
	require.Equal(t, 480, animation.Image[0].Bounds().Dx())
	require.Equal(t, 160, animation.Image[0].Bounds().Dy())
	require.Zero(t, animation.LoopCount, "the animation loops forever")
	require.NotEqual(t, animation.Image[0].Pix, animation.Image[2].Pix, "the frames differ")
}
