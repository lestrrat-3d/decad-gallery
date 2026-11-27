package sevensegment

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
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
	return lightPosedSteps(nil)
}

// lightPosedSteps is lightSteps with the parts of extra poses filed into extras.
func lightPosedSteps(extras []*Parts) ([]buildStep, *Model) {
	m := &Model{}
	var steps []buildStep
	for _, st := range append(partSteps(&m.Parts, extras), hardwareSteps(m)...) {
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
		out = append(out, bodyPrint(ctx, t, i, b))
	}
	return out
}

// bodyPrint returns the print of b, which stands at index in its list.
func bodyPrint(ctx context.Context, t *testing.T, index int, b *decad.Body) stepPrint {
	t.Helper()
	mesh, err := b.Tessellate(ctx, units.Millimeters(0.1))
	require.NoError(t, err, "body %d", index)
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
	require.NoError(t, err, "body %d", index)
	p := stepPrint{index: index, vertices: len(vertices), triangles: len(triangles), volume: fmt.Sprintf("%v", vol)}
	copy(p.meshHash[:], h.Sum(nil))
	return p
}

// lightPartBodies returns the bodies of the light steps that a Parts holds, in a fixed order.
func lightPartBodies(p *Parts) []*decad.Body {
	var out []*decad.Body
	for _, seg := range Segments() {
		out = append(out, p.PivotBlocks[seg][0], p.PivotBlocks[seg][1])
	}
	return append(out, p.GuidePlate, p.Shaft, p.Hub, p.Spring, p.Cams[SegA], p.Cams[SegD], p.Sliders[SegA],
		p.GroovePins[SegA])
}

// lightPartPrints returns the prints of lightPartBodies of p.
func lightPartPrints(ctx context.Context, t *testing.T, p *Parts) []stepPrint {
	t.Helper()
	var out []stepPrint
	for i, b := range lightPartBodies(p) {
		require.NotNil(t, b, "body %d", i)
		out = append(out, bodyPrint(ctx, t, i, b))
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

// sketchShapes returns one line per sketch, in order: its point, entity and constraint counts. Equal lists mean the
// same sketches in the same order.
func sketchShapes(sketches []*sketch.Sketch) []string {
	out := make([]string, len(sketches))
	for i, s := range sketches {
		out[i] = fmt.Sprintf("%d/%d/%d", len(s.Points()), len(s.Entities()), len(s.Constraints()))
	}
	return out
}

// offsetFrame returns the world frame moved to origin.
func offsetFrame(t *testing.T, origin r3.Vec) *r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(origin, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	return &f
}

// printDiffs names every body whose print differs between two lists of equal length, with the first field that
// differs. It never formats the prints.
func printDiffs(want, got []stepPrint) []string {
	if len(want) != len(got) {
		return []string{fmt.Sprintf("body count %d, want %d", len(got), len(want))}
	}
	var out []string
	for i := range want {
		if d := firstPrintDiff(want[i:i+1], got[i:i+1]); d != "" {
			out = append(out, fmt.Sprintf("body %d: %s", i, d[strings.Index(d, ": ")+2:]))
		}
	}
	return out
}
