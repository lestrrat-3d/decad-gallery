package turbinefan_test

import (
	"cmp"
	"fmt"
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// clutchGapMesh tessellates body at tol with every verification on and wraps it for gap queries.
func clutchGapMesh(t *testing.T, body *decad.Body, tol float64) meshgap.Mesh {
	t.Helper()
	mesh := meshOf(t, body, tol)
	require.True(t, mesh.BoundaryVerified())
	mg, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return mg
}

// TestTurbineFanOnClutch builds the real one-way clutch with its adapter in one document and the clutch-mount fan placed
// on it in another, and checks the face interface, the stack and every pair of bodies through the pair. The two documents
// are separate because both models file their hub in a container named Hub.
func TestTurbineFanOnClutch(t *testing.T) {
	for _, cp := range []onewayclutch.Params{
		onewayclutch.DefaultParams(),
		{Shaft: "608ZZ", Freewheel: onewayclutch.CounterClockwise, BoreClearance: 0.05},
	} {
		t.Run(cp.Shaft, func(t *testing.T) { requireFanOnClutch(t, cp) })
	}
}

func requireFanOnClutch(t *testing.T, cp onewayclutch.Params) {
	t.Helper()
	cdoc := onewayclutch.NewDocument()
	clutch, err := onewayclutch.Build(t.Context(), cdoc, cp)
	require.NoError(t, err)
	adapter, err := onewayclutch.BuildAdapter(t.Context(), cdoc, cp)
	require.NoError(t, err)
	fp, err := turbinefan.ParamsForClutch(cp)
	require.NoError(t, err)
	fdoc := turbinefan.NewDocument()
	fan, err := turbinefan.Build(t.Context(), fdoc, fp)
	require.NoError(t, err)
	require.NoError(t, cdoc.VerifyContainers().Err())
	require.NoError(t, fdoc.VerifyContainers().Err())

	// Pairing and face alignment: the fan's adapter bolts sit on the adapter's first two face sites.
	cdm, err := cp.Derive()
	require.NoError(t, err)
	require.Equal(t, cp.Freewheel, fp.Hand.Freewheel())
	// The hub's bore flat (-Y in the fan frame) points where the clutch's shaft flat does (+Y in the clutch frame).
	mount, err := r3.FromFrame(*fp.Placement)
	require.NoError(t, err)
	require.InDelta(t, 0, mount.ApplyDir(r3.Vec{Y: -1}).Sub(r3.Vec{Y: 1}).Len(), 1e-9, "bore flat direction")
	for k, site := range fan.AdapterSites {
		want := cdm.FaceSites[k+1] // the fan holes lie at clutch angles 210 and 330; face hole 1 (90) stays empty
		require.InDelta(t, 0, site.Head.Sub(want.Head).Len(), 1e-9, "adapter bolt %d head", k+1)
		require.InDelta(t, 0, site.Dir.Sub(want.Dir).Len(), 1e-9, "adapter bolt %d direction", k+1)
		require.InDelta(t, 0, site.Corner.Sub(want.Corner).Len(), 1e-9, "adapter bolt %d corner", k+1)
		require.InDelta(t, want.Cavity, site.Cavity, 1e-9)
		require.InDelta(t, want.Length, site.Length, 0)
	}

	// Overlap pass: proves "clear" only. Meshes of printed parts at 0.03 and hardware and blades at 0.05.
	type part struct {
		name string
		body *decad.Body
		tol  float64
	}
	parts := []part{
		{"clutch hub", clutch.Hub, 0.03}, {"clutch ring", clutch.Ring, 0.03}, {"back plate", clutch.BackPlate, 0.03},
		{"cover", clutch.Cover, 0.03}, {"adapter", adapter.Body, 0.03}, {"fan hub", fan.Hub, 0.03},
	}
	for i := range 3 {
		parts = append(parts,
			part{fmt.Sprintf("blade %d", i+1), fan.Blades[i], 0.05},
			part{fmt.Sprintf("blade bolt %d", i+1), fan.BladeBolts[i].Body, 0.05},
			part{fmt.Sprintf("blade nut %d", i+1), fan.BladeNuts[i].Body, 0.05},
			part{fmt.Sprintf("clutch bolt %d", i+1), clutch.Bolts[i].Body, 0.05},
			part{fmt.Sprintf("clutch nut %d", i+1), clutch.Nuts[i].Body, 0.05},
			part{fmt.Sprintf("ear bolt %d", i+1), adapter.Bolts[i].Body, 0.05},
			part{fmt.Sprintf("ear nut %d", i+1), adapter.Nuts[i].Body, 0.05})
	}
	for i := range 2 {
		parts = append(parts,
			part{fmt.Sprintf("adapter bolt %d", i+1), fan.AdapterBolts[i].Body, 0.05},
			part{fmt.Sprintf("adapter nut %d", i+1), fan.AdapterNuts[i].Body, 0.05})
	}
	slices.SortFunc(parts, func(a, b part) int { return cmp.Compare(a.name, b.name) })
	meshes := map[string]meshgap.Mesh{}
	for _, p := range parts {
		meshes[p.name] = clutchGapMesh(t, p.body, p.tol)
	}
	// A head rests on its pocket's floor, so a bolt and the part it sits in touch; "TestTurbineFanClutchAdapterBolts"
	// and the clutch's own tests measure those fits.
	skip := map[[2]string]struct{}{}
	for i := 1; i <= 3; i++ {
		skip[[2]string{fmt.Sprintf("clutch bolt %d", i), fmt.Sprintf("clutch nut %d", i)}] = struct{}{}
		skip[[2]string{fmt.Sprintf("clutch bolt %d", i), "cover"}] = struct{}{}
		skip[[2]string{fmt.Sprintf("ear bolt %d", i), fmt.Sprintf("ear nut %d", i)}] = struct{}{}
		skip[[2]string{fmt.Sprintf("ear bolt %d", i), "adapter"}] = struct{}{}
		skip[[2]string{fmt.Sprintf("blade bolt %d", i), fmt.Sprintf("blade nut %d", i)}] = struct{}{}
		skip[[2]string{fmt.Sprintf("blade bolt %d", i), "fan hub"}] = struct{}{}
	}
	for i := 1; i <= 2; i++ {
		skip[[2]string{fmt.Sprintf("adapter bolt %d", i), fmt.Sprintf("adapter nut %d", i)}] = struct{}{}
		skip[[2]string{fmt.Sprintf("adapter bolt %d", i), "adapter"}] = struct{}{}
	}
	for i, a := range parts {
		for _, b := range parts[i+1:] {
			_, hit1 := skip[[2]string{a.name, b.name}]
			_, hit2 := skip[[2]string{b.name, a.name}]
			if hit1 || hit2 {
				continue
			}
			clear, err := meshes[a.name].ExceedsClearance(meshes[b.name], 0)
			require.NoError(t, err)
			require.True(t, clear, "%s and %s overlap or touch", a.name, b.name)
		}
	}

	// Gap lower bounds between the pairs the stack and the blades make.
	for _, g := range []struct {
		a, b  string
		least float64
	}{
		{"fan hub", "adapter", 0.20},       // the modeled seam, 0.25
		{"adapter", "clutch ring", 0.20},   // the seam, 0.25
		{"adapter", "cover", 1.45},         // modeled 1.5
		{"adapter", "clutch bolt 1", 1.45}, // the clutch's own bolt heads
		{"adapter bolt 1", "cover", 1.45},  // head top flush with the adapter's rear face
		{"blade 1", "adapter", 5.0},        // the blades stand clear of the adapter
		{"blade 2", "adapter", 5.0},
		{"blade 3", "adapter", 5.0},
		{"blade 1", "ear bolt 1", 5.0}, // hex key room between the blades
	} {
		gap, _, err := meshes[g.a].Exceeds(meshes[g.b], 50)
		require.NoError(t, err)
		t.Logf("gap lower bound %s / %s: %.3f", g.a, g.b, gap)
		require.GreaterOrEqual(t, gap, g.least, "%s / %s", g.a, g.b)
	}

	// Axial z of the stack: the adapter's rear face (head tops flush) and front face, and the hub's back face.
	for k := range fan.AdapterBolts {
		box, err := fan.AdapterBolts[k].Body.Bounds()
		require.NoError(t, err)
		tol := box.Bound.Base() + 2e-3
		require.InDelta(t, onewayclutch.AdapterFrom, box.Min.Z, tol, "adapter bolt %d head top", k+1)
		require.InDelta(t, onewayclutch.AdapterFrom+fan.AdapterSites[k].Tip(), box.Max.Z, tol, "adapter bolt %d tip", k+1)
		nb, err := fan.AdapterNuts[k].Body.Bounds()
		require.NoError(t, err)
		ntol := nb.Bound.Base() + 2e-3
		require.GreaterOrEqual(t, nb.Min.Z, turbinefan.MountZ+1.5-ntol, "adapter nut %d inside its slot", k+1)
		require.LessOrEqual(t, nb.Max.Z, turbinefan.MountZ+3.5+ntol, "adapter nut %d inside its slot", k+1)
	}
	abox, err := adapter.Body.Bounds()
	require.NoError(t, err)
	hbox, err := fan.Hub.Bounds()
	require.NoError(t, err)
	require.InDelta(t, onewayclutch.AdapterTo, abox.Max.Z, abox.Bound.Base()+2e-3)
	require.InDelta(t, onewayclutch.AdapterTo+turbinefan.ClutchSeam, hbox.Min.Z, hbox.Bound.Base()+2e-3)
}
