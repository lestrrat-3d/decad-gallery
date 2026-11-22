package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentWidthMargins checks the margins that set the box's width: the bolt sites against the plates, ribs
// and cams, the columns against the top flange ribs, and the rear column against rotor a.
func TestSevenSegmentWidthMargins(t *testing.T) {
	const slack = 1e-9
	t.Run("site material", func(t *testing.T) {
		dm, err := DefaultParams().Derive()
		require.NoError(t, err)
		hp := m2fastener.HeadPocketRadius
		flats := m2fastener.NutCavityRadius * math.Sqrt(3) / 2
		require.InDelta(t, WallHalfWidth+1.5, wallHalf, slack)
		require.LessOrEqual(t, CamRadius, wallHalf+slack)
		for i, s := range dm.Sites {
			x, y := math.Abs(s.Head.X), s.Head.Y
			switch {
			case i < 4:
				require.LessOrEqual(t, x+hp+1.0, WallHalfWidth-wallChamfer+slack, "T%d head pocket", i+1)
				require.GreaterOrEqual(t, x-flats-1.0, ribInner-slack, "T%d nut pocket, rib inner side", i+1)
				require.LessOrEqual(t, x+flats+1.0, WallHalfWidth+slack, "T%d nut pocket, rib end", i+1)
			case i < 8:
				require.GreaterOrEqual(t, x-hp-1.0, ribInner-slack, "G%d head pocket, rib inner side", i-3)
				require.LessOrEqual(t, x+hp+1.0, WallHalfWidth+slack, "G%d head pocket, rib end", i-3)
				require.LessOrEqual(t, x+flats+1.0, WallHalfWidth-wallChamfer+slack, "G%d nut pocket", i-3)
				if i >= 6 {
					require.GreaterOrEqual(t, y-hp, rearInnerY-rearLedgeDeep+1.0-slack, "G%d head pocket, rear rib end", i-3)
				}
			default:
				require.GreaterOrEqual(t, housingFlangeR-chamfer-(x+hp), 1.0-slack, "H%d head pocket", i-7)
				require.LessOrEqual(t, housingFlangeR, wallHalf+slack, "H%d flange", i-7)
				require.LessOrEqual(t, x+flats+1.0, wallHalf-wallChamfer+slack, "H%d nut pocket", i-7)
			}
		}
	})

	t.Run("G heads clear the cams", func(t *testing.T) {
		dm, err := DefaultParams().Derive()
		require.NoError(t, err)
		hp := m2fastener.HeadPocketRadius
		for i := 4; i < 8; i++ {
			site := dm.Sites[i]
			x, y := math.Abs(site.Head.X), site.Head.Y
			require.LessOrEqual(t, HubRadius+1.0, x-hp+slack, "G%d head pocket, hub", i-3)
			for _, s := range Segments() {
				lo, hi := dm.camDiscFace(s), dm.camDiscFace(s)+CamThickness
				gap := math.Max(lo-(y+hp), (y-hp)-hi)
				require.GreaterOrEqual(t, gap, 1.0-slack, "G%d head pocket, cam %s", i-3, s)
			}
		}
	})

	t.Run("columns clear the top flange ribs", func(t *testing.T) {
		_, guide := sharedBody(t, "guide plate", (*run).columnGuidePlate)
		cols := planSection(t, guide, 77)
		for _, front := range []bool{true, false} {
			_, wall := sharedWall(t, front)
			gap := loopsGap(cols, planSection(t, wall, 77))
			t.Logf("front wall %v: column to rib gap %.3f", front, gap)
			require.GreaterOrEqual(t, gap, LooseFit-fitBound, "front wall %v", front)
		}
	})

	t.Run("rotor a clears the rear column", func(t *testing.T) {
		_, guide := sharedBody(t, "guide plate", (*run).columnGuidePlate)
		least := math.Inf(1)
		for _, p := range densify(planSection(t, guide, columnTop-0.05), 0.05) {
			if p.Y <= 0 {
				continue
			}
			gap := math.Hypot(p.Y-SegmentSpan, rotorSettledZ-columnTop) - rotorSweptRadius(math.Abs(p.X)-rotorEndPlay)
			least = math.Min(least, gap)
		}
		t.Logf("least gap to rotor a's swept envelope %.3f", least)
		require.GreaterOrEqual(t, least, LooseFit)
	})
}
