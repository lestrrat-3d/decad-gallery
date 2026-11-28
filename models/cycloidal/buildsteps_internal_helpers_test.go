package cycloidal

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// stepDocument returns a document with the chain tolerance Build needs.
func stepDocument() *decad.Document {
	return decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))
}

// stepRun returns the run begin makes for the default drive in a new document.
func stepRun(t *testing.T) *run {
	t.Helper()
	r, err := begin(t.Context(), stepDocument(), DefaultParams())
	require.NoError(t, err)
	return r
}

// builtPrint identifies a finished body: its place in doc.Bodies(), its container, and a hash of its mesh.
type builtPrint struct {
	name      string
	index     int
	container string
	vertices  int
	triangles int
	volume    string
	meshHash  [sha256.Size]byte
}

// builtPrints returns one print per body of drive, in Drive.Named order.
func builtPrints(t *testing.T, doc *decad.Document, drive Drive) []builtPrint {
	t.Helper()
	var out []builtPrint
	for _, nb := range drive.Named() {
		mesh, err := nb.Body.Tessellate(t.Context(), units.Millimeters(0.25))
		require.NoError(t, err, nb.Name)
		h := sha256.New()
		var buf [8]byte
		vertices := mesh.Vertices()
		for _, v := range vertices {
			for _, f := range []float64{v.X, v.Y, v.Z} {
				binary.LittleEndian.PutUint64(buf[:], math.Float64bits(f))
				h.Write(buf[:])
			}
		}
		triangles := mesh.Triangles()
		for _, tri := range triangles {
			for _, k := range tri {
				binary.LittleEndian.PutUint64(buf[:], uint64(k))
				h.Write(buf[:])
			}
		}
		vol, err := nb.Body.Volume()
		require.NoError(t, err, nb.Name)
		p := builtPrint{
			name: nb.Name, index: slices.Index(doc.Bodies(), nb.Body), vertices: len(vertices),
			triangles: len(triangles), volume: fmt.Sprintf("%v", vol),
		}
		for _, c := range doc.Root().Children() {
			if slices.Contains(c.Bodies(), nb.Body) {
				p.container = c.Name()
			}
		}
		copy(p.meshHash[:], h.Sum(nil))
		out = append(out, p)
	}
	return out
}

// firstBuiltDiff names the first difference between two print lists, or returns "". It never formats the prints.
func firstBuiltDiff(want, got []builtPrint) string {
	if len(want) != len(got) {
		return fmt.Sprintf("body count %d, want %d", len(got), len(want))
	}
	for i := range want {
		switch {
		case want[i].name != got[i].name:
			return fmt.Sprintf("body %d: name", i)
		case want[i].index != got[i].index:
			return fmt.Sprintf("body %d: doc.Bodies() index %d, want %d", i, got[i].index, want[i].index)
		case want[i].container != got[i].container:
			return fmt.Sprintf("body %d: container %q, want %q", i, got[i].container, want[i].container)
		case want[i].vertices != got[i].vertices:
			return fmt.Sprintf("body %d: vertex count", i)
		case want[i].triangles != got[i].triangles:
			return fmt.Sprintf("body %d: triangle count", i)
		case want[i].volume != got[i].volume:
			return fmt.Sprintf("body %d: volume", i)
		case want[i].meshHash != got[i].meshHash:
			return fmt.Sprintf("body %d: mesh", i)
		}
	}
	return ""
}

// noStep is a step that makes nothing and files nothing.
func noStep(name string) buildStep {
	return buildStep{name, func(context.Context, *run) (func(), error) { return nil, nil }}
}
