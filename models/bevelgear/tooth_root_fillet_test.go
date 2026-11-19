package bevelgear_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

// TestRootFilletToothProfile draws every route of the straight member's tooth sketch and requires each fully
// constrained: the radial-flank tooth with a root fillet, the radial-flank tooth whose fillet does not fit, and the
// embedded-flank tooth.
func TestRootFilletToothProfile(t *testing.T) {
	embedded := bevelgear.DefaultParams()
	embedded.Teeth, embedded.MateTeeth = 60, 60
	unfitted := bevelgear.DefaultParams() // base circle 20.598 above the sketch root, but the fillet's foot is above it
	for _, tc := range []struct {
		name  string
		p     bevelgear.Params
		edges int
	}{
		{"radial_filleted", filletPinion(), 10},
		{"radial_fillet_does_not_fit", unfitted, 6},
		{"embedded_flank", embedded, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := sketch.NewWorld()
			s, profile, err := bevelgear.BuildFilletedToothProfileForTest(t.Context(), world, tc.p)
			require.NoError(t, err)
			requireFullyConstrained(t, s)
			require.True(t, profile.Valid)
			require.Empty(t, profile.Holes)
			require.Len(t, profile.Outer, tc.edges)
		})
	}
}

// TestRootFilletShape reads the solved fillet of the fan pinion's heel section. The back-cone origin is the apex of
// the virtual spur gear, whose blank root radius is R_d. Each side holds a radial flank line from T1 up to the base
// circle, a fillet arc of radius 0.38 m tangent to it at T1, a run-out line tangent to the arc at T3 that crosses the
// blank's root circle at 10 degrees, and the tooth's own root arc 0.05 mm inside the blank's root.
func TestRootFilletShape(t *testing.T) {
	p := filletPinion()
	band, err := p.ToothBand()
	require.NoError(t, err)
	require.InDelta(t, 0.76, band.RootFillet, 1e-12)
	gamma := math.Atan2(float64(p.Teeth), float64(p.MateTeeth))
	pitch := p.Module * float64(p.Teeth) / (2 * math.Cos(gamma))
	blankRoot := pitch - 1.25*p.Module
	base := pitch * math.Cos(p.PressureAngle)

	world := sketch.NewWorld()
	s, profile, err := bevelgear.BuildFilletedToothProfileForTest(t.Context(), world, p)
	require.NoError(t, err)
	requireFullyConstrained(t, s)

	var fillets, others []*sketch.Arc
	var flanks, runouts []*sketch.Line
	for _, edge := range profile.Outer {
		switch e := edge.Entity.(type) {
		case *sketch.Arc:
			if math.Abs(e.R()-band.RootFillet) < 1e-6 {
				fillets = append(fillets, e)
				continue
			}
			others = append(others, e)
		case *sketch.Line:
			a, b := e.Start, e.End
			if math.Abs(a.X()*b.Y()-a.Y()*b.X()) < 1e-6 {
				flanks = append(flanks, e)
				continue
			}
			runouts = append(runouts, e)
		}
	}
	require.Len(t, fillets, 2)
	require.Len(t, flanks, 2)
	require.Len(t, runouts, 2)
	require.Len(t, others, 2, "the tip arc and the root arc")
	var rootArc *sketch.Arc
	for _, a := range others {
		if a.R() < pitch {
			rootArc = a
		}
	}
	require.NotNil(t, rootArc)
	require.InDelta(t, blankRoot-0.05, rootArc.R(), 1e-9, "the tooth's own root arc lies 0.05 mm inside the blank's root")

	for _, flank := range flanks {
		// T1 lies on the radial line through the origin, below the base circle, where the line ends.
		tip := math.Max(math.Hypot(flank.Start.X(), flank.Start.Y()), math.Hypot(flank.End.X(), flank.End.Y()))
		require.InDelta(t, base, tip, 1e-6)
	}
	for _, runout := range runouts {
		// The run-out leaves the arc tangentially at T3, which lies above the blank's root, and ends on the root arc.
		var arc *sketch.Arc
		var t3, d *sketch.Point
		for _, f := range fillets {
			for _, end := range []*sketch.Point{runout.Start, runout.End} {
				if math.Hypot(end.X()-f.Start.X(), end.Y()-f.Start.Y()) < 1e-9 ||
					math.Hypot(end.X()-f.End.X(), end.Y()-f.End.Y()) < 1e-9 {
					arc, t3 = f, end
				}
			}
		}
		require.NotNil(t, arc)
		d = runout.Start
		if d == t3 {
			d = runout.End
		}
		require.InDelta(t, rootArc.R(), math.Hypot(d.X(), d.Y()), 1e-9)
		dx, dy := d.X()-t3.X(), d.Y()-t3.Y()
		length := math.Hypot(dx, dy)
		radial := (t3.X() - arc.Center.X()) / arc.R()
		radialY := (t3.Y() - arc.Center.Y()) / arc.R()
		require.InDelta(t, 0, (dx*radial+dy*radialY)/length, 1e-9, "run-out is tangent to the fillet arc at T3")
		require.Greater(t, math.Hypot(t3.X(), t3.Y()), blankRoot)
		// Where the run-out crosses the blank's root circle, the angle between the line and the circle's tangent.
		bq := (t3.X()*dx + t3.Y()*dy) / length
		cq := math.Hypot(t3.X(), t3.Y())*math.Hypot(t3.X(), t3.Y()) - blankRoot*blankRoot
		step := -bq - math.Sqrt(bq*bq-cq)
		require.Less(t, step, length, "the run-out crosses the blank's root before it ends")
		x, y := t3.X()+step*dx/length, t3.Y()+step*dy/length
		require.InDelta(t, blankRoot, math.Hypot(x, y), 1e-6)
		radialDot := (x*dx + y*dy) / (math.Hypot(x, y) * length)
		require.InDelta(t, 10.0, math.Asin(math.Abs(radialDot))*180/math.Pi, 0.5, "crease at the blank's root")
	}
}
