package clamparm_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/fastenertool"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func interfaceFrame(t *testing.T, z float64) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	return f
}

func meshOf(t *testing.T, body *decad.Body) meshgap.Mesh {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.004))
	require.NoError(t, err)
	m, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return m
}

func TestClampArmInterfaceConstants(t *testing.T) {
	require.InDelta(t, 6.0, clamparm.BossBaseRadius, 0)
	require.InDelta(t, 2.0, clamparm.BossHeight, 0)
	require.InDelta(t, 0.35, clamparm.RecessClearance, 0)
	require.InDelta(t, 6.495, clamparm.RecessRadius, 0)
	require.InDelta(t, clamparm.BossBaseRadius+clamparm.RecessClearance*math.Sqrt2, clamparm.RecessRadius, 0.001)
	require.InDelta(t, 13.5, clamparm.StudProtrusion, 0)
	require.InDelta(t, 4.7, clamparm.AdapterNutMin, 0)
	require.InDelta(t, 8.3, clamparm.AdapterNutMax, 0)
	require.InDelta(t, 14.5, clamparm.AdapterStudClear, 0)
}

func TestClampArmInterfaceRecessToolSketch(t *testing.T) {
	doc := clamparm.NewDocument()
	body, s, err := clamparm.InterfaceRecessTool(t.Context(), doc, interfaceFrame(t, 0))
	require.NoError(t, err)
	require.True(t, body.IsSolid())
	requireFullyConstrained(t, []*sketch.Sketch{s})
	requireBounds(t, body, r3.NewVec(-7.495, -7.495, -1), r3.NewVec(7.495, 7.495, 16))
}

func TestClampArmInterfaceAdapterBuild(t *testing.T) {
	doc := clamparm.NewDocument()
	body, sketches, err := clamparm.BuildAdapter(t.Context(), doc, clamparm.DefaultParams())
	require.NoError(t, err)
	require.True(t, body.IsSolid())
	require.Len(t, sketches, 3)
	requireFullyConstrained(t, sketches)
	requireBounds(t, body, r3.NewVec(-20, -20, 0), r3.NewVec(20, 20, 15))
}

// TestClampArmInterfaceBossClearance checks the cone boss of the head (steps.md Section 9, in the adapter's frame)
// against the adapter's recess.
func TestClampArmInterfaceBossClearance(t *testing.T) {
	doc := clamparm.NewDocument()
	adapter, _, err := clamparm.BuildAdapter(t.Context(), doc, clamparm.DefaultParams())
	require.NoError(t, err)
	bf, err := r3.NewFrame(r3.NewVec(0, 0, -1), r3.NewVec(0, 0, 1), r3.NewVec(1, 0, 0))
	require.NoError(t, err)
	boss, _, err := fastenertool.RoundedRevolve(t.Context(), doc, bf, [][2]float64{{0, 0}, {0, 6}, {1, 6}, {3, 4}, {3, 0}})
	require.NoError(t, err)
	// The mesh gap is a lower bound that charges both meshes' tessellation bounds (about 0.008 mm), so it reads 0.342
	// for the true 0.35 normal clearance; the check allows 0.01.
	gap, separated, err := meshOf(t, boss).Exceeds(meshOf(t, adapter), clamparm.RecessClearance-0.01)
	require.NoError(t, err)
	require.True(t, separated, "gap lower bound %.4f", gap)
	require.GreaterOrEqual(t, gap, clamparm.RecessClearance-0.01)
}

// TestClampArmInterfaceFloorOverRecess probes the adapter's material between the recess cone and the nut pocket's
// floor on a thin column just outside the stud hole, where the floor is thinnest.
func TestClampArmInterfaceFloorOverRecess(t *testing.T) {
	doc := clamparm.NewDocument()
	adapter, _, err := clamparm.BuildAdapter(t.Context(), doc, clamparm.DefaultParams())
	require.NoError(t, err)
	f, err := r3.NewFrame(r3.NewVec(2.83, 0, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	column, _, err := fastenertool.Cylinder(t.Context(), doc, f, 0.02, 15)
	require.NoError(t, err)
	probe, err := decad.Intersect(t.Context(), adapter, column)
	require.NoError(t, err)
	box, err := probe.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 4.7, box.Max.Z, 0.001)
	// The column spans 0.04 mm of radius, so its bounds overstate the thinnest point by at most 0.04.
	require.GreaterOrEqual(t, box.Max.Z-box.Min.Z-0.04, 1.0)
}

// TestClampArmInterfaceStudTip seats the nut in the adapter's pocket and checks the stud's tip against it and the top
// face.
func TestClampArmInterfaceStudTip(t *testing.T) {
	doc := clamparm.NewDocument()
	adapter, _, err := clamparm.BuildAdapter(t.Context(), doc, clamparm.DefaultParams())
	require.NoError(t, err)
	nut, err := m5fastener.PlaceNut(t.Context(), doc, interfaceFrame(t, 4.7))
	require.NoError(t, err)
	nutBox, err := nut.Body.Bounds()
	require.NoError(t, err)
	adapterBox, err := adapter.Bounds()
	require.NoError(t, err)

	// The nut's near face is within the allowed band over the mating face.
	require.GreaterOrEqual(t, nutBox.Min.Z, clamparm.AdapterNutMin-0.001)
	require.LessOrEqual(t, nutBox.Min.Z, clamparm.AdapterNutMax)
	// The stud tip is past the nut's far face and inside the top face.
	require.Greater(t, clamparm.StudProtrusion, nutBox.Max.Z)
	require.Less(t, clamparm.StudProtrusion, adapterBox.Max.Z)
	// The hole clears the stud's length and the nut stays below the top face.
	require.GreaterOrEqual(t, adapterBox.Max.Z, clamparm.AdapterStudClear)
	require.Less(t, nutBox.Max.Z, adapterBox.Max.Z)
	// The nut sits in its pocket without touching the adapter body.
	requireDisjoint(t, nut.Body, adapter)
}
