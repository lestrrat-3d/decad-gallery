package swisslever

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func arborTestBuilder(t *testing.T) *builder {
	t.Helper()
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	return newBuilder(t.Context(), NewDocument(), g)
}

// arborSection returns the section outline of body at height z. A faceted Boolean result refuses a chord below its
// own bound, so the cut retries coarser.
func arborSection(t *testing.T, body *decad.Body, z float64) decad.SectionOutline {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	var o decad.SectionOutline
	for _, tol := range []float64{0.005, 0.02, 0.05} {
		o, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol))
		if err == nil {
			break
		}
	}
	require.NoError(t, err)
	return o
}

func arborLoop(l []r3.Vec) []Pt {
	out := make([]Pt, len(l))
	for i, v := range l {
		out[i] = Pt{v.X, v.Y}
	}
	return out
}

// arborInPoly reports whether x lies inside the closed polygon (even-odd).
func arborInPoly(poly []Pt, x Pt) bool {
	in := false
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		if (a.Y > x.Y) == (b.Y > x.Y) {
			continue
		}
		if x.X < a.X+(x.Y-a.Y)/(b.Y-a.Y)*(b.X-a.X) {
			in = !in
		}
	}
	return in
}

// arborInside reports whether the point is inside the solid, by the even-odd rule on the section at its height.
func arborInside(t *testing.T, body *decad.Body, x, y, z float64) bool {
	t.Helper()
	n := 0
	for _, l := range arborSection(t, body, z).Loops {
		if arborInPoly(arborLoop(l), Pt{x, y}) {
			n++
		}
	}
	return n%2 == 1
}

func arborVolume(t *testing.T, body *decad.Body) float64 {
	t.Helper()
	v, err := body.Volume()
	require.NoError(t, err)
	return v.Value.Base()
}

// segmentArea is the area of the circular segment of a circle of radius r beyond the chord x = flat.
func segmentArea(r, flat float64) float64 {
	if r <= flat {
		return 0
	}
	return r*r*math.Acos(flat/r) - flat*math.Sqrt(r*r-flat*flat)
}

// sectionRadius returns the radius of the meridian pts at height z.
func sectionRadius(pts meridianPoints, z float64) float64 {
	for i, p := range pts {
		q := pts[(i+1)%len(pts)]
		lo, hi := math.Min(p[1], q[1]), math.Max(p[1], q[1])
		if p[1] == q[1] || z <= lo || z >= hi || (p[0] == 0 && q[0] == 0) {
			continue
		}
		return p[0] + (q[0]-p[0])*(z-p[1])/(q[1]-p[1])
	}
	return 0
}

// flatPlaneAt returns the least flat plane the spans put at height z, and whether any covers z.
func flatPlaneAt(spans []flatSpan, z float64) (float64, bool) {
	best, found := math.Inf(1), false
	for _, sp := range spans {
		x := math.Inf(1)
		switch {
		case z >= sp.Z0 && z <= sp.Z1:
			x = sp.X
		case sp.RampEnd && z > sp.Z1:
			x = sp.X + (z-sp.Z1)*math.Tan(coneHalfAngle)
		case sp.RampStart && z < sp.Z0:
			x = sp.X + (sp.Z0-z)*math.Tan(coneHalfAngle)
		}
		if x < best {
			best, found = x, true
		}
	}
	return best, found
}

// meridianVolume integrates the meridian pts, less the circular segment each flat takes off a section.
func meridianVolume(pts meridianPoints, spans []flatSpan) float64 {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, p := range pts {
		lo, hi = math.Min(lo, p[1]), math.Max(hi, p[1])
	}
	const step = 0.001
	sum := 0.0
	for z := lo + step/2; z < hi; z += step {
		r := sectionRadius(pts, z)
		area := math.Pi * r * r
		if x, ok := flatPlaneAt(spans, z); ok {
			area -= segmentArea(r, x)
		}
		sum += area * step
	}
	return sum
}

