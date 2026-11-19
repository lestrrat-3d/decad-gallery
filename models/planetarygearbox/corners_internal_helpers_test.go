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
