package handcranksiren_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/stretchr/testify/require"
)

// partGroups lists, per top-level container, the parts the build has so far. Later tasks extend it.
var partGroups = map[string][]string{
	handcranksiren.HousingContainerName: {"gearbox-front", "gearbox-frame", "gearbox-rear"},
	handcranksiren.BoltsContainerName: {
		"gearbox-front-bolt-1", "gearbox-front-bolt-2", "gearbox-front-bolt-3", "gearbox-front-bolt-4",
		"gearbox-rear-bolt-1", "gearbox-rear-bolt-2", "gearbox-rear-bolt-3", "gearbox-rear-bolt-4",
	},
	handcranksiren.NutsContainerName: {
		"gearbox-front-nut-1", "gearbox-front-nut-2", "gearbox-front-nut-3", "gearbox-front-nut-4",
		"gearbox-rear-nut-1", "gearbox-rear-nut-2", "gearbox-rear-nut-3", "gearbox-rear-nut-4",
	},
	handcranksiren.ShaftsContainerName: {"crank-shaft", "intermediate-shaft", "rotor-shaft"},
	handcranksiren.BearingsContainerName: {
		"bearing-rotor-front", "bearing-intermediate-front", "bearing-crank-front",
		"bearing-rotor-rear", "bearing-intermediate-rear", "bearing-crank-rear",
	},
}

// gearPairs lists the parts of "Pair" (stage 1) and "Pair 2" (stage 2).
var gearPairs = [2][2]string{{"crank-gear", "intermediate-pinion"}, {"intermediate-gear", "rotor-pinion"}}

func partNames() []string {
	var names []string
	for _, group := range partGroups {
		names = append(names, group...)
	}
	for _, pair := range gearPairs {
		names = append(names, pair[:]...)
	}
	return names
}

func buildSiren(t *testing.T, p handcranksiren.Params) (*decad.Document, map[string]handcranksiren.Part, *handcranksiren.Assembly) {
	t.Helper()
	doc := handcranksiren.NewDocument()
	a, err := handcranksiren.Build(t.Context(), doc, p)
	require.NoError(t, err)
	parts := map[string]handcranksiren.Part{}
	for _, part := range a.Parts {
		_, dup := parts[part.Name]
		require.False(t, dup, "duplicate part %s", part.Name)
		parts[part.Name] = part
		require.True(t, part.Body.IsSolid(), part.Name)
	}
	return doc, parts, a
}

func bounds(t *testing.T, part handcranksiren.Part) decad.Box {
	t.Helper()
	box, err := part.Body.Bounds()
	require.NoError(t, err)
	return box
}

// checkSiren runs every check on one real build.
func checkSiren(t *testing.T, p handcranksiren.Params) {
	doc, parts, a := buildSiren(t, p)
	names := partNames()
	require.Len(t, a.Parts, len(names))
	for _, name := range names {
		require.Contains(t, parts, name)
	}
	require.Len(t, doc.Bodies(), len(names), "no tool or prototype body remains")

	t.Run("sketches", func(t *testing.T) {
		require.NotEmpty(t, a.Sketches)
		for i, s := range a.Sketches {
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
		}
	})
	t.Run("containers", func(t *testing.T) { requireContainers(t, doc, parts) })
	if p == handcranksiren.DefaultParams() {
		t.Run("mesh", func(t *testing.T) {
			requireSpurMesh(t, p, parts)
			requireTipToFrame(t, p, parts)
		})
		t.Run("clearances", func(t *testing.T) { requireClearances(t, p, parts) })
		t.Run("fasteners", func(t *testing.T) { requireFasteners(t, doc, p, parts) })
	}
}

// requireContainers checks that every body sits in the container for its kind and that the document verifies.
func requireContainers(t *testing.T, doc *decad.Document, parts map[string]handcranksiren.Part) {
	t.Helper()
	for group, names := range partGroups {
		for _, name := range names {
			c, err := doc.ContainerOf(parts[name].Body)
			require.NoError(t, err, name)
			require.Equal(t, group, c.Name(), name)
		}
	}
	root := doc.Root()
	pairs, err := root.ChildNamed(handcranksiren.GearPairsContainerName)
	require.NoError(t, err)
	for i, want := range []string{"Pair", "Pair 2"} {
		for _, name := range gearPairs[i] {
			c, err := doc.ContainerOf(parts[name].Body)
			require.NoError(t, err, name)
			require.Equal(t, want, c.Name(), name)
			require.Equal(t, pairs, c.Parent(), name)
		}
	}
	require.NoError(t, doc.VerifyContainers().Err())
}

func TestHandCrankSirenDefault(t *testing.T) { checkSiren(t, handcranksiren.DefaultParams()) }
