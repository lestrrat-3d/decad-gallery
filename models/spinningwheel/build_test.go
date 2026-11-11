package spinningwheel_test

import (
	"fmt"
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
	checkSpinningWheel(t, spinningwheel.DefaultParams(), true)
}

func TestSpinningWheelVariants(t *testing.T) {
	// requireLinkageFrameClearances poses the linkage at 72 angles and costs 150 to 290 s per
	// build, so it runs only where the linkage geometry differs. The linkage depends on Scale and
	// the brake travel, not on the ratio. The default build covers scale 1, small-seven covers scale
	// 0.8 and large-nine covers scale 1.2. TestSpinningWheelConnectorPoses and
	// TestSpinningWheelConnectorScales do not cover this: they check only the rebuilt connector
	// solid and its Sketches, never its clearance against the frame. Twelve and small-seventeen
	// repeat a scale already checked, so they skip it.
	for _, test := range []struct {
		name                        string
		linkage                     bool
		scale                       float64
		ratio                       int
		tension, brake, feed, angle float64
	}{
		{"small-seven", true, .8, 7, 0, 0, 0, math.Pi},
		{"large-nine", true, 1.2, 9, 12, 6, 200, -math.Pi / 2},
		{"twelve", false, 1, 12, 0, 6, 0, math.Pi / 7},
		{"small-seventeen", false, .8, 17, 12, 6, 200, math.Pi / 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := spinningwheel.DefaultParams()
			p.Scale, p.Ratio, p.DriveTension, p.BrakeTravel, p.YarnFeedPerTurn, p.WheelAngle = test.scale, test.ratio, test.tension, test.brake, test.feed, test.angle
			checkSpinningWheel(t, p, test.linkage)
		})
	}
}

// checkSpinningWheel runs every wheel check. With linkage it runs two subtests, each with its own Build,
// so CI can give them separate jobs: "verify" runs everything except the linkage frame clearances, and
// "linkage" runs only requireLinkageFrameClearances. Without linkage it runs the verify checks inline.
func checkSpinningWheel(t *testing.T, p spinningwheel.Params, linkage bool) {
	t.Helper()
	if !linkage {
		verifySpinningWheel(t, p)
		return
	}
	t.Run("verify", func(t *testing.T) { verifySpinningWheel(t, p) })
	t.Run("linkage", func(t *testing.T) { linkageSpinningWheel(t, p) })
}

func buildSpinningWheel(t *testing.T, p spinningwheel.Params) (*decad.Document, *spinningwheel.Assembly) {
	t.Helper()
	doc := spinningwheel.NewDocument()
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
	return doc, a
}

// tessellateParts meshes every final part with full verification and returns the meshes by part name.
func tessellateParts(t *testing.T, a *spinningwheel.Assembly) map[string]surfaceTriangles {
	t.Helper()
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
		require.Equal(t, 1, meshComponents(len(mesh.Vertices()), mesh.Triangles()), "%s must be connected", p.Name)
		meshes[p.Name] = surfaceTriangles{vertices: mesh.Vertices(), triangles: mesh.Triangles(), bound: mesh.Bound().Base()}
	}
	return meshes
}

func linkageSpinningWheel(t *testing.T, p spinningwheel.Params) {
	t.Helper()
	_, a := buildSpinningWheel(t, p)
	meshes := tessellateParts(t, a)
	requireLinkageFrameClearances(t, a, p, meshes)
}

