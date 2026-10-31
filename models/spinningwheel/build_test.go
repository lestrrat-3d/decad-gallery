package spinningwheel_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// Start with the real producer and every real final solid before adding variants.
func TestSpinningWheelDefault(t *testing.T) {
	checkSpinningWheel(t, spinningwheel.DefaultParams())
}

func TestSpinningWheelVariants(t *testing.T) {
	for _, test := range []struct {
		name                        string
		scale                       float64
		ratio                       int
		tension, brake, feed, angle float64
	}{
		{"small-seven", .8, 7, 0, 0, 0, math.Pi},
		{"large-nine", 1.2, 9, 12, 6, 200, -math.Pi / 2},
		{"twelve", 1, 12, 0, 6, 0, math.Pi / 7},
		{"small-seventeen", .8, 17, 12, 6, 200, math.Pi / 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := spinningwheel.DefaultParams()
			p.Scale, p.Ratio, p.DriveTension, p.BrakeTravel, p.YarnFeedPerTurn, p.WheelAngle = test.scale, test.ratio, test.tension, test.brake, test.feed, test.angle
			checkSpinningWheel(t, p)
		})
	}
}

func checkSpinningWheel(t *testing.T, p spinningwheel.Params) {
	t.Helper()
	doc := decad.New()
	a, err := spinningwheel.Build(t.Context(), doc, p)
	if err != nil {
		bodies := doc.Bodies()
		for i := max(0, len(bodies)-4); i < len(bodies); i++ {
			box, boundErr := bodies[i].Bounds()
			if boundErr == nil {
				t.Logf("live body %d: %d faces, bounds (%g,%g,%g)..(%g,%g,%g)", i, len(bodies[i].Faces()), box.Min.X, box.Min.Y, box.Min.Z, box.Max.X, box.Max.Y, box.Max.Z)
			}
		}
	}
	require.NoError(t, err)
	requireOnePieceWhorl(t, a, p)
	require.Equal(t, len(a.Parts), len(doc.Bodies()), "only final assembly bodies may remain live")
	names := make(map[string]struct{})
	meshes := make(map[string]surfaceTriangles)
	for _, p := range a.Parts {
		_, exists := names[p.Name]
		require.False(t, exists, "duplicate name %s", p.Name)
		names[p.Name] = struct{}{}
		require.True(t, p.Body.IsSolid(), "%s", p.Name)
		chord := .12
		if p.Motion == "gimbal" || p.Motion == "footman" {
			chord = .01
		}
		if p.Name == "flyer-drive-whorl" {
			chord = .01
		}
		mesh, err := p.Body.Tessellate(t.Context(), units.Millimeters(chord), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, "part %s", p.Name)
		require.True(t, mesh.BoundaryVerified(), "part %s boundary", p.Name)
		require.True(t, mesh.VolumeVerified(), "part %s volume", p.Name)
		meshes[p.Name] = surfaceTriangles{vertices: mesh.Vertices(), triangles: mesh.Triangles(), bound: mesh.Bound().Base()}
	}
	for i, s := range a.Sketches {
		r := s.Verify(t.Context())
		require.True(t, r.Status == sketch.FullyConstrained, "sketch %d status %s", i, r.Status)
		require.Zero(t, r.DOF, "sketch %d", i)
		require.Zero(t, len(r.Redundant), "sketch %d", i)
		require.Zero(t, len(r.Conflicts), "sketch %d", i)
		require.NoError(t, r.Check(), "sketch %d", i)
	}
	requireWheelFeatures(t, a, p, meshes)
	requireHeadAdjustment(t, a, p)
	requireLinkageFrameClearances(t, a, p, meshes)
	requireFlyerSweptClearances(t, a)
	requireDriveBandFrameClearances(t, a)
	coils := 0
	for _, op := range doc.OperationHistory().Operations() {
		if op.Kind() == decad.OperationCoil {
			coils++
		}
	}
	require.Equal(t, 2, coils, "both brake springs must come from native Coil")
	t.Logf("verified %d final bodies and %d sketches", len(a.Parts), len(a.Sketches))
}
