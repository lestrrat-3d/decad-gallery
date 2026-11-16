package planetarygearbox_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// turned returns the angle that name's transform turns a probe point about the gearbox axis by, in radians in
// (-pi, pi], and whether the body moves at all.
func turned(t *testing.T, m *planetarygearbox.Motion, name string, input float64) (float64, bool) {
	t.Helper()
	xf, moves, err := m.Transform(name, input)
	require.NoError(t, err, name)
	if !moves {
		return 0, false
	}
	probe := r3.NewVec(1, 0, 0)
	moved := xf.Apply(probe)
	return math.Atan2(moved.Y, moved.X), true
}

func TestMotionRatio(t *testing.T) {
	kit, err := planetarygearbox.DefaultParamsFor("planet608-shaft608")
	require.NoError(t, err)
	two := kit
	two.Stages = 2
	for _, tc := range []struct {
		name   string
		p      planetarygearbox.Params
		mode   planetarygearbox.DriveMode
		expect float64
	}{
		{"default sun-driven", planetarygearbox.DefaultParams(), planetarygearbox.SunDriven, 13.0 / 84},
		{"default carrier-driven", planetarygearbox.DefaultParams(), planetarygearbox.CarrierDriven, 84.0 / 13},
		{"14/28/70 sun-driven", kit, planetarygearbox.SunDriven, 1.0 / 6},
		{"14/28/70 carrier-driven", kit, planetarygearbox.CarrierDriven, 6},
		{"two stages sun-driven", two, planetarygearbox.SunDriven, 1.0 / 36},
		{"two stages carrier-driven", two, planetarygearbox.CarrierDriven, 36},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := planetarygearbox.NewMotion(tc.p, tc.mode)
			require.NoError(t, err)
			require.Equal(t, tc.mode, m.Mode())
			require.InDelta(t, tc.expect, m.Ratio(), 1e-12)
			require.InDelta(t, 0.3*tc.expect, m.OutputAngle(0.3), 1e-12)
		})
	}
}

func TestMotionSunDriven(t *testing.T) {
	kit, err := planetarygearbox.DefaultParamsFor("planet608-shaft608")
	require.NoError(t, err)
	kit.Stages = 2
	m, err := planetarygearbox.NewMotion(kit, planetarygearbox.SunDriven)
	require.NoError(t, err)
	const input = 0.6
	for name, want := range map[string]float64{
		planetarygearbox.InputShaftName:  input,
		planetarygearbox.SunName(0):      input,
		planetarygearbox.CarrierName(0):  input / 6,
		planetarygearbox.ShaftName(0, 2): input / 6,
		planetarygearbox.SunName(1):      input / 6,
		planetarygearbox.CarrierName(1):  input / 36,
		planetarygearbox.OutputShaftName: input / 36,
		planetarygearbox.PinName(1, 0):   input / 36,
		planetarygearbox.FrontCollarName: input,
		planetarygearbox.RearCollarName:  input / 36,
	} {
		got, moves := turned(t, m, name, input)
		require.True(t, moves, name)
		require.InDelta(t, want, got, 1e-9, name)
	}
	for _, name := range []string{
		planetarygearbox.RingName, planetarygearbox.FrontCoverName, planetarygearbox.RearCoverName,
		planetarygearbox.FrontBearingName, planetarygearbox.RearBearingName, planetarygearbox.EarBoltName(0),
		planetarygearbox.EarNutName(0), "Unknown",
	} {
		_, moves := turned(t, m, name, input)
		require.False(t, moves, name)
	}
}

func TestMotionCarrierDriven(t *testing.T) {
	kit, err := planetarygearbox.DefaultParamsFor("planet608-shaft608")
	require.NoError(t, err)
	m, err := planetarygearbox.NewMotion(kit, planetarygearbox.CarrierDriven)
	require.NoError(t, err)
	const input = 0.1
	for name, want := range map[string]float64{
		planetarygearbox.CarrierName(0):  input,
		planetarygearbox.OutputShaftName: input,
		planetarygearbox.SunName(0):      input * 6,
		planetarygearbox.InputShaftName:  input * 6,
		planetarygearbox.FrontCollarName: input * 6,
		planetarygearbox.RearCollarName:  input,
	} {
		got, moves := turned(t, m, name, input)
		require.True(t, moves, name)
		require.InDelta(t, want, got, 1e-9, name)
	}
	_, moves := turned(t, m, planetarygearbox.RingName, input)
	require.False(t, moves)
}

// TestMotionPlanetRolls checks that a planet, carried by its carrier, turns on its pin by -Zr/Zp of the carrier's
// angle, in both modes: its tooth direction turns by (1 - Zr/Zp) times the carrier's angle.
func TestMotionPlanetRolls(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	for _, mode := range []planetarygearbox.DriveMode{planetarygearbox.SunDriven, planetarygearbox.CarrierDriven} {
		m, err := planetarygearbox.NewMotion(p, mode)
		require.NoError(t, err)
		const input = 0.2
		carried, _ := turned(t, m, planetarygearbox.CarrierName(0), input)
		xf, moves, err := m.Transform(planetarygearbox.PlanetName(0, 0), input)
		require.NoError(t, err)
		require.True(t, moves)
		dir := xf.ApplyDir(r3.NewVec(1, 0, 0))
		want := (1 - float64(p.RingTeeth)/float64(p.PlanetTeeth)) * carried
		require.InDelta(t, want, math.Atan2(dir.Y, dir.X), 1e-9)
	}
}

func TestNewMotionRejects(t *testing.T) {
	_, err := planetarygearbox.NewMotion(planetarygearbox.DefaultParams(), planetarygearbox.DriveMode(9))
	require.Error(t, err)
	_, err = planetarygearbox.NewMotion(planetarygearbox.Params{}, planetarygearbox.SunDriven)
	require.Error(t, err)
}

func TestBuildPartsFilesNothing(t *testing.T) {
	doc := planetarygearbox.NewDocument()
	box, err := planetarygearbox.BuildParts(t.Context(), doc, planetarygearbox.DefaultParams())
	require.NoError(t, err)
	parts := box.Parts()
	named := box.Named()
	require.Len(t, parts, len(named))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		require.NotContains(t, seen, part.Name)
		seen[part.Name] = struct{}{}
		require.NotEmpty(t, part.Container, part.Name)
		c, err := doc.ContainerOf(part.Body)
		require.NoError(t, err, part.Name)
		require.NotEqual(t, part.Container, c.Name(), "%s is in no container yet", part.Name)
	}
	for _, nb := range named {
		require.Contains(t, seen, nb.Name)
	}
}
