package swisslever_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/swisslever"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestInventory(t *testing.T) {
	doc, m := swisslever.SharedAt(t, swisslever.DefaultParams().Phase)
	requireInventory(t, doc, m)

	doc, m = swisslever.SharedAt(t, 0.6)
	requireInventory(t, doc, m)
	require.Equal(t, 0.6, m.State.Phase)
}

func TestBuildValidates(t *testing.T) {
	doc := swisslever.NewDocument()
	p := swisslever.DefaultParams()
	p.Amplitude = 0.1
	_, err := swisslever.Build(t.Context(), doc, p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Amplitude")
	require.Empty(t, doc.Bodies(), "Build rejects the parameters before it touches the document")
	_, err = swisslever.Build(nil, doc, swisslever.DefaultParams()) //nolint:staticcheck // the nil context is the case under test
	require.Error(t, err)
	_, err = swisslever.Build(t.Context(), nil, swisslever.DefaultParams())
	require.Error(t, err)
	require.Empty(t, doc.Bodies())
}

func TestSketches(t *testing.T) {
	_, m := swisslever.SharedAt(t, swisslever.DefaultParams().Phase)
	require.NotEmpty(t, m.Sketches)
	for i, s := range m.Sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
	}
}

func TestPlacement(t *testing.T) {
	_, base := swisslever.SharedAt(t, swisslever.DefaultParams().Phase)
	frame, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p := swisslever.DefaultParams()
	p.Placement = &frame
	doc, m, err := swisslever.BuildForTest(t, p)
	require.NoError(t, err)
	requireInventory(t, doc, m)

	want, got := base.Named(), m.Named()
	for i := range want {
		wv, err := want[i].Body.Volume()
		require.NoError(t, err)
		gv, err := got[i].Body.Volume()
		require.NoError(t, err)
		require.InDelta(t, wv.Value.Base(), gv.Value.Base(), 1e-3*wv.Value.Base(), "volume of %s", want[i].Name)
		wb, err := want[i].Body.Bounds()
		require.NoError(t, err)
		gb, err := got[i].Body.Bounds()
		require.NoError(t, err)
		// The frame maps (x, y, z) to (10 + z, -5 + x, 7 + y).
		require.InDelta(t, 10+wb.Min.Z, gb.Min.X, 2e-3, "bounds of %s", want[i].Name)
		require.InDelta(t, 10+wb.Max.Z, gb.Max.X, 2e-3, want[i].Name)
		require.InDelta(t, -5+wb.Min.X, gb.Min.Y, 2e-3, want[i].Name)
		require.InDelta(t, -5+wb.Max.X, gb.Max.Y, 2e-3, want[i].Name)
		require.InDelta(t, 7+wb.Min.Y, gb.Min.Z, 2e-3, want[i].Name)
		require.InDelta(t, 7+wb.Max.Y, gb.Max.Z, 2e-3, want[i].Name)
	}
}
