package cubepuzzle_test

// CI: isolated-tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/cubepuzzle"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestPrintedPartCorners(t *testing.T) {
	doc := cubepuzzle.NewDocument()
	a, err := cubepuzzle.Build(t.Context(), doc, cubepuzzle.DefaultParams())
	require.NoError(t, err)
	for _, part := range []struct {
		name string
		body *decad.Body
		axis r3.Vec
	}{
		{"center", a.Centers[4], r3.Vec{Z: -1}},
		{"edge", a.Edges[3], r3.Vec{X: -1}},
		{"corner", a.Corners[7], r3.Vec{Z: -1}},
		{"disk", a.Disks[4], r3.Vec{Z: 1}},
	} {
		findings, err := cornercheck.Check(t.Context(), part.body, part.axis, nil)
		require.NoError(t, err, "%s", part.name)
		require.Zero(t, len(findings), "%s: %s", part.name, cornercheck.Report(findings, 8))
	}
	var exemptions []cornercheck.Exempt
	for _, z := range []float64{-9.5, 9.5} {
		for i := range 6 {
			theta := float64(i) * math.Pi / 3
			point := r3.Vec{X: m2fastener.NutCavityRadius * math.Cos(theta), Y: m2fastener.NutCavityRadius * math.Sin(theta), Z: z}
			exemptions = append(exemptions, cornercheck.Exempt{
				Reason: fmt.Sprintf("nut pocket corner %d at z=%g", i, z),
				Near: func(p r3.Vec) bool {
					return p.Sub(point).Len() < .1
				},
			})
		}
	}
	findings, err := cornercheck.Check(t.Context(), a.Hub, r3.Vec{Z: 1}, exemptions)
	require.NoError(t, err)
	require.Zero(t, len(findings), "hub: %s", cornercheck.Report(findings, 8))
}
