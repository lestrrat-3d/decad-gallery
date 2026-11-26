package m5fastener_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	require.NoError(t, site().Validate())
	s := site()
	s.Length = 17
	require.Error(t, s.Validate())
	s = site()
	s.Dir = r3.Vec{Z: 2}
	require.Error(t, s.Validate())
	s = site()
	s.Corner = r3.Vec{X: 1, Z: 1}.Scale(1 / r3.Vec{X: 1, Z: 1}.Len())
	require.Error(t, s.Validate())
}

func TestSite(t *testing.T) {
	s := site()
	require.InDelta(t, 19.5, s.Tip(), 1e-9)
	require.InDelta(t, 5, s.At(5).Z, 1e-9)
	f, err := s.Across(2)
	require.NoError(t, err)
	require.InDelta(t, 2, f.Origin().Z, 1e-9)
	require.InDelta(t, 1, f.U().X, 1e-9)
	require.InDelta(t, 1, f.N().Z, 1e-9)
}

func TestHoleTool(t *testing.T) {
	hole, sk, err := m5fastener.HoleTool(t.Context(), decad.New(), site())
	require.NoError(t, err)
	require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), sk))
	box := bounds(t, hole)
	require.InDelta(t, -1, box.Min.Z, 1e-3)
	require.InDelta(t, 20.5, box.Max.Z, 1e-3)
	require.InDelta(t, 2.8, box.Max.X, 1e-3)
}

// TestPocketTools cuts each pocket tool from a block whose face lies on the pocket's opening and checks the result is
// a solid whose removed volume is a hexagonal prism less its rounded floor.
func TestPocketTools(t *testing.T) {
	type tool func(*decad.Document, r3.Frame) (*decad.Body, *sketch.Sketch, error)
	ctx := t.Context()
	tests := []struct {
		name   string
		radius float64
		depth  float64
		round  float64
		build  tool
	}{
		{"head", m5fastener.HeadPocketRadius, m5fastener.HeadPocketDepth, m5fastener.PocketRound,
			func(d *decad.Document, f r3.Frame) (*decad.Body, *sketch.Sketch, error) {
				return m5fastener.HeadPocketTool(ctx, d, f)
			}},
		{"nut", m5fastener.NutPocketRadius, m5fastener.NutCavityHeight, m5fastener.PocketRound,
			func(d *decad.Document, f r3.Frame) (*decad.Body, *sketch.Sketch, error) {
				return m5fastener.NutPocketTool(ctx, d, f, m5fastener.NutCavityHeight)
			}},
		{"press", m5fastener.PressPocketRadius, m5fastener.HeadPocketDepth, m5fastener.PressPocketRound,
			func(d *decad.Document, f r3.Frame) (*decad.Body, *sketch.Sketch, error) {
				return m5fastener.PressPocketTool(ctx, d, f)
			}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := decad.New()
			floorZ := blockSize - tc.depth
			cutter, sk, err := tc.build(doc, frameAt(t, floorZ))
			require.NoError(t, err)
			require.NoError(t, sketchcheck.VerifyFullyConstrained(ctx, sk))

			box := bounds(t, cutter)
			require.InDelta(t, floorZ, box.Min.Z, 1e-3)
			require.InDelta(t, blockSize+m5fastener.Stock, box.Max.Z, 1e-3)
			require.InDelta(t, tc.radius, box.Max.X, 1e-3, "corners lie on U")
			require.InDelta(t, tc.radius*math.Sqrt(3)/2, box.Max.Y, 1e-3)

			host := block(t, doc)
			before := volume(t, host)
			cut, err := decad.Cut(host, cutter).Do(ctx)
			require.NoError(t, err)
			require.Greater(t, volume(t, cut), 0.0)
			removed := before - volume(t, cut)
			prism := 3 * math.Sqrt(3) / 2 * tc.radius * tc.radius * tc.depth
			// The floor round takes a ring off the prism's lower edge: less than a square section of the pocket's round round the
			// hexagon's perimeter (6*radius), more than a tenth of that.
			ring := 6 * tc.radius * tc.round * tc.round
			require.Less(t, removed, prism-ring/10, "floor is rounded")
			require.Greater(t, removed, prism-ring, "floor round is bounded")
		})
	}
}

