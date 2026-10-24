package cycloidal_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func chainDocument() *decad.Document {
	return decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))
}

func TestBuildDefaultDrive(t *testing.T) {
	doc := chainDocument()
	p := cycloidal.DefaultParams()
	p.ChamferSize = 0
	drive, err := cycloidal.Build(t.Context(), doc, p)
	require.NoError(t, err)
	named := drive.Named()
	require.Len(t, named, 1+1+1+1+6)
	require.Len(t, doc.Bodies(), len(named))
	for _, nb := range named {
		require.True(t, nb.Body.IsSolid(), nb.Name)
		mesh, err := nb.Body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, nb.Name)
		require.True(t, mesh.BoundaryVerified(), nb.Name)
		require.True(t, mesh.VolumeVerified(), nb.Name)
	}
	require.Equal(t, "Cycloidal Disk 1", named[0].Name)
	require.Equal(t, "Output Pin 6", named[len(named)-1].Name)
}

func TestBuildTwoDiscDrive(t *testing.T) {
	doc := chainDocument()
	p := cycloidal.DefaultParams()
	p.ChamferSize = 0
	p.DiscCount = 2
	drive, err := cycloidal.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, drive.Discs, 2)
	named := drive.Named()
	require.Equal(t, "Cycloidal Disk 2", named[1].Name)
	for _, nb := range named {
		mesh, err := nb.Body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, nb.Name)
		require.True(t, mesh.BoundaryVerified(), nb.Name)
		require.True(t, mesh.VolumeVerified(), nb.Name)
	}
}

// overlap returns the volume the two bodies share, zero when they only touch or are apart.
func overlap(t *testing.T, a, b *decad.Body) float64 {
	t.Helper()
	both, err := decad.Intersect(t.Context(), a, b)
	if err != nil {
		require.ErrorContains(t, err, "the intersect result is empty")
		return 0
	}
	volume, err := both.Volume()
	require.NoError(t, err)
	return volume.Value.Base()
}

// Every moving part clears the parts around it at the home pose: disc against housing, cam, and pins.
func TestDriveClearances(t *testing.T) {
	doc := chainDocument()
	p := cycloidal.DefaultParams()
	p.ChamferSize = 0
	drive, err := cycloidal.Build(t.Context(), doc, p)
	require.NoError(t, err)
	disc := drive.Discs[0]
	require.Zero(t, overlap(t, disc, drive.Housing), "the rotor clears the casing")
	require.Zero(t, overlap(t, disc, drive.Cam), "the rotor clears the cam")
	for i, pin := range drive.OutputPins {
		require.Zero(t, overlap(t, disc, pin), "the rotor clears output pin %d", i+1)
	}
	require.Zero(t, overlap(t, drive.Housing, drive.Cam))
	require.Zero(t, overlap(t, drive.OutputPlate, drive.Housing))
}

// Each option the add-in offers builds a drive whose bodies all pass 0.1 mm VerifyAll.
func TestBuildOptions(t *testing.T) {
	t.Parallel()
	placed, err := r3.NewFrame(r3.NewVec(10, 20, 30), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	cases := []struct {
		name string
		mod  func(p *cycloidal.Params)
	}{
		{"solid_cam", func(p *cycloidal.Params) { p.InputShaftDiameter = 0 }},
		{"pin_diameters", func(p *cycloidal.Params) { p.PinDiameter, p.OutputPinDiameter = 8, 6 }},
		{"pins_12", func(p *cycloidal.Params) { p.PinCount = 12 }},
		{"pins_24", func(p *cycloidal.Params) { p.PinCount = 24 }},
		{"output_pins_5", func(p *cycloidal.Params) { p.OutputPinCount = 5 }},
		{"two_discs_pins_20", func(p *cycloidal.Params) { p.DiscCount, p.PinCount, p.DiscGap = 2, 20, 1 }},
		{"placement", func(p *cycloidal.Params) { p.Placement = &placed }},
		{"thin_housing", func(p *cycloidal.Params) { p.HousingWall, p.BaseThickness, p.OutputPlateThickness = 1, 2, 2 }},
		{"clearances", func(p *cycloidal.Params) { p.DiskClearance, p.BearingClearance = 0.5, 0.4 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := cycloidal.DefaultParams()
			p.ChamferSize = 0
			tc.mod(&p)
			require.Empty(t, cycloidal.Problems(p))
			doc := chainDocument()
			drive, err := cycloidal.Build(t.Context(), doc, p)
			require.NoError(t, err)
			for _, nb := range drive.Named() {
				mesh, err := nb.Body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
				require.NoError(t, err, nb.Name)
				require.True(t, mesh.BoundaryVerified(), nb.Name)
				require.True(t, mesh.VolumeVerified(), nb.Name)
			}
		})
	}
}

// ChamferSize chamfers the housing, the output plate and the pins, and leaves the rotor discs sharp.
func TestBuildChamfers(t *testing.T) {
	t.Parallel()
	volumes := func(chamfer float64) map[string]float64 {
		p := cycloidal.DefaultParams()
		p.ChamferSize = chamfer
		drive, err := cycloidal.Build(t.Context(), chainDocument(), p)
		require.NoError(t, err)
		out := map[string]float64{}
		for _, nb := range drive.Named() {
			mesh, err := nb.Body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err, nb.Name)
			require.True(t, mesh.VolumeVerified(), nb.Name)
			volume, err := nb.Body.Volume()
			require.NoError(t, err)
			out[nb.Name] = volume.Value.Base()
		}
		return out
	}
	sharp, chamfered := volumes(0), volumes(0.5)
	require.InDelta(t, sharp["Cycloidal Disk 1"], chamfered["Cycloidal Disk 1"], 1e-6, "discs are not chamfered")
	require.InDelta(t, sharp["Eccentric Cam"], chamfered["Eccentric Cam"], 1e-6, "the cam is not chamfered")
	for _, name := range []string{"Housing", "Output Plate", "Output Pin 1"} {
		require.Less(t, chamfered[name], sharp[name]-1e-3, name)
	}
}
