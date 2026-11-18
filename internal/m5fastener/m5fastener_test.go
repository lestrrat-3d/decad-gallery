package m5fastener_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

const blockSize = 30.0

func site() m5fastener.Site {
	return m5fastener.Site{Dir: r3.Vec{Z: 1}, Corner: r3.Vec{X: 1}, Length: 16}
}

func frameAt(t *testing.T, z float64) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	return f
}

func bounds(t *testing.T, b *decad.Body) decad.Box {
	t.Helper()
	box, err := b.Bounds()
	require.NoError(t, err)
	return box
}

func volume(t *testing.T, b *decad.Body) float64 {
	t.Helper()
	v, err := b.Volume()
	require.NoError(t, err)
	return v.Value.Base()
}

// block builds a blockSize cube spanning -15..15 in X and Y and 0..blockSize in Z.
func block(t *testing.T, doc *decad.Document) *decad.Body {
	t.Helper()
	world := sketch.NewWorld()
	plane, err := world.CreatePlaneFromFrame(frameAt(t, 0))
	require.NoError(t, err)
	s, err := world.CreateSketch(plane)
	require.NoError(t, err)
	h := blockSize / 2
	r := s.CreateRectangle(-h, -h, h, h)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), r.A, -h),
		sketch.NewVerticalDistance(s.Origin(), r.A, -h),
		sketch.NewHorizontalDistance(r.A, r.B, blockSize),
		sketch.NewVerticalDistance(r.B, r.C, blockSize),
	)
	_, err = s.Solve(t.Context())
	require.NoError(t, err)
	var chosen *sketch.Profile
	for _, p := range s.Profiles() {
		if p.Valid && len(p.Holes) == 0 {
			chosen = p
		}
	}
	require.NotNil(t, chosen)
	body, err := decad.Extrude(t.Context(), doc, s, chosen,
		decad.Distance{D: units.Millimeters(blockSize), Dir: decad.Along})
	require.NoError(t, err)
	return body
}

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
		build  tool
	}{
		{"head", m5fastener.HeadPocketRadius, m5fastener.HeadPocketDepth,
			func(d *decad.Document, f r3.Frame) (*decad.Body, *sketch.Sketch, error) {
				return m5fastener.HeadPocketTool(ctx, d, f)
			}},
		{"nut", m5fastener.NutPocketRadius, m5fastener.NutCavityHeight,
			func(d *decad.Document, f r3.Frame) (*decad.Body, *sketch.Sketch, error) {
				return m5fastener.NutPocketTool(ctx, d, f, m5fastener.NutCavityHeight)
			}},
		{"press", m5fastener.PressPocketRadius, m5fastener.HeadPocketDepth,
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
			cut, err := decad.Cut(ctx, host, cutter)
			require.NoError(t, err)
			require.Greater(t, volume(t, cut), 0.0)
			removed := before - volume(t, cut)
			prism := 3 * math.Sqrt(3) / 2 * tc.radius * tc.radius * tc.depth
			// The floor round takes a ring off the prism's lower edge: less than a 0.6 mm square section round the
			// hexagon's perimeter (6*radius), more than a tenth of that.
			ring := 6 * tc.radius * m5fastener.PocketRound * m5fastener.PocketRound
			require.Less(t, removed, prism-ring/10, "floor is rounded")
			require.Greater(t, removed, prism-ring, "floor round is bounded")
		})
	}
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