func TestArbors(t *testing.T) {
	b := arborTestBuilder(t)
	cases := []struct {
		name  string
		build func() (*decad.Body, error)
		at    Pt
		pts   meridianPoints
		spans []flatSpan
		seats [][2]float64 // z ranges of the round 5.0 bearing seats
	}{
		{"escape arbor", b.buildEscapeArbor, b.g.O, escapeArborPoints, escapeArborFlats, [][2]float64{{-5, 0}, {27, 32}}},
		{"pallet arbor", b.buildPalletArbor, b.g.P, palletArborPoints, palletArborFlats, [][2]float64{{-5, 0}, {27, 32}}},
		{"balance staff", b.buildBalanceStaff, b.g.B, palletArborPoints, palletArborFlats, [][2]float64{{-5, 0}, {27, 32}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body, err := c.build()
			require.NoError(t, err)
			require.True(t, body.IsSolid())
			// Volume: the revolve less, over each flat and its ramp, the circular segment of each section beyond it.
			want := meridianVolume(c.pts, c.spans)
			require.InDelta(t, want, arborVolume(t, body), 0.01*want)
			for _, sp := range c.spans {
				mid := (sp.Z0 + sp.Z1) / 2
				r := sectionRadius(c.pts, mid)
				require.Greater(t, r, sp.X, "the flat at z %v cuts a section", mid)
				require.False(t, arborInside(t, body, c.at.X+(sp.X+r)/2, c.at.Y, mid), "flat cut at z %v", mid)
				require.True(t, arborInside(t, body, c.at.X+sp.X-0.15, c.at.Y, mid), "shaft kept at z %v", mid)
				require.True(t, arborInside(t, body, c.at.X-r+0.1, c.at.Y, mid), "round side kept at z %v", mid)
				require.False(t, arborInside(t, body, c.at.X-r-0.05, c.at.Y, mid), "section radius at z %v", mid)
			}
			// A ramp is a plane at 30 degrees to the axis: 0.3 mm past the end of the flat the cut plane stands
			// 0.3 tan 30 degrees further out; the shaft is kept below it and cut above it.
			for _, sp := range c.spans {
				for _, ramp := range []struct {
					on bool
					z  float64
				}{{sp.RampEnd, sp.Z1 + 0.3}, {sp.RampStart, sp.Z0 - 0.3}} {
					if !ramp.on {
						continue
					}
					plane := sp.X + 0.3*math.Tan(coneHalfAngle)
					require.Greater(t, sectionRadius(c.pts, ramp.z), plane+0.05, "the ramp at z %v cuts a section", ramp.z)
					require.True(t, arborInside(t, body, c.at.X+plane-0.05, c.at.Y, ramp.z), "shaft kept under the ramp")
					require.False(t, arborInside(t, body, c.at.X+plane+0.05, c.at.Y, ramp.z), "ramp cut at z %v", ramp.z)
				}
			}
			// Each bearing seat is the full 5.0 mm round shaft: no flat and no clearance.
			for _, seat := range c.seats {
				mid := (seat[0] + seat[1]) / 2
				require.True(t, arborInside(t, body, c.at.X+2.45, c.at.Y, mid), "seat radius at z %v", mid)
				require.False(t, arborInside(t, body, c.at.X+2.55, c.at.Y, mid), "seat radius at z %v", mid)
				require.True(t, arborInside(t, body, c.at.X, c.at.Y+2.45, mid))
				require.False(t, arborInside(t, body, c.at.X, c.at.Y+2.55, mid))
			}
		})
	}
	require.NotEmpty(t, b.sketches)
}

// TestArborSketches verifies every sketch the arbor builders drew is fully constrained: DOF 0, no redundant constraint
// and no conflict.
func TestArborSketches(t *testing.T) {
	b := arborTestBuilder(t)
	for _, build := range []func() (*decad.Body, error){b.buildEscapeArbor, b.buildPalletArbor, b.buildBalanceStaff} {
		_, err := build()
		require.NoError(t, err)
	}
	// Meridian, one flat tool per flat and, for a ramp, its meridian: escape 2+... counted by the builders.
	require.GreaterOrEqual(t, len(b.sketches), 3+4)
	for _, s := range b.sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s))
	}
}
