package sevensegment

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

// lightStepNames are the build steps the step tests run. They cover every pose (spin, lift of a lit slider and groove
// pin, the pivot blocks' toSegment, none), a step with copies, the M2 hole, pocket and hex pocket tools, edge
// finishing, and the bearing, bolt and nut builders. The heavy steps (walls, plates, rotor, flags) are left out.
var lightStepNames = []string{
	"pivot blocks", "guide plate", "shaft", "hub", "spring", "cam a", "cam d", "slider a", "groove pin a",
	"bearing 1", "bolt 1", "nut 1",
}

// lightSteps returns the light steps of the real BuildParts and Build step lists, in build order, and the model they
// file into.
func lightSteps() ([]buildStep, *Model) {
	m := &Model{}
	var steps []buildStep
	for _, st := range append(partSteps(&m.Parts), hardwareSteps(m)...) {
		if slices.Contains(lightStepNames, st.name) {
			steps = append(steps, st)
		}
	}
	return steps, m
}

// stepPrint is a body's identity for comparisons: its place in the document and a hash of its mesh and volume.
type stepPrint struct {
	index     int
	vertices  int
	triangles int
	volume    string
	meshHash  [sha256.Size]byte
}

// stepPrints returns one print per body of doc, in doc.Bodies() order.
func stepPrints(ctx context.Context, t *testing.T, doc *decad.Document) []stepPrint {
	t.Helper()
	var out []stepPrint
	for i, b := range doc.Bodies() {
		mesh, err := b.Tessellate(ctx, units.Millimeters(0.1))
		require.NoError(t, err, "body %d", i)
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
		vol, err := b.Volume()
		require.NoError(t, err, "body %d", i)
		p := stepPrint{index: i, vertices: len(vertices), triangles: len(triangles), volume: fmt.Sprintf("%v", vol)}
		copy(p.meshHash[:], h.Sum(nil))
		out = append(out, p)
	}
	return out
}

// firstPrintDiff names the first difference between two print lists, or returns "". It never formats the prints.
func firstPrintDiff(want, got []stepPrint) string {
	if len(want) != len(got) {
		return fmt.Sprintf("body count %d, want %d", len(got), len(want))
	}
	for i := range want {
		switch {
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
