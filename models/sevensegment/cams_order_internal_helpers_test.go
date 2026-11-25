package sevensegment

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// camPrint is the shape of a cam's mesh and solid that a different order of lettering loops changes.
type camPrint struct {
	Vertices, Triangles, Faces int
	VolumeBits                 uint64
	// Mesh hashes the vertices and triangles in order: it changes when the same shape is triangulated differently.
	Mesh [sha256.Size]byte
}

// meshHash hashes the vertex coordinates and the triangle indices of mesh in order.
func meshHash(mesh *decad.Mesh) [sha256.Size]byte {
	h := sha256.New()
	var b [8]byte
	for _, v := range mesh.Vertices() {
		for _, x := range []float64{v.X, v.Y, v.Z} {
			binary.LittleEndian.PutUint64(b[:], math.Float64bits(x))
			h.Write(b[:])
		}
	}
	for _, tri := range mesh.Triangles() {
		for _, i := range tri {
			binary.LittleEndian.PutUint64(b[:], uint64(i))
			h.Write(b[:])
		}
	}
	return [sha256.Size]byte(h.Sum(nil))
}

// freshCamPrint builds segment s's cam in a fresh run and returns its camPrint.
func freshCamPrint(t *testing.T, s Segment) camPrint {
	t.Helper()
	body, err := newTestRun(t).cam(s)
	require.NoError(t, err)
	var mesh *decad.Mesh
	for _, tol := range []float64{0.1, 0.25} {
		if mesh, err = body.Tessellate(t.Context(), units.Millimeters(tol)); err == nil {
			break
		}
	}
	require.NoError(t, err)
	vol, err := body.Volume()
	require.NoError(t, err)
	return camPrint{
		Vertices:   len(mesh.Vertices()),
		Triangles:  len(mesh.Triangles()),
		Faces:      len(body.Faces()),
		VolumeBits: math.Float64bits(vol.Value.Base()),
		Mesh:       meshHash(mesh),
	}
}
