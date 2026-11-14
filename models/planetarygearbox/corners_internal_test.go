package planetarygearbox

import (
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// isPurchased reports whether a body is a bolt, a nut or a bearing. Purchased parts keep their nominal size and are
// exempt from "Corners and roots".
func isPurchased(name string) bool {
	return strings.HasSuffix(name, "Bearing") || strings.Contains(name, " Bearing ") ||
		strings.Contains(name, " Bolt ") || strings.Contains(name, " Nut ")
}

// requireCorners applies "Corners and roots" to every printed body of g, built from p with a nil Placement, in its
// print orientation, axis +Z. Bolts, nuts and bearings are purchased parts. The exemptions match "Retained sharp
// corners" in instructions.md. The face levels are the runs of one outline, so cornercheck.Check's MidHeights apply.
func requireCorners(t *testing.T, g Gearbox, p Params) {
	t.Helper()
	d := derive(p)
	z := r3.Vec{Z: 1}
	exempt := map[string][]cornercheck.Exempt{
		// The ring's teeth: the tool that cuts the spaces has no root fillet, and the internal tips are working flanks.
		RingName: {{Reason: "ring teeth", Near: cornercheck.Annulus(r3.Vec{}, z, d.ringTip-0.2, d.ringRoot+0.2)}},
	}
	for k := range g.Stages {
		for i := range p.PlanetCount {
			a := PlanetOrbitAngle(p, i)
			c := r3.Vec{X: d.orbit * math.Cos(a), Y: d.orbit * math.Sin(a)}
			exempt[PlanetName(k, i)] = []cornercheck.Exempt{{
				Reason: "planet teeth", Near: cornercheck.Annulus(c, z, d.planetRoot-0.2, d.planetTip+0.2),
			}}
		}
	}
	for _, nb := range g.Named() {
		if isPurchased(nb.Name) {
			continue
		}
		t.Run(nb.Name, func(t *testing.T) {
			fs, err := cornercheck.Check(t.Context(), nb.Body, z, exempt[nb.Name])
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
}

// cornersDefault builds the default gearbox once for TestCorners.
var cornersDefault struct {
	once sync.Once
	box  Gearbox
	err  error
}

// TestCorners applies "Corners and roots" to every printed part of the default gearbox. The variants run in
// TestCornersVariants, which is too slow for pull request CI.
func TestCorners(t *testing.T) {
	cornersDefault.once.Do(func() {
		cornersDefault.box, cornersDefault.err = Build(t.Context(), NewDocument(), DefaultParams())
	})
	require.NoError(t, cornersDefault.err)
	requireCorners(t, cornersDefault.box, DefaultParams())
}

// TestCornersVariants applies "Corners and roots" to every parameter case that changes an outline: the other kits,
// every ear count the gallery tests, and two stages. It takes about 5 minutes, so it runs in the scheduled full suite
// and not in pull request CI. Each build takes under a minute; the 608ZZ kits' checks are the slow part of
// TestKitGearboxes, not their builds.
func TestCornersVariants(t *testing.T) {
	type variant struct {
		name  string
		build func(t *testing.T) Params
	}
	kit := func(name string) func(t *testing.T) Params {
		return func(t *testing.T) Params {
			p, err := DefaultParamsFor(name)
			require.NoError(t, err)
			return p
		}
	}
	ears := func(n int) func(t *testing.T) Params {
		return func(*testing.T) Params {
			p := DefaultParams()
			p.EarCount = n
			return p
		}
	}
	variants := []variant{
		{"planet605-shaft608", kit("planet605-shaft608")},
		{"planet608-shaft605", kit("planet608-shaft605")},
		{"planet608-shaft608", kit("planet608-shaft608")},
		{"EarCount 0", ears(0)},
		{"EarCount 1", ears(1)},
		{"EarCount 6", ears(6)},
		{"Stages 2", func(*testing.T) Params {
			p := DefaultParams()
			p.Stages = 2
			return p
		}},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			p := v.build(t)
			box, err := Build(t.Context(), NewDocument(), p)
			require.NoError(t, err)
			requireCorners(t, box, p)
		})
	}
}
