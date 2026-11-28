package screwgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/stretchr/testify/require"
)

// shortParams keeps the ribbons to 16 teeth, the least the bores' range check admits, so
// a build takes about 15 s instead of 40 s.
func shortParams() screwgear.Params {
	p := screwgear.DefaultParams()
	p.ToothCount = 16
	return p
}

func build(t *testing.T, p screwgear.Params) (*screwgear.Mechanism, *decad.Document) {
	t.Helper()
	doc := screwgear.NewDocument()
	m, err := screwgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	checkBuild(t, doc, m)
	return m, doc
}

// checkBuild requires the three solid bodies of a finished build, in their containers.
func checkBuild(t *testing.T, doc *decad.Document, m *screwgear.Mechanism) {
	t.Helper()
	require.Len(t, doc.Bodies(), 3)
	for name, b := range map[string]*decad.Body{"gear A": m.GearA, "gear B": m.GearB, "cage": m.Cage} {
		require.True(t, b.IsSolid(), name)
	}
	requireContainers(t, doc, m)
}

// requireContainers checks that the gears sit in "Gears" and the sleeve in "Sleeve", and that
// no body is left in the root.
func requireContainers(t *testing.T, doc *decad.Document, m *screwgear.Mechanism) {
	t.Helper()
	var names []string
	for _, c := range doc.Root().Children() {
		names = append(names, c.Name())
	}
	require.ElementsMatch(t, []string{"Gears", "Sleeve"}, names)
	gears, err := doc.Root().ChildNamed("Gears")
	require.NoError(t, err)
	sleeve, err := doc.Root().ChildNamed("Sleeve")
	require.NoError(t, err)
	require.ElementsMatch(t, []*decad.Body{m.GearA, m.GearB}, gears.Bodies())
	require.Equal(t, []*decad.Body{m.Cage}, sleeve.Bodies())
	require.NoError(t, doc.VerifyContainers().Err())
}

// ribbonVolume integrates, without decad, the section area of a ribbon over its length.
// The twist turns the section rigidly and so leaves its area alone; the toothed edge
// u = W/2 - H/2 + H/2*cos(2*pi*(s + tan(slant)*v - phase)/P) - bow*v^2 sets it.
func ribbonVolume(p screwgear.Params, phase float64) float64 {
	const nv, ns = 200, 20000
	length := p.Length()
	total := 0.0
	for i := range ns {
		s := -length/2 + phase + (float64(i)+0.5)*length/ns
		area := 0.0
		for j := range nv {
			v := -p.Thickness/2 + (float64(j)+0.5)*p.Thickness/nv
			cell := (s + math.Tan(p.ToothSlant)*v - phase) / p.ToothPitch
			edge := p.Width/2 - p.ToothHeight/2 + p.ToothHeight/2*math.Cos(2*math.Pi*cell) - p.ToothBow*v*v
			area += (edge + p.Width/2) * p.Thickness / nv
		}
		total += area * length / ns
	}
	return total
}
