package turbinefan_test

import (
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// poseTransform returns the transform of PrintPoseFor(sz, pitch, hand).
func poseTransform(t *testing.T, sz turbinefan.Size, pitch float64, hand turbinefan.Hand) (r3.Frame, r3.Transform) {
	t.Helper()
	f, err := turbinefan.PrintPoseFor(sz, pitch, hand)
	require.NoError(t, err)
	xf, err := r3.FromFrame(f)
	require.NoError(t, err)
	return f, xf
}

// posedBlade builds the blade of hand at rotor d and pitch in a fresh NewDocument, moves it into the print pose and
// returns the document and the posed blade.
func posedBlade(t *testing.T, d, pitch float64, hand turbinefan.Hand) (*decad.Document, *decad.Body) {
	t.Helper()
	sz := mustSize(t, d)
	doc := turbinefan.NewDocument()
	build := turbinefan.BuildBlade
	if hand == turbinefan.LeftHand {
		build = turbinefan.BuildBladeMirrored
	}
	blade, err := build(t.Context(), doc, pitch, sz)
	require.NoError(t, err)
	_, xf := poseTransform(t, sz, pitch, hand)
	posed, err := decad.Transform(t.Context(), blade.Body, xf)
	require.NoError(t, err)
	return doc, posed
}

// requireBoxNear requires two boxes to agree within tol on every face; mirror negates y of want first.
func requireBoxNear(t *testing.T, want, got decad.Box, tol float64, mirror bool, name string) {
	t.Helper()
	lo, hi := want.Min, want.Max
	if mirror {
		lo.Y, hi.Y = -want.Max.Y, -want.Min.Y
	}
	for i, pair := range [][2]float64{
		{lo.X, got.Min.X}, {lo.Y, got.Min.Y}, {lo.Z, got.Min.Z}, {hi.X, got.Max.X}, {hi.Y, got.Max.Y}, {hi.Z, got.Max.Z},
	} {
		require.InDelta(t, pair[0], pair[1], tol, "%s bound %d", name, i)
	}
}

// cradleCase is one harness case: rotor, pitch in degrees and hand.
type cradleCase struct {
	name     string
	rotor    float64
	pitchDeg float64
	hand     turbinefan.Hand
}

// cradleCases are the harness cases. The left hand at 192 / 25 pairs with the right hand for the mirror check.
var cradleCases = []cradleCase{
	{"128", 128, 25, turbinefan.RightHand},
	{"192", 192, 25, turbinefan.RightHand},
	{"192-left", 192, 25, turbinefan.LeftHand},
	{"192-steep-left", 192, 40, turbinefan.LeftHand},
	{"224-left", 224, 25, turbinefan.LeftHand},
	{"224-steep-left", 224, 40, turbinefan.LeftHand},
}

// builtCradle is one built harness case: its document, the posed blade and the harness.
type builtCradle struct {
	doc     *decad.Document
	blade   *decad.Body
	harness *turbinefan.BladeHarness
	sz      turbinefan.Size
	err     error
}

var (
	builtCradleMu    sync.Mutex
	builtCradleCases = map[string]*builtCradle{}
)

// cradleFor builds a harness case on first use and shares it between the tests, which only read it. A build takes 17 to
// 45 s.
func cradleFor(t *testing.T, c cradleCase) *builtCradle {
	t.Helper()
	builtCradleMu.Lock()
	defer builtCradleMu.Unlock()
	if b, ok := builtCradleCases[c.name]; ok {
		require.NoError(t, b.err, c.name)
		return b
	}
	sz := mustSize(t, c.rotor)
	doc, blade := posedBlade(t, c.rotor, c.pitchDeg*math.Pi/180, c.hand)
	h, err := turbinefan.BuildBladeHarness(t.Context(), doc, blade, sz, c.hand)
	b := &builtCradle{doc: doc, blade: blade, harness: h, sz: sz, err: err}
	builtCradleCases[c.name] = b
	require.NoError(t, err, c.name)
	return b
}

// eachCradle runs fn as a subtest for each harness case.
func eachCradle(t *testing.T, fn func(t *testing.T, c cradleCase, b *builtCradle)) {
	t.Helper()
	for _, c := range cradleCases {
		t.Run(c.name, func(t *testing.T) { fn(t, c, cradleFor(t, c)) })
	}
}

// shadowSectionCount returns the number of shadow sections: the Sections from the step zone's lead-in on.
func shadowSectionCount(h *turbinefan.BladeHarness) int {
	n := 0
	for _, c := range h.Sections {
		if c.X >= h.StepZone[0]-1.0-1e-9 {
			n++
		}
	}
	return n
}
