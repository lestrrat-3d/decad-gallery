package sevensegment

// CI: isolated-tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSevenSegmentCamLetteringOrder builds every cam twice in one process. The lettering loops start at fixed
// corners, so both builds match each other mesh for mesh, and the pinned counts and volumes; ranging over a map would give each build its own
// triangulation of the engraving.
func TestSevenSegmentCamLetteringOrder(t *testing.T) {
	t.Parallel()
	want := map[Segment]camPrint{
		SegA: {Vertices: 8858, Triangles: 17732, Faces: 90, VolumeBits: 0x40d3a77f9f68d8bb},
		SegB: {Vertices: 8327, Triangles: 16662, Faces: 102, VolumeBits: 0x40d3c4ffe7e6a2a7},
		SegC: {Vertices: 8507, Triangles: 17026, Faces: 68, VolumeBits: 0x40d3bd255ecbf584},
		SegD: {Vertices: 8619, Triangles: 17242, Faces: 69, VolumeBits: 0x40d3ac908afd3e8f},
		SegE: {Vertices: 8333, Triangles: 16674, Faces: 55, VolumeBits: 0x40d3b1bff14a09ab},
		SegF: {Vertices: 8602, Triangles: 17216, Faces: 54, VolumeBits: 0x40d3a500cb09f096},
		SegG: {Vertices: 8634, Triangles: 17272, Faces: 68, VolumeBits: 0x40d3b7247b953f29},
	}
	for _, seg := range []Segment{SegA, SegB, SegC, SegD, SegE, SegF, SegG} {
		t.Run(seg.String(), func(t *testing.T) {
			first, second := freshCamPrint(t, seg), freshCamPrint(t, seg)
			require.Equal(t, first, second, "the same cam, built twice, has one mesh")
			first.Mesh = [len(first.Mesh)]byte{}
			require.Equal(t, want[seg], first)
		})
	}
}

// TestSevenSegmentLetterLoopsOrder checks that letterLoops lists the same loops from the same corners on every call.
func TestSevenSegmentLetterLoopsOrder(t *testing.T) {
	t.Parallel()
	for _, seg := range []Segment{SegA, SegB, SegC, SegD, SegE, SegF, SegG} {
		t.Run(seg.String(), func(t *testing.T) {
			want := letterLoops(seg)
			require.NotEmpty(t, want)
			for range 50 {
				require.Equal(t, want, letterLoops(seg))
			}
		})
	}
}