// TestPressPocketRoundClearsHeadChamfer checks, from the dimensions alone, that the bolt head's 30 degree chamfer
// clears the press pocket's floor round at the flats: no point of the head's underside lies inside the round.
func TestPressPocketRoundClearsHeadChamfer(t *testing.T) {
	apothem := m5fastener.PressPocketRadius * math.Sqrt(3) / 2
	round := m5fastener.PressPocketRound
	headFlat := m5fastener.BoltSize.Hex.AcrossFlats / 2
	start := m5fastener.BoltSize.Hex.BearingDiameter / 2
	slope := math.Tan(30 * math.Pi / 180)
	require.Greater(t, apothem, headFlat, "head fits the flats")
	center := apothem - round // radial position where the round meets the floor
	minGap := math.Inf(1)
	for rho := center; rho <= headFlat; rho += 1e-4 {
		d := rho - center
		floorZ := round - math.Sqrt(round*round-d*d) // round's surface above the floor
		headZ := math.Max(0, slope*(rho-start))      // chamfer's underside above the floor
		minGap = math.Min(minGap, headZ-floorZ)
	}
	require.GreaterOrEqual(t, minGap, 0.0, "head chamfer clears the floor round at mid-flat")
}

func TestRoundedSocketTool(t *testing.T) {
	doc := decad.New()
	f := frameAt(t, 10)
	tool, sk, err := m5fastener.RoundedSocketTool(t.Context(), doc, f, 5, 4, 1)
	require.NoError(t, err)
	require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), sk))
	box := bounds(t, tool)
	require.InDelta(t, 10, box.Min.Z, 1e-3)
	require.InDelta(t, 15, box.Max.Z, 1e-3)
	require.InDelta(t, 5, box.Max.X, 1e-2)
	// A round floor corner of radius 1 removes about (1 - pi/4) * 1 * 2*pi*5 mm3 from the cylinder.
	full := math.Pi * 25 * 5
	require.Less(t, volume(t, tool), full-3)
}

func TestPlacement(t *testing.T) {
	doc := decad.New()
	s := site()
	bolt, err := m5fastener.PlaceBolt(t.Context(), doc, s)
	require.NoError(t, err)
	box := bounds(t, bolt.Body)
	require.InDelta(t, 0, box.Min.Z, 1e-3, "head end face at Head")
	require.InDelta(t, 19.5, box.Max.Z, 1e-3)

	nut, err := m5fastener.PlaceNut(t.Context(), doc, frameAt(t, 6))
	require.NoError(t, err)
	nb := bounds(t, nut.Body)
	require.InDelta(t, 6, nb.Min.Z, 1e-3)
	require.InDelta(t, 10.7, nb.Max.Z, 1e-3)
	require.InDelta(t, 4.6188, nb.Max.X, 1e-3, "corners on U")
}

func TestDimensionsMatchPreset(t *testing.T) {
	require.Equal(t, "M5", m5fastener.PresetName)
	require.Equal(t, fastener.HeadHex, m5fastener.BoltSize.Head)
	require.InDelta(t, m5fastener.BoltSize.ShankDiameter/2+m5fastener.HeadClearance, m5fastener.HoleRadius, 1e-12)
	require.InDelta(t, m5fastener.BoltSize.HeadHeight, m5fastener.HeadPocketDepth, 1e-12)
	corner := m5fastener.BoltSize.Hex.AcrossCorners() / 2
	require.InDelta(t, corner+m5fastener.HeadClearance, m5fastener.HeadPocketRadius, 1e-4)
	require.InDelta(t, m5fastener.NutSize.AcrossCorners()/2+m5fastener.NutClearance, m5fastener.NutPocketRadius, 1e-4)
	require.InDelta(t, m5fastener.NutSize.Thickness+0.4, m5fastener.NutCavityHeight, 1e-12)
	require.InDelta(t, corner+0.05, m5fastener.PressPocketRadius, 1e-4)
}
