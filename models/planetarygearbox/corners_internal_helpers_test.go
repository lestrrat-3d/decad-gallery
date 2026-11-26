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

func requireCornerKit(t *testing.T, name string) {
	t.Helper()
	p, err := DefaultParamsFor(name)
	require.NoError(t, err)
	requireCornerVariant(t, p)
}

func requireCornerEars(t *testing.T, count int) {
	t.Helper()
	p := DefaultParams()
	p.EarCount = count
	// Only the corner check uses this variant. Leave the paired shaft fixture in its slot.
	requireCorners(t, buildCornerSplitVariant(t, p), p)
}

// Corners and split shafts inspect the same assembly without changing its bodies.
// Keep just the most recent unplaced variant so a full package run does not
// retain every gearbox. Placed variants always get a fresh build.
var lastCornerSplitVariant struct {
	mu     sync.Mutex
	params Params
	box    Gearbox
}

func cornerSplitVariant(t *testing.T, p Params) Gearbox {
	t.Helper()
	if p.Placement != nil {
		return buildCornerSplitVariant(t, p)
	}
	lastCornerSplitVariant.mu.Lock()
	defer lastCornerSplitVariant.mu.Unlock()
	if lastCornerSplitVariant.box.Ring != nil && lastCornerSplitVariant.params == p {
		return lastCornerSplitVariant.box
	}
	lastCornerSplitVariant.box = Gearbox{}
	lastCornerSplitVariant.box = buildCornerSplitVariant(t, p)
	lastCornerSplitVariant.params = p
	return lastCornerSplitVariant.box
}

func buildCornerSplitVariant(t *testing.T, p Params) Gearbox {
	t.Helper()
	box, err := Build(t.Context(), NewDocument(), p)
	require.NoError(t, err)
	return box
}

func requireCornerVariant(t *testing.T, p Params) {
	t.Helper()
	requireCorners(t, cornerSplitVariant(t, p), p)
}
