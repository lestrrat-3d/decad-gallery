package handcranksiren_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestGearboxParams(t *testing.T) {
	p, err := handcranksiren.GearboxParams()
	require.NoError(t, err)
	require.Equal(t, 14, p.SunTeeth)
	require.Equal(t, 28, p.PlanetTeeth)
	require.Equal(t, 70, p.RingTeeth)
	require.Equal(t, "608ZZ", p.PlanetBearing)
	require.Equal(t, "608ZZ", p.ShaftBearing)
	require.InDelta(t, 24.05, p.InputShaftLength, 1e-9)
	require.InDelta(t, 22.05, p.OutputShaftLength, 1e-9)
	require.NotNil(t, p.Placement)
	require.Equal(t, r3.Vec{Z: 84}, p.Placement.Origin())
	require.Equal(t, r3.Vec{Z: 1}, p.Placement.U())
	require.Equal(t, r3.Vec{X: 1}, p.Placement.V())
}

// TestGearboxSpike builds the planetary gearbox with the siren's parameters, then the current siren (rotor and crank)
// in the same document, and reports the build times and heap use.
func TestGearboxSpike(t *testing.T) {
	p, err := handcranksiren.GearboxParams()
	require.NoError(t, err)
	doc := planetarygearbox.NewDocument()
	start := time.Now()
	box, err := planetarygearbox.BuildParts(t.Context(), doc, p)
	require.NoError(t, err)
	t.Logf("gearbox build: %s, %d parts", time.Since(start).Round(time.Millisecond), len(box.Parts()))
	mid := time.Now()
	_, err = handcranksiren.Build(t.Context(), doc, handcranksiren.DefaultParams())
	require.NoError(t, err)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	t.Logf("siren build: %s; heap sys %d MiB", time.Since(mid).Round(time.Millisecond), m.Sys>>20)
}
