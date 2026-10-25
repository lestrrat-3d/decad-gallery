package cycloidal

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func verifyBody(t *testing.T, name string, body *decad.Body) {
	t.Helper()
	require.True(t, body.IsSolid(), name)
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err, name)
	require.True(t, mesh.BoundaryVerified(), name)
	require.True(t, mesh.VolumeVerified(), name)
}

func TestCamAndOutput(t *testing.T) {
	for name, discs := range map[string]int{"one disc": 1, "two discs": 2} {
		t.Run(name, func(t *testing.T) {
			p := DefaultParams()
			p.DiscCount = discs
			doc := decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))
			r, err := begin(t.Context(), doc, p)
			require.NoError(t, err)
			cam, err := r.buildCam(t.Context())
			require.NoError(t, err)
			verifyBody(t, "cam", cam)
			plate, pin, err := r.buildOutput(t.Context())
			require.NoError(t, err)
			verifyBody(t, "plate", plate)
			pins, err := r.patternPins(t.Context(), pin)
			require.NoError(t, err)
			require.Len(t, pins, p.OutputPinCount)
			for k, pin := range pins {
				verifyBody(t, "pin", pin)
				// Pin k stands at angle 2πk/M on the output pin circle.
				c, err := pin.Centroid()
				require.NoError(t, err)
				want := r.outputPinCenter(k)
				require.InDelta(t, want.x, c.Value.X, 1e-6)
				require.InDelta(t, want.y, c.Value.Y, 1e-6)
			}
			box, err := cam.Bounds()
			require.NoError(t, err)
			stack := float64(discs)*p.DiscThickness + float64(discs-1)*p.DiscGap
			require.InDelta(t, stack, box.Max.Z-box.Min.Z, 1e-6)
			pinBox, err := pins[0].Bounds()
			require.NoError(t, err)
			require.InDelta(t, stack+planeGap+p.OutputPlateThickness, pinBox.Max.Z-pinBox.Min.Z, 1e-6)
			require.InDelta(t, 0, pinBox.Min.Z, 1e-6)
			plateBox, err := plate.Bounds()
			require.NoError(t, err)
			require.InDelta(t, p.OutputPlateThickness, plateBox.Max.Z-plateBox.Min.Z, 1e-6)
		})
	}
}
