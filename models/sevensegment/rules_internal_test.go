package sevensegment

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentCorners applies "Corners and roots" to every printed part in its print orientation: only the listed
// exemptions are allowed, and the shaft is read in TestSevenSegmentSplitShaft.
func TestSevenSegmentCorners(t *testing.T) {
	t.Parallel()
	for _, c := range partCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.pending != "" {
				t.Skip(c.pending)
			}
			r, body := sharedBody(t, c.name, c.build)
			var exempt []cornercheck.Exempt
			if c.exempt != nil {
				exempt = c.exempt(r)
			}
			fs, err := cornercheck.Check(t.Context(), body, c.up, exempt)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 8))
		})
	}
}

// TestSevenSegmentEdgeFinishing checks "Surface edge finishing" on the built bodies. Each face loop that carries the
// 0.5 mm chamfer is read from two sections, 0.1 and 1.0 inside its face: the chamfer pulls the outline in by
// 0.5 - 0.1 = 0.4 at the first, on every side of a plate and all round a cam or the knob. Every other part has no
// planar face of 400 mm2 or more.
func TestSevenSegmentEdgeFinishing(t *testing.T) {
	t.Parallel()
	specs := chamferSpecs()
	for _, c := range partCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.pending != "" {
				t.Skip(c.pending)
			}
			r, body := sharedBody(t, c.name, c.build)
			spec, ok := specs[c.name]
			require.Equal(t, c.chamfered, ok, "the part's chamfer flag and its chamfer spec agree")
			if !ok {
				require.Zero(t, largePlanes(t, body), "no face of 400 mm2 or more needs a chamfer")
				return
			}
			faces := spec.faces(r, body)
			require.NotEmpty(t, faces, "the large faces carry a chamfer")
			for _, f := range faces {
				nMinX, nMaxX, nMinY, nMaxY, nR := outlineReach(spec.section(t, body, f.level+f.inward*0.1))
				dMinX, dMaxX, dMinY, dMaxY, dR := outlineReach(spec.section(t, body, f.level+f.inward*1.0))
				const want = 0.5 - 0.1
				if spec.round {
					require.InDelta(t, want, dR-nR, 0.05, "the rim at face %.2f is chamfered all round", f.level)
					continue
				}
				require.InDelta(t, want, nMinX-dMinX, 0.05, "the -x side at face %.2f is chamfered", f.level)
				require.InDelta(t, want, dMaxX-nMaxX, 0.05, "the +x side at face %.2f is chamfered", f.level)
				if spec.xSides {
					continue
				}
				require.InDelta(t, want, nMinY-dMinY, 0.05, "the -y side at face %.2f is chamfered", f.level)
				require.InDelta(t, want, dMaxY-nMaxY, 0.05, "the +y side at face %.2f is chamfered", f.level)
			}
		})
	}
}

// TestSevenSegmentLookAlike checks every look-alike set: the counts differ within a set, and each hole goes through.
func TestSevenSegmentLookAlike(t *testing.T) {
	t.Parallel()
	type set struct {
		name   string
		parts  []string
		counts []int
	}
	sets := []set{
		{"front cams", []string{"cam d", "cam e", "cam c"}, []int{1, 2, 3}},
		{"rear cams", []string{"cam g", "cam b", "cam f", "cam a"}, []int{1, 2, 3, 4}},
		{"walls", []string{"front wall", "rear wall"}, []int{1, 2}},
	}
	cases := map[string]partCase{}
	for _, c := range partCases() {
		cases[c.name] = c
	}
	for _, s := range sets {
		seen := map[int]bool{}
		for _, n := range s.counts {
			require.False(t, seen[n], "set %s repeats a count", s.name)
			seen[n] = true
		}
		for i, name := range s.parts {
			c := cases[name]
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				if c.pending != "" {
					t.Skip(c.pending)
				}
				_, body := sharedBody(t, c.name, c.build)
				box, err := body.Bounds()
				require.NoError(t, err)
				radius := markRadius
				switch name {
				case "cam d", "cam e", "cam c":
					radius = 18.0
				case "cam g", "cam b", "cam f", "cam a":
					radius = 12.0
				}
				// Sections just inside each face of the plate or disc show every hole: it goes through the whole part.
				ys := []float64{box.Max.Y - 0.3, box.Max.Y - CamThickness/2, box.Max.Y - CamThickness + 0.3}
				switch name {
				case "front wall":
					ys = []float64{frontInnerY - 0.3, frontInnerY - WallThickness/2, frontInnerY - WallThickness + 0.3}
				case "rear wall":
					ys = []float64{rearInnerY + 0.3, rearInnerY + WallThickness/2, rearInnerY + WallThickness - 0.3}
				}
				for _, y := range ys {
					loops := sectionAt(t, body, y)
					require.Equal(t, s.counts[i], markCount(loops, radius), "%s at y %.2f", name, y)
				}
			})
		}
	}
}
