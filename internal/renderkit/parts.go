package renderkit

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/solidlens"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// TransformedMesh applies a rigid transform when its vertices are read.
type TransformedMesh struct {
	solidlens.TriangleSource
	Transform r3.Transform
}

func (m TransformedMesh) Vertices() []r3.Vec {
	source := m.TriangleSource.Vertices()
	out := make([]r3.Vec, len(source))
	for i, v := range source {
		out[i] = m.Transform.Apply(v)
	}
	return out
}

// MeshBounds returns the bounds of the vertices in a mesh.
func MeshBounds(mesh *decad.Mesh) (low, high r3.Vec) {
	for i, v := range mesh.Vertices() {
		if i == 0 {
			low, high = v, v
			continue
		}
		low = r3.NewVec(math.Min(low.X, v.X), math.Min(low.Y, v.Y), math.Min(low.Z, v.Z))
		high = r3.NewVec(math.Max(high.X, v.X), math.Max(high.Y, v.Y), math.Max(high.Z, v.Z))
	}
	return low, high
}

// ClipRemovesAll reports whether a clip plane removes every mesh vertex.
func ClipRemovesAll(mesh *decad.Mesh, clip solidlens.ClipPlane) bool {
	low, _ := MeshBounds(mesh)
	switch {
	case clip.Normal.Z == 1:
		return low.Z >= clip.Origin.Z
	case clip.Normal.X == 1:
		return low.X >= clip.Origin.X
	}
	// Any other normal: every vertex lies on the removed side.
	n, o := r3.Vec(clip.Normal), r3.Vec(clip.Origin)
	for _, v := range mesh.Vertices() {
		if v.Sub(o).Dot(n) < 0 {
			return false
		}
	}
	return true
}

// PartModel draws a mesh with the gallery part edge style.
func PartModel(mesh solidlens.TriangleSource, color solidlens.Color) solidlens.Model {
	return solidlens.Model{Mesh: mesh, Material: solidlens.Matte(color),
		Edges: solidlens.Edges{Enabled: true, Color: solidlens.RGB(.04, .06, .08), Width: 1.2}}
}

// Caption draws text at the given baseline.
func Caption(canvas draw.Image, face font.Face, text string, x, y int) {
	drawer := font.Drawer{Dst: canvas, Src: image.NewUniform(color.RGBA{R: 26, G: 38, B: 46, A: 255}), Face: face,
		Dot: fixed.P(x, y)}
	drawer.DrawString(text)
}
