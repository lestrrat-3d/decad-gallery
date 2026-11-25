package cycloidal_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/cycloidal"
	"github.com/stretchr/testify/require"
)

// The printed rings pressed on a shaft or pin keep 1.6 mm of wall over their whole pressed length ("Corners and
// roots" in docs/model-pipeline.md): the output plate's rim and the wall between its sockets, and the cam's wall
// round the input bore, which for two discs is the neck between the sections. A section's measure can read up to
// 0.03 mm under the nominal wall, so each wall is proven to 1.6 less that.
func TestPressFitWalls(t *testing.T) {
	const wall, section = 1.6, 0.03
	for name, discs := range map[string]int{"one disc": 1, "two discs": 2} {
		t.Run(name, func(t *testing.T) {
			p := cycloidal.DefaultParams()
			p.DiscCount = discs
			_, drive := sharedDrive(t, p)

			// The plate's rim: its radius, less the socket circle's reach.
			plate, err := drive.OutputPlate.Bounds()
			require.NoError(t, err)
			pin, err := drive.OutputPins[0].Bounds()
			require.NoError(t, err)
			pinDiameter := pin.Max.X - pin.Min.X
			socket := pinDiameter + 0.1
			rop := p.OutputPinCircleDiameter / 2
			require.GreaterOrEqual(t, plate.Max.X-(rop+socket/2), wall-section, "plate rim")
			between := 2*rop*math.Sin(math.Pi/float64(p.OutputPinCount)) - socket
			require.GreaterOrEqual(t, between, wall-section, "wall between sockets")

			// The cam's bore wall, at the middle of a section and, for two discs, of the neck's straight bridge.
			heights := []float64{p.DiscThickness / 2}
			if discs == 2 {
				heights = append(heights, p.DiscThickness+p.DiscGap/2)
			}
			for _, z := range heights {
				least, most := loopRadii(t, drive.Cam, z)
				require.Len(t, least, 2, "cam section at %.2f", z)
				bore, outer := 0, 1
				if least[0] > least[1] {
					bore, outer = 1, 0
				}
				require.InDelta(t, p.InputShaftDiameter/2+0.05, most[bore], section, "bore radius at %.2f", z)
				require.GreaterOrEqual(t, least[outer]-most[bore], wall-section, "cam bore wall at %.2f", z)
			}
		})
	}
}
