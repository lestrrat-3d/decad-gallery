package main

// CI: isolated-tests

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/renderkit"
	"github.com/stretchr/testify/require"
)

func TestRegistryNamesAreUniqueAndSorted(t *testing.T) {
	var names []string
	for _, m := range registry() {
		names = append(names, m.Name)
	}
	require.NotEmpty(t, names)
	require.Equal(t, "gallery", names[len(names)-1], "the gallery comes last")
	rest := names[:len(names)-1]
	require.True(t, slices.IsSorted(rest), "models are in alphabetical order: %v", rest)
	seen := map[string]struct{}{}
	for _, name := range names {
		_, dup := seen[name]
		require.False(t, dup, "model %q is registered twice", name)
		seen[name] = struct{}{}
	}
}

func TestRegistryDirectoriesExist(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, m := range registry() {
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(m.Dir)))
		require.NoError(t, err)
		require.Len(t, matches, 1, "directory %q of model %q does not exist", m.Dir, m.Name)
	}
}

func TestRegistryJobPathsAreDistinct(t *testing.T) {
	paths := map[string]string{}
	for _, m := range registry() {
		for _, j := range m.Jobs {
			path := renderkit.JobPath("root", m, j)
			other, dup := paths[path]
			require.False(t, dup, "jobs %s/%s and %s share the path %s", m.Name, j.Shot, other, path)
			paths[path] = m.Name + "/" + j.Shot
			_, isShot := m.Shots[j.Shot]
			_, isPreset := m.Presets[j.Shot]
			require.True(t, isShot || isPreset, "job shot %s/%s does not exist", m.Name, j.Shot)
		}
	}
	require.Len(t, paths, 34)
}

func TestRegistryOnlyPlanetaryAcceptsKit(t *testing.T) {
	for _, m := range registry() {
		require.Equal(t, m.Name == "planetarygearbox", m.AcceptsKit, "AcceptsKit of %s", m.Name)
	}
}
