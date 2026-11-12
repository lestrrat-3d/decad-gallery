package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSpinningWheelPartsTable checks that the README's parts table lists every distinct body once, with a part image
// that exists, and that the counts add up to the whole build.
func TestSpinningWheelPartsTable(t *testing.T) {
	meshes, _, err := spinningWheelModels(t.Context())
	require.NoError(t, err)
	kinds, err := wheelKinds(meshes)
	require.NoError(t, err)
	readme, err := os.ReadFile("../../models/spinningwheel/README.md")
	require.NoError(t, err)
	total := 0
	for _, kind := range kinds {
		total += len(kind.members)
		link := "images/parts/" + kind.slug + ".png"
		require.Equal(t, 1, strings.Count(string(readme), `<img src="`+link+`"`), "README row for %s", kind.slug)
		_, err := os.Stat("../../models/spinningwheel/" + link)
		require.NoError(t, err)
	}
	require.Equal(t, len(meshes), total)
}

func TestSpinningWheelBlockOfEveryPart(t *testing.T) {
	meshes, _, err := spinningWheelModels(t.Context())
	require.NoError(t, err)
	counts := map[string]int{}
	for _, m := range meshes {
		counts[wheelBlockOf(m.part.Name, m.part.Group, m.part.Motion).name]++
	}
	for _, block := range []wheelBlock{wheelBlockFrame, wheelBlockWheel, wheelBlockTreadle, wheelBlockLinkage,
		wheelBlockHead, wheelBlockFlyer, wheelBlockBrake, wheelBlockDrive} {
		require.NotZero(t, counts[block.name], "no part in block %s", block.name)
	}
}