func verifySpinningWheel(t *testing.T, p spinningwheel.Params) {
	t.Helper()
	doc, a := buildSpinningWheel(t, p)
	requirePartContainers(t, doc, a)
	requireOnePieceWhorl(t, a, p)
	requirePartHomes(t, a)
	require.Equal(t, len(a.Parts), len(doc.Bodies()), "only final assembly bodies may remain live")
	meshes := tessellateParts(t, a)
	for i, s := range a.Sketches {
		r := s.Verify(t.Context())
		require.True(t, r.Status == sketch.FullyConstrained, "sketch %d status %s", i, r.Status)
		require.Zero(t, r.DOF, "sketch %d", i)
		require.Zero(t, len(r.Redundant), "sketch %d", i)
		require.Zero(t, len(r.Conflicts), "sketch %d", i)
		require.NoError(t, r.Check(), "sketch %d", i)
	}
	requireBrakeCaptivity(t, a, p, meshes)
	// The part-home check reads only the meshes, so it runs while Verify works on the document.
	homes := make(chan partHomesResult, 1)
	go func() { homes <- geometricPartHomes(a, meshes) }()
	requireNoMaterialOverlap(t, doc, a)
	requirePartHomesResult(t, <-homes)
	requireWheelFeatures(t, a, p, meshes)
	requireHeadAdjustment(t, a, p)
	requireFlyerSweptClearances(t, a)
	requireDriveBandFrameClearances(t, a)
	requireBrakeSpringsFromOneCoil(t, doc, a)
	requireYarnHooksFromOneMaster(t, doc, a)
	if p == spinningwheel.DefaultParams() {
		requireBearingSeats(t, a)
		requireReferenceLayout(t, a)
		requireRearRailClearance(t, p, a)
	}
	t.Logf("verified %d final bodies and %d sketches", len(a.Parts), len(a.Sketches))
}

// requireBrakeSpringsFromOneCoil checks that the build runs one native Coil and that both brake springs
// come from it: the left spring through its Union and Refine, the right spring through a Copy of the left
// spring's refined body.
func requireBrakeSpringsFromOneCoil(t *testing.T, doc *decad.Document, a *spinningwheel.Assembly) {
	t.Helper()
	history := doc.OperationHistory()
	var coil *decad.Operation
	coils := 0
	for _, op := range history.Operations() {
		if op.Kind() != decad.OperationCoil {
			continue
		}
		coil = op
		coils++
	}
	require.Equal(t, 1, coils, "the left brake spring must be the only native Coil")
	for _, test := range []struct {
		side   string
		copied bool
	}{
		{"left", false},
		{"right", true},
	} {
		name := "brake-spring-" + test.side
		found, copied := brakeSpringAncestry(t, history, wheelPart(t, a, name))
		require.Equal(t, 1, len(found), "%s must come from exactly one Coil", name)
		_, ok := found[coil]
		require.True(t, ok, "%s must come from the build's native Coil", name)
		require.Equal(t, test.copied, copied, "%s copy step", name)
	}
}

// requireYarnHooksFromOneMaster checks that the first ordinary yarn hook and the active hook are built
// directly and each of the other ten ordinary hooks is a Copy of the first, and that exactly ten Copy
// operations take yarn-hook-1 as their source.
func requireYarnHooksFromOneMaster(t *testing.T, doc *decad.Document, a *spinningwheel.Assembly) {
	t.Helper()
	history := doc.OperationHistory()
	master := wheelPart(t, a, "yarn-hook-1")
	copies := 0
	for _, op := range history.Operations() {
		if op.Kind() == decad.OperationCopy && len(op.Inputs()) == 1 && op.Inputs()[0].Body == master {
			copies++
		}
	}
	require.Equal(t, 10, copies, "ten yarn hook copies")
	for i := 1; i <= 12; i++ {
		name := fmt.Sprintf("yarn-hook-%d", i)
		op, err := history.Producer(wheelPart(t, a, name))
		require.NoError(t, err, name)
		if i == 1 || i == 9 {
			require.NotEqual(t, decad.OperationCopy, op.Kind(), "%s is built directly", name)
			continue
		}
		require.Equal(t, decad.OperationCopy, op.Kind(), "%s must be a copy", name)
		require.Len(t, op.Inputs(), 1, name)
		require.Same(t, master, op.Inputs()[0].Body, "%s must copy yarn-hook-1", name)
	}
}

// brakeSpringAncestry follows body's producers back through every input. It returns the Coil operations it
// reaches and whether a Copy lies on the way.
func brakeSpringAncestry(t *testing.T, history *decad.OperationHistory, body *decad.Body) (map[*decad.Operation]struct{}, bool) {
	t.Helper()
	coils := make(map[*decad.Operation]struct{})
	copied := false
	seen := make(map[*decad.Body]struct{})
	pending := []*decad.Body{body}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}
		op, err := history.Producer(current)
		require.NoError(t, err, "every brake spring ancestor needs a recorded producer")
		if op.Kind() == decad.OperationCoil {
			coils[op] = struct{}{}
			continue
		}
		if op.Kind() == decad.OperationCopy {
			copied = true
		}
		for _, input := range op.Inputs() {
			pending = append(pending, input.Body)
		}
	}
	return coils, copied
}
