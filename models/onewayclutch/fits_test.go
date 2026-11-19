package onewayclutch_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestOneWayClutchFits(t *testing.T) {
	for _, c := range fitCases() {
		t.Run(c.name, func(t *testing.T) {
			m := fitBuild(t, c)
			cl := c.p.BoreClearance
			requireBoreFit(t, "P1 hub", m.Hub, midHub, m, cl)
			requireBoreFit(t, "P2 back plate", m.BackPlate, onewayclutch.BackPlateThickness-1, m, cl)
			requireBoreFit(t, "P2 cover", m.Cover, onewayclutch.CoverFrom+1, m, cl)
			requireFitPairs(t, m)
			band := fitAxialFits(t, m)
			require.GreaterOrEqual(t, band, onewayclutch.BandTo-onewayclutch.BandFrom-0.05, "tooth band height")
			fitRimFits(t, m)
			fitArmFits(t, m)
		})
	}
	t.Run("placed", func(t *testing.T) {
		p := onewayclutch.DefaultParams()
		f := fitFrame(t, r3.Vec{X: 30, Y: 20, Z: 10}, r3.Vec{Y: 1}, r3.Vec{X: -1})
		p.Placement = &f
		m := fitBuild(t, fitCase{name: "placed", p: p})
		requireFitPairs(t, m)
	})
}

func TestOneWayClutchPawls(t *testing.T) {
	for _, c := range fitCases() {
		t.Run(c.name, func(t *testing.T) {
			m := fitBuild(t, c)
			dm := m.Dimensions
			// The values that come from Dimensions alone.
			arm := dm.ArmOuter - dm.ArmInner
			require.GreaterOrEqual(t, arm, onewayclutch.MinArmThickness-1e-9, "arm thickness: 1.5 mm minimum")
			require.InDelta(t, onewayclutch.ArmThickness, arm, 1e-9)
			margin := dm.SlotWidth - dm.HeadTravel
			require.GreaterOrEqual(t, margin, 0.25, "slot margin s - delta")
			flankAngle := fitAngle(dm.FlankEnd)
			armSpan := flankAngle - onewayclutch.SlotAngle
			length := armSpan * math.Pi / 180 * (dm.ArmInner + dm.ArmOuter) / 2
			strain := 3 * arm * dm.HeadTravel / (2 * length * length)
			require.LessOrEqual(t, strain, 0.012, "arm strain at full travel")
			contact := dm.TipCenter.Y - dm.ToothTip
			require.GreaterOrEqual(t, contact, 0.8, "lock face contact, radial")
			require.GreaterOrEqual(t, onewayclutch.BandTo-onewayclutch.BandFrom, 9.6-1e-9, "lock face contact, axial")
			t.Logf("from Dimensions: arm %.3f, slot margin %.3f, strain %.3f%%, arm length %.1f over %.1f deg, contact %.3f",
				arm, margin, strain*100, length, armSpan, contact)

			sigma := dm.Sign
			plane := fitPlane(t, midHub)
			hub, ring := fitSection(t, m.Hub, plane), fitSection(t, m.Ring, plane)
			off := hub.OffSurface.Base() + 1e-6

			// Arm thickness and slot width, on rays through the arm and slot (the arm ends at 67.4 degrees): the
			// last three crossings are the core, the arm's inner arc and its outer arc.
			var slotMin = math.Inf(1)
			for _, layout := range []float64{10, 25, 40, 55} {
				cr := fitCrossings(hub, fitMapAngle(sigma, layout))
				require.GreaterOrEqual(t, len(cr), 3, "ray at layout %g", layout)
				n := len(cr)
				got, slot := cr[n-1]-cr[n-2], cr[n-2]-cr[n-3]
				require.InDelta(t, onewayclutch.ArmThickness, got, 2*off+0.005, "arm thickness at layout %g", layout)
				require.InDelta(t, dm.SlotWidth, slot, 2*off+0.005, "slot width at layout %g", layout)
				slotMin = math.Min(slotMin, slot)
			}

			// Head travel: the outermost hub point against the tooth tips.
			reach := 0.0
			var armMax = math.Inf(-1)
			for _, p := range fitPoints(hub) {
				r := math.Hypot(p.X, p.Y)
				reach = math.Max(reach, r)
				if math.Abs(r-dm.ArmOuter) < 0.02 {
					layout := fitMapAngle(sigma, fitAngle(p)) // the map is its own inverse
					if layout > -20+0.5 && layout < 90 {
						armMax = math.Max(armMax, layout)
					}
				}
			}
			travel := reach - dm.ToothTip
			require.InDelta(t, dm.HeadReach, reach, off+0.005, "head reach")
			require.GreaterOrEqual(t, slotMin-travel-2*off, 0.25, "slot margin from the section")
			require.InDelta(t, flankAngle, armMax, 0.3, "the arm's outer arc ends at the flank")
			l := (armMax - onewayclutch.SlotAngle) * math.Pi / 180 * (dm.ArmInner + dm.ArmOuter) / 2
			sectionStrain := 3 * (onewayclutch.ArmThickness - 2*off) * travel / (2 * l * l)
			require.LessOrEqual(t, sectionStrain, 0.012, "strain from the section")

			// Web width: from the slot's end round to the web's back face.
			centre := dm.CoreRadius + dm.SlotWidth/2
			slotEnd := r3.Vec{
				X: centre * math.Cos(fitMapAngle(sigma, onewayclutch.SlotAngle)*math.Pi/180),
				Y: centre * math.Sin(fitMapAngle(sigma, onewayclutch.SlotAngle)*math.Pi/180),
			}
			ws, wc := math.Sincos(fitMapAngle(sigma, onewayclutch.WebAngle) * math.Pi / 180)
			web, rounds := math.Inf(1), 0
			for _, p := range fitPoints(hub) {
				if math.Abs(math.Hypot(p.X-slotEnd.X, p.Y-slotEnd.Y)-dm.SlotWidth/2) < 0.02 {
					web, rounds = math.Min(web, math.Abs(wc*p.Y-ws*p.X)), rounds+1
				}
			}
			require.NotZero(t, rounds, "slot end round vertices")
			web -= off
			require.GreaterOrEqual(t, web, onewayclutch.ArmThickness, "web at least as wide as the arm")

			// Lock contact: the head's lock face against the ring's, on the 90 degree axis.
			span := func(s decad.SectionOutline, x float64) (lo, hi float64) {
				lo, hi = math.Inf(1), math.Inf(-1)
				for _, p := range fitPoints(s) {
					if p.Y > 10 && math.Abs(math.Abs(p.X)-x) < 0.005 {
						lo, hi = math.Min(lo, p.Y), math.Max(hi, p.Y)
					}
				}
				return lo, hi
			}
			headLo, headHi := span(hub, onewayclutch.HeadGap)
			ringLo, ringHi := span(ring, 0)
			contactMeasured := math.Min(headHi, ringHi) - math.Max(headLo, ringLo) - 2*off
			require.GreaterOrEqual(t, contactMeasured, 0.8, "lock face contact from the sections")
			t.Logf("sections: arm %.3f, slot %.3f, travel %.3f, margin %.3f, strain %.3f%%, web %.3f, contact %.3f",
				onewayclutch.ArmThickness, slotMin, travel, slotMin-travel, sectionStrain*100, web, contactMeasured)
		})
	}
}
