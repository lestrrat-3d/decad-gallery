package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"slices"
	"testing"

	"github.com/lestrrat-3d/solidlens"
	"github.com/stretchr/testify/require"
)

func TestGalleryCacheRealSpurGear(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "render")
	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", exe, ".")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	info, err := buildinfo.ReadFile(exe)
	require.NoError(t, err)
	cache, err := discoverGalleryCacheFor(t.Context(), exe, info)
	require.NoError(t, err)
	require.NotEmpty(t, cache.keys["spurgear"])
	cache.dir = filepath.Join(dir, "cache")
	count := 0
	tile := galleryTile{name: "spurgear", cell: image.Rect(0, 0, 560, 900),
		scene: func(ctx context.Context) (solidlens.Scene, error) {
			count++
			return spurGearHero(ctx)
		}}
	cold := filepath.Join(dir, "cold.png")
	warm := filepath.Join(dir, "warm.png")
	require.NoError(t, renderGalleryWithCache(t.Context(), cold, 1600, 900, []galleryTile{tile}, cache))
	require.Equal(t, 1, count)
	require.NoError(t, renderGalleryWithCache(t.Context(), warm, 1600, 900, []galleryTile{tile}, cache))
	require.Equal(t, 1, count, "warm render should skip the real scene builder")
	coldPNG, err := os.ReadFile(cold)
	require.NoError(t, err)
	warmPNG, err := os.ReadFile(warm)
	require.NoError(t, err)
	require.True(t, bytes.Equal(coldPNG, warmPNG), "cold and warm gallery output differs")

	file := cache.filename(tile.name, 1600, 900, tile.cell)
	require.NoError(t, os.WriteFile(file, []byte("invalid"), 0o644))
	require.NoError(t, renderGalleryWithCache(t.Context(), warm, 1600, 900, []galleryTile{tile}, cache))
	require.Equal(t, 2, count, "invalid entry should rebuild the scene")
	cache.fresh = true
	require.NoError(t, renderGalleryWithCache(t.Context(), warm, 1600, 900, []galleryTile{tile}, cache))
	require.Equal(t, 3, count, "fresh render should rebuild the scene")
}

func TestGalleryCacheIdentity(t *testing.T) {
	cache := galleryCache{dir: t.TempDir(), keys: map[string]string{"model": "key"}}
	cell := image.Rect(0, 0, 260, 300)
	first := cache.filename("model", 1600, 900, cell)
	require.NotEqual(t, first, cache.filename("model", 3200, 1800, cell))
	require.NotEqual(t, first, cache.filename("model", 1600, 900, cell.Add(image.Pt(1, 0))))
	cache.keys["model"] = "other key"
	require.NotEqual(t, first, cache.filename("model", 1600, 900, cell))
	require.Empty(t, cache.filename("unknown", 1600, 900, cell))
}

func TestGalleryCacheDiscoversRunningRenderer(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "render")
	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", exe, ".")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	info, err := buildinfo.ReadFile(exe)
	require.NoError(t, err)
	cache, err := discoverGalleryCacheFor(t.Context(), exe, info)
	require.NoError(t, err)
	require.NotEmpty(t, cache.keys["spurgear"])
	require.NotEmpty(t, cache.keys["bevelgear"])

	_, err = discoverGalleryCacheFor(t.Context(), os.Args[0], info)
	require.ErrorContains(t, err, "does not match current source")
}

func TestGalleryHashPackageChangesWithSource(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "source.go")
	require.NoError(t, os.WriteFile(name, []byte("original"), 0o644))
	pkg := galleryPackage{ImportPath: "example/model", Dir: dir, GoFiles: []string{"source.go"}}
	first := sha256.New()
	require.NoError(t, galleryHashPackage(first, pkg))
	require.NoError(t, os.WriteFile(name, []byte("changed"), 0o644))
	second := sha256.New()
	require.NoError(t, galleryHashPackage(second, pkg))
	require.NotEqual(t, first.Sum(nil), second.Sum(nil))
}

func TestGalleryCacheInvalidatesTransitiveSource(t *testing.T) {
	packages, err := galleryPackages(t.Context())
	require.NoError(t, err)
	renderer := packages[galleryRendererPackage]
	const dependency = "github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	require.False(t, slices.Contains(renderer.Imports, dependency), "dependency must be transitive")
	require.True(t, slices.Contains(renderer.Deps, dependency), "dependency must appear in the real graph")
	pkg := packages[dependency]
	require.Len(t, pkg.GoFiles, 1)
	contents, err := os.ReadFile(filepath.Join(pkg.Dir, pkg.GoFiles[0]))
	require.NoError(t, err)
	pkg.Dir = t.TempDir()
	file := filepath.Join(pkg.Dir, pkg.GoFiles[0])
	require.NoError(t, os.WriteFile(file, contents, 0o644))
	packages[dependency] = pkg
	info, ok := debug.ReadBuildInfo()
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(renderer.Dir, "../.."))
	before, err := galleryFingerprint(packages, renderer, root, info)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(file, append(contents, []byte("\n// changed\n")...), 0o644))
	after, err := galleryFingerprint(packages, renderer, root, info)
	require.NoError(t, err)
	require.NotEqual(t, before, after, "transitive source changes must invalidate cached tiles")
}
