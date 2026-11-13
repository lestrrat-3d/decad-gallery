package planetarygearbox_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestKitGearboxes builds every non-default kit once, one at a time, and applies the default gearbox's fit, edge,
// container, fastener, gear-clearance and explode checks to it. The default kit has its own tests. The kits with a
// 608ZZ planet bearing are skipped, not passed, until the Decad explode fix lands.
func TestKitGearboxes(t *testing.T) {
	for _, kit := range planetarygearbox.SupportedCombinations() {
		if kit.Name == "planet605-shaft605" {
			continue
		}
		t.Run(kit.Name, func(t *testing.T) {
			if kit.PlanetBearing == "608ZZ" {
				// Decad's derived explode moves bearing 1.1 and pin 1.1 about 14 mm less than the other two planets'
				// bearings and pins for a 608ZZ planet bearing, so the explode check fails. The kit has not been
				// verified. Remove this skip once the Decad fix lands.
				t.Skip("not verified: pending the Decad fix for the lopsided derived explode of 608ZZ planet bearings")
			}
			p, err := planetarygearbox.DefaultParamsFor(kit.Name)
			require.NoError(t, err)
			planetSpec, ok := ballbearing.Lookup(p.PlanetBearing)
			require.True(t, ok)
			shaftSpec, ok := ballbearing.Lookup(p.ShaftBearing)
			require.True(t, ok)
			doc := planetarygearbox.NewDocument()
			box, err := planetarygearbox.Build(t.Context(), doc, p)
			require.NoError(t, err)

			require.Len(t, doc.Bodies(), 34)
			for _, nb := range box.Named() {
				require.True(t, nb.Body.IsSolid(), "%s is solid", nb.Name)
				volume, err := nb.Body.Volume()
				require.NoError(t, err)
				require.Positive(t, volume.Value.Base(), "%s has volume", nb.Name)
			}
			requireContainers(t, doc, box)

			// Each planet bearing is the planet bearing's size and sits lifted in its pocket. The covers' bearings
			// are the shaft bearing's size.
			stage := box.Stages[0]
			for i, bearing := range stage.Bearings {
				b := bounds(t, bearing)
				require.InDelta(t, planetSpec.OuterDiameter, b.Max.X-b.Min.X, 0.05, "planet bearing %d diameter", i)
				require.InDelta(t, planetSpec.Width, b.Max.Z-b.Min.Z, 1e-6, "planet bearing %d width", i)
				require.InDelta(t, p.GearThickness-planetSpec.Width+0.25, b.Min.Z, 1e-6, "planet bearing %d height", i)
			}
			for name, bearing := range map[string]*decad.Body{"front": box.FrontBearing, "rear": box.RearBearing} {
				b := bounds(t, bearing)
				require.InDelta(t, shaftSpec.OuterDiameter, b.Max.X-b.Min.X, 0.05, "%s bearing diameter", name)
				require.InDelta(t, shaftSpec.Width, b.Max.Z-b.Min.Z, 1e-6, "%s bearing width", name)
			}

			// The rear cover's boss is as wide as its pocket plus two carrier walls, at the boss's mid-height.
			pitch := p.GearThickness + p.CarrierThickness + 2*p.StageGap
			bossMid := pitch + p.CoverThickness + shaftSpec.Width/2
			boss := sectionRadialSpan(t, box.RearCover, r3.Vec{}, bossMid, 0)
			wantBoss := (shaftSpec.OuterDiameter + 2*p.BearingClearance + 2*p.CarrierWall) / 2
			require.InDelta(t, wantBoss, boss.max, boss.bound+2e-3, "rear cover boss radius")

			requireFitClearances(t, box, p)
			requireEdgeFinishing(t, box, p)
			requireFastenersFlush(t, box, p)
			for i, planet := range stage.Planets {
				require.Zero(t, penetration(t, planet, stage.Sun), "planet %d inside the sun", i)
				require.Zero(t, penetration(t, planet, box.Ring), "planet %d inside the ring", i)
			}
			requireTipRootClearance(t, box, p)
			requireExplodedGearbox(t, doc, box, p)
			// It copies bodies into the document, so it runs last.
			requireFastenersClear(t, box)
		})
	}
}
