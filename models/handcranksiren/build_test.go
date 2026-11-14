package handcranksiren_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/stretchr/testify/require"
)

// partGroups lists, per top-level container, the parts the build has so far. Later tasks extend it.
var partGroups = map[string][]string{
	handcranksiren.HousingContainerName: {"gearbox-front", "gearbox-frame", "gearbox-rear", "drum", "collector"},
	handcranksiren.CrankContainerName:   {"crank-arm", "crank-pin", "crank-handle"},
	handcranksiren.StatorContainerName:  {"stator"},
	handcranksiren.RotorContainerName:   {"rotor"},
	handcranksiren.HornContainerName:    {"horn"},
	handcranksiren.GripContainerName:    {"grip"},
	handcranksiren.BoltsContainerName: {
		"gearbox-front-bolt-1", "gearbox-front-bolt-2", "gearbox-front-bolt-3", "gearbox-front-bolt-4",
		"gearbox-rear-bolt-1", "gearbox-rear-bolt-2", "gearbox-rear-bolt-3", "gearbox-rear-bolt-4",
		"drum-bolt-1", "drum-bolt-2", "drum-bolt-3", "collector-bolt-1", "collector-bolt-2", "collector-bolt-3", "collector-bolt-4",
		"throat-bolt-1", "throat-bolt-2", "throat-bolt-3", "throat-bolt-4",
		"grip-bolt-1", "grip-bolt-2", "grip-bolt-3", "grip-bolt-4", "crank-pin-bolt",
	},
	handcranksiren.NutsContainerName: {
		"gearbox-front-nut-1", "gearbox-front-nut-2", "gearbox-front-nut-3", "gearbox-front-nut-4",
		"gearbox-rear-nut-1", "gearbox-rear-nut-2", "gearbox-rear-nut-3", "gearbox-rear-nut-4",
		"drum-nut-1", "drum-nut-2", "drum-nut-3", "collector-nut-1", "collector-nut-2", "collector-nut-3", "collector-nut-4",
		"throat-nut-1", "throat-nut-2", "throat-nut-3", "throat-nut-4",
		"grip-nut-1", "grip-nut-2", "grip-nut-3", "grip-nut-4", "crank-pin-nut",
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
	require.Len(t, a.Parts, 73)
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
	t.Run("ports", func(t *testing.T) { requirePorts(t, p, parts) })
	t.Run("outlet", func(t *testing.T) { requireOutlet(t, parts) })
	t.Run("orientation", func(t *testing.T) { requireOrientation(t, p, parts) })
	t.Run("overlap", func(t *testing.T) { requireNoOverlap(t, doc, p, parts) })
	t.Run("split shafts", func(t *testing.T) { requireSplitShafts(t, a, parts) })
	if p != handcranksiren.DefaultParams() {
		// The variants prove only what Slots changes: the rotor's fits.
		t.Run("rotor fits", func(t *testing.T) {
			c := &fitCheck{t: t}
			requireRotorFits(t, p, parts, c)
			c.done()
		})
		return
	}
	t.Run("grip", func(t *testing.T) { requireGrip(t, parts) })
	t.Run("knuckle room", func(t *testing.T) { requireKnuckleRoom(t, parts) })
	t.Run("window height", func(t *testing.T) { requireWindowHeight(t, parts) })
	t.Run("sweep", func(t *testing.T) { requireSweep(t, parts) })
	t.Run("tessellate", func(t *testing.T) { requireTessellates(t, parts) })
	t.Run("edges", func(t *testing.T) { requireEdgeFinishing(t, p, parts) })
	t.Run("mesh", func(t *testing.T) {
		requireMesh(t, p, parts)
	})
	t.Run("clearances", func(t *testing.T) { requireClearances(t, p, parts) })
	t.Run("fasteners", func(t *testing.T) { requireFasteners(t, doc, p, parts) })
	t.Run("marks", func(t *testing.T) { requireMarkHoles(t, p, parts) })
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

// TestHandCrankSirenVariant builds the siren with the fewest and the most slots, at two crank angles.
func TestHandCrankSirenVariant(t *testing.T) {
	for name, p := range map[string]handcranksiren.Params{
		"slots 12": {CrankAngle: 0, Slots: 12},
		"slots 24": {CrankAngle: 5 * math.Pi / 4, Slots: 24},
	} {
		t.Run(name, func(t *testing.T) { checkSiren(t, p) })
	}
}
