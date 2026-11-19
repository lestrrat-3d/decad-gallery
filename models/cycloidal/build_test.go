package cycloidal_test

// CI: isolated-tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestBuildDefaultDrive(t *testing.T) {
	p := cycloidal.DefaultParams()
	doc, drive, err := defaultFixture(t.Context(), p)
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
	p.DiscCount, p.DiscGap = 2, cycloidal.MinCamNeck
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

// Every moving part clears the parts around it at the home pose: disc against housing, cam, and pins.
func TestDriveClearances(t *testing.T) {
	doc := chainDocument()
	p := cycloidal.DefaultParams()
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
		{"two_discs_pins_20", func(p *cycloidal.Params) { p.DiscCount, p.PinCount, p.DiscGap = 2, 20, 1.4 }},
		{"two_discs_chamfered", func(p *cycloidal.Params) {
			p.DiscCount, p.DiscGap, p.ChamferSize = 2, cycloidal.MinCamNeck, cycloidal.MinChamferSize
		}},
		{"pins_12_chamfered", func(p *cycloidal.Params) { p.PinCount, p.ChamferSize = 12, 0.5 }},
		{"pins_20_chamfered", func(p *cycloidal.Params) { p.PinCount, p.ChamferSize = 20, 0.5 }},
		{"clearances_chamfered", func(p *cycloidal.Params) {
			p.DiskClearance, p.BearingClearance, p.ChamferSize = cycloidal.MinClearance, 2*cycloidal.MinClearance, cycloidal.MinChamferSize
		}},
		{"placement", func(p *cycloidal.Params) { p.Placement = &placed }},
		{"thin_housing", func(p *cycloidal.Params) { p.HousingWall, p.BaseThickness, p.OutputPlateThickness = 1.6, 2, 2 }},
		{"clearances", func(p *cycloidal.Params) { p.DiskClearance, p.BearingClearance = 0.6, 1.2 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := cycloidal.DefaultParams()
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

// ChamferSize chamfers the rotor discs, housing, cam, output plate and pins.
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
	standard, larger := volumes(cycloidal.MinChamferSize), volumes(cycloidal.MinChamferSize+0.25)
	for _, name := range []string{"Cycloidal Disk 1", "Housing", "Eccentric Cam", "Output Plate", "Output Pin 1"} {
		require.Less(t, larger[name], standard[name]-1e-3, name)
	}
}

// A two-disc cam is two sections flush with the discs' faces, joined by a bridge across the gap.
func TestTwoDiscCam(t *testing.T) {
	p := cycloidal.DefaultParams()
	p.DiscCount, p.DiscGap = 2, cycloidal.MinCamNeck
	require.Empty(t, cycloidal.Problems(p))
	drive, err := cycloidal.Build(t.Context(), chainDocument(), p)
	require.NoError(t, err)
	for i, disc := range drive.Discs {
		require.Zero(t, overlap(t, disc, drive.Cam), "the cam clears disc %d", i+1)
		require.Zero(t, overlap(t, disc, drive.Housing), "the casing clears disc %d", i+1)
	}
	box, err := drive.Cam.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 2*p.DiscThickness+p.DiscGap, box.Max.Z-box.Min.Z, 1e-6)
	volume, err := drive.Cam.Volume()
	require.NoError(t, err)
	// Two cylinders of the cam diameter over the discs, and between them the bridge, a cylinder halfway between
	// the bore and the largest circle about the axis that fits inside both sections. The input bore runs
	// through all of it.
	// The bore is the press-fit shaft plus 0.05 mm on each side. The neck adds a 0.5 mm round (a ring of
	// 2π(r_b + 0.2234 r)·r²(1 - π/4)) where the bridge meets the lower section and a 0.5 mm chamfer (a ring of
	// 2π(r_b + L/3)·L²/2) where it meets the upper section.
	bore, cam := p.InputShaftDiameter/2+0.05, p.CenterBearingDiameter/2
	bridge := (cam - 0.05 - p.Eccentricity + p.InputShaftDiameter/2) / 2
	chamfer := p.ChamferSize
	chamferLoss := float64(2*p.DiscCount) * math.Pi * (cam*chamfer*chamfer - chamfer*chamfer*chamfer/3)
	const neck = 0.5
	round := 2 * math.Pi * (bridge + 0.2234*neck) * neck * neck * (1 - math.Pi/4)
	bevel := 2 * math.Pi * (bridge + neck/3) * neck * neck / 2
	want := math.Pi*((cam*cam-bore*bore)*2*p.DiscThickness+(bridge*bridge-bore*bore)*p.DiscGap) - chamferLoss +
		round + bevel
	require.InDelta(t, want, volume.Value.Base(), 1e-3*want)
}

func TestBuildContainers(t *testing.T) {
	for _, discs := range []int{1, 2} {
		t.Run(fmt.Sprintf("discs=%d", discs), func(t *testing.T) {
			doc := chainDocument()
			p := cycloidal.DefaultParams()
			if discs == 2 {
				p.DiscCount, p.DiscGap = 2, cycloidal.MinCamNeck
			}
			drive, err := cycloidal.Build(t.Context(), doc, p)
			require.NoError(t, err)
			require.NoError(t, doc.VerifyContainers().Err())

			var names []string
			for _, c := range doc.Root().Children() {
				names = append(names, c.Name())
			}
			require.ElementsMatch(t, []string{
				cycloidal.DiscsContainerName, cycloidal.HousingContainerName, cycloidal.CamContainerName,
				cycloidal.PlateContainerName, cycloidal.OutputPinsContainerName,
			}, names)
			require.Empty(t, doc.Root().Bodies())

			want := map[string][]*decad.Body{
				cycloidal.DiscsContainerName:      drive.Discs,
				cycloidal.HousingContainerName:    {drive.Housing},
				cycloidal.CamContainerName:        {drive.Cam},
				cycloidal.PlateContainerName:      {drive.OutputPlate},
				cycloidal.OutputPinsContainerName: drive.OutputPins,
			}
			for name, bodies := range want {
				c, err := doc.Root().ChildNamed(name)
				require.NoError(t, err)
				require.ElementsMatch(t, bodies, c.Bodies(), name)
			}
			require.Len(t, drive.Discs, discs)
		})
	}
}
