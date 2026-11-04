package meshgap_test

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestRealBearingSupportPath(t *testing.T) {
	p := ballbearing.DefaultParams()
	meshes := map[string]meshgap.Mesh{
		"ground": bearingMesh(t, p, r3.Vec{}),
		"part":   bearingMesh(t, p, r3.Vec{Z: 7}),
	}
	reached, _, err := meshgap.Reachable([]meshgap.Mount{
		{Part: "ground", Ground: true},
		{Part: "part", References: []string{"ground"}},
	}, meshes)
	require.NoError(t, err)
	require.Equal(t, 2, len(reached))
	_, supported := reached["part"]
	require.True(t, supported)
}

func TestFloatingCycleHasNoSupportPath(t *testing.T) {
	p := ballbearing.DefaultParams()
	meshes := map[string]meshgap.Mesh{
		"ground": bearingMesh(t, p, r3.Vec{}),
		"a":      bearingMesh(t, p, r3.Vec{Z: 40}),
		"b":      bearingMesh(t, p, r3.Vec{Z: 47}),
	}
	reached, nearest, err := meshgap.Reachable([]meshgap.Mount{
		{Part: "ground", Ground: true},
		{Part: "a", References: []string{"b", "ground"}},
		{Part: "b", References: []string{"a"}},
	}, meshes)
	require.NoError(t, err)
	require.Equal(t, 1, len(reached), "a distant ground reference cannot support a touching floating group")
	_, supported := reached["a"]
	require.False(t, supported)
	_, supported = reached["b"]
	require.False(t, supported)
	require.Equal(t, "b", nearest["a"].Part)
	require.Zero(t, nearest["a"].LowerGap, "local contact alone does not establish a path to ground")
}

func TestPartInsideEmptyBoreHasNoSupportPath(t *testing.T) {
	meshes := map[string]meshgap.Mesh{
		"ground": bearingMesh(t, ballbearing.DefaultParams(), r3.Vec{}),
		"part":   bearingMesh(t, ballbearing.Params{BoreDiameter: 2, OuterDiameter: 4, Width: 1}, r3.Vec{Z: 3}),
	}
	reached, nearest, err := meshgap.Reachable([]meshgap.Mount{
		{Part: "ground", Ground: true},
		{Part: "part", References: []string{"ground"}, Clearance: .1},
	}, meshes)
	require.NoError(t, err)
	require.Equal(t, 1, len(reached), "overlapping bounding boxes must not hide a floating part")
	require.Greater(t, nearest["part"].LowerGap, 1.8)
}

func TestContactBridgeProvidesSupportPath(t *testing.T) {
	p := ballbearing.DefaultParams()
	bridge := p
	bridge.Width = 1
	meshes := map[string]meshgap.Mesh{
		"ground": bearingMesh(t, p, r3.Vec{}),
		"part":   bearingMesh(t, p, r3.Vec{Z: 8}),
		"bridge": bearingMesh(t, bridge, r3.Vec{Z: 7}),
	}
	reached, nearest, err := meshgap.Reachable([]meshgap.Mount{
		{Part: "part", References: []string{"ground", "bridge"}},
		{Part: "bridge", References: []string{"part", "ground"}},
		{Part: "ground", Ground: true},
	}, meshes)
	require.NoError(t, err)
	require.Equal(t, 3, len(reached), "a real intervening part can carry a support path across the direct gap")
	require.Equal(t, "bridge", nearest["part"].Part)
}

func TestDesignedEyeClearanceIsCharged(t *testing.T) {
	pin := ballbearing.Params{BoreDiameter: 1, OuterDiameter: 3, Width: 6}
	eye := ballbearing.Params{BoreDiameter: 3.4, OuterDiameter: 5, Width: .8}
	meshes := map[string]meshgap.Mesh{
		"pin": bearingMesh(t, pin, r3.Vec{}),
		"eye": bearingMesh(t, eye, r3.Vec{Z: 2.6}),
	}
	mounts := []meshgap.Mount{
		{Part: "pin", Ground: true},
		{Part: "eye", References: []string{"pin"}},
	}
	reached, _, err := meshgap.Reachable(mounts, meshes)
	require.NoError(t, err)
	require.Equal(t, 1, len(reached), "the eye and pin surfaces do not touch")
	mounts[1].Clearance = (eye.BoreDiameter - pin.OuterDiameter) / 2
	reached, _, err = meshgap.Reachable(mounts, meshes)
	require.NoError(t, err)
	require.Equal(t, 2, len(reached), "the specified radial free play permits this geometric connection")
}

func TestMountGraphRejectsMissingGeometryAndReferences(t *testing.T) {
	a := bearingMesh(t, ballbearing.DefaultParams(), r3.Vec{})
	_, _, err := meshgap.Reachable([]meshgap.Mount{{Part: "a", Ground: true}}, nil)
	require.ErrorContains(t, err, "missing mesh for a")
	meshes := map[string]meshgap.Mesh{"a": a}
	_, _, err = meshgap.Reachable([]meshgap.Mount{{Part: "a", Ground: true}, {Part: "a"}}, meshes)
	require.ErrorContains(t, err, "duplicate mount a")
	_, _, err = meshgap.Reachable([]meshgap.Mount{{Part: "a", References: []string{"missing"}}}, meshes)
	require.ErrorContains(t, err, "absent mount missing")
	_, _, err = meshgap.Reachable([]meshgap.Mount{{Part: "a", References: []string{"a"}}}, meshes)
	require.ErrorContains(t, err, "cannot support itself")
}
