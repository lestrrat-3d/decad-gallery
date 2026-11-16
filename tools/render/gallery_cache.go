package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
)

const galleryRendererPackage = "github.com/lestrrat-3d/decad-gallery/tools/render"

type galleryPackage struct {
	ImportPath, Dir, BuildID, Export                                                                                    string
	Standard                                                                                                            bool
	Deps                                                                                                                []string
	GoFiles, CgoFiles, CFiles, CXXFiles, MFiles, HFiles, FFiles, SFiles, SwigFiles, SwigCXXFiles, SysoFiles, EmbedFiles []string
}

type galleryCache struct {
	dir   string
	keys  map[string]string
	fresh bool
}

// newGalleryCache disables caching if the source and dependency graph cannot be identified.
func newGalleryCache(ctx context.Context, fresh bool) galleryCache {
	cache, err := discoverGalleryCache(ctx)
	if err != nil {
		return galleryCache{}
	}
	cache.fresh = fresh
	return cache
}

func discoverGalleryCache(ctx context.Context) (galleryCache, error) {
	executable, err := os.Executable()
	if err != nil {
		return galleryCache{}, err
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return galleryCache{}, fmt.Errorf("renderer build settings are unavailable")
	}
	return discoverGalleryCacheFor(ctx, executable, info)
}

func discoverGalleryCacheFor(ctx context.Context, executable string, info *debug.BuildInfo) (galleryCache, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-deps", "-export", "-json", ".")
	data, err := cmd.Output()
	if err != nil {
		return galleryCache{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	packages := map[string]galleryPackage{}
	for {
		var pkg galleryPackage
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			return galleryCache{}, err
		}
		packages[pkg.ImportPath] = pkg
	}
	renderer, ok := packages[galleryRendererPackage]
	if !ok || renderer.Dir == "" || renderer.BuildID == "" {
		return galleryCache{}, fmt.Errorf("renderer build information is unavailable")
	}
	buildID, err := exec.CommandContext(ctx, "go", "tool", "buildid", executable).Output()
	if err != nil || !strings.Contains("/"+strings.TrimSpace(string(buildID))+"/", "/"+renderer.BuildID+"/") {
		return galleryCache{}, fmt.Errorf("renderer executable does not match current source")
	}
	root := filepath.Clean(filepath.Join(renderer.Dir, "../.."))
	base := sha256.New()
	_, _ = fmt.Fprintln(base, "gallery-cache-v1", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	for _, dep := range info.Deps {
		_, _ = fmt.Fprintln(base, dep)
	}
	for _, setting := range info.Settings {
		if !strings.HasPrefix(setting.Key, "vcs.") {
			_, _ = fmt.Fprintln(base, setting.Key, setting.Value)
		}
	}
	for _, file := range []string{"go.mod", "go.sum", "tools/render/go.mod", "tools/render/go.sum"} {
		if err := galleryHashFile(base, filepath.Join(root, file)); err != nil {
			return galleryCache{}, err
		}
	}
	if err := galleryHashPackage(base, renderer); err != nil {
		return galleryCache{}, err
	}
	// Any renderer function can call another model, so every dependency invalidates all tiles.
	for _, name := range renderer.Deps {
		pkg, ok := packages[name]
		if !ok {
			return galleryCache{}, fmt.Errorf("missing dependency %s", name)
		}
		if pkg.Standard {
			continue
		}
		if err := galleryHashPackage(base, pkg); err != nil {
			return galleryCache{}, err
		}
	}
	keys := make(map[string]string, len(galleryTiles))
	for _, tile := range galleryTiles {
		keys[tile.name] = hex.EncodeToString(base.Sum(nil))
	}
	return galleryCache{dir: filepath.Join(root, ".tmp", "gallery-cache"), keys: keys}, nil
}

func galleryHashPackage(h hash.Hash, pkg galleryPackage) error {
	_, _ = fmt.Fprintln(h, pkg.ImportPath)
	files := slices.Concat(pkg.GoFiles, pkg.CgoFiles, pkg.CFiles, pkg.CXXFiles, pkg.MFiles, pkg.HFiles, pkg.FFiles,
		pkg.SFiles, pkg.SwigFiles, pkg.SwigCXXFiles, pkg.SysoFiles, pkg.EmbedFiles)
	slices.Sort(files)
	for _, name := range files {
		if err := galleryHashFile(h, filepath.Join(pkg.Dir, name)); err != nil {
			return err
		}
	}
	return nil
}

func galleryHashFile(h hash.Hash, name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(h, name, stat.Size())
	_, err = io.Copy(h, f)
	return err
}

func (c galleryCache) filename(name string, width, height int, cell image.Rectangle) string {
	key, ok := c.keys[name]
	if !ok || c.dir == "" {
		return ""
	}
	h := sha256.New()
	_, _ = fmt.Fprintln(h, key, name, width, height, cell)
	return filepath.Join(c.dir, hex.EncodeToString(h.Sum(nil))+".png")
}

func (c galleryCache) read(name string, width, height int, cell image.Rectangle) (image.Image, color.Color, bool) {
	file := c.filename(name, width, height, cell)
	if file == "" || c.fresh {
		return nil, nil, false
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, nil, false
	}
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	if err != nil || img.Bounds().Empty() {
		return nil, nil, false
	}
	background := color.RGBAModel.Convert(img.At(0, 0))
	if foregroundBounds(img, background).Empty() {
		return nil, nil, false
	}
	return img, background, true
}

func (c galleryCache) write(name string, width, height int, cell image.Rectangle, img image.Image) {
	file := c.filename(name, width, height, cell)
	if file != "" {
		_ = writePNG(file, img)
	}
}
