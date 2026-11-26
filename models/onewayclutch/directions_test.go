package onewayclutch_test

// CI: isolated-tests

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestOneWayClutchRejectsInvalidInputs(t *testing.T) {
	with := func(edit func(p *onewayclutch.Params)) onewayclutch.Params {
		p := onewayclutch.DefaultParams()
		edit(&p)
		return p
	}
	tests := []struct {
		name    string
		params  onewayclutch.Params
		message string
	}{
		{"unknown shaft", with(func(p *onewayclutch.Params) { p.Shaft = "6x6" }),
			`onewayclutch: unknown shaft preset "6x6"; want one of 605ZZ, 608ZZ, 608ZZ-gearbox`},
		{"empty shaft", with(func(p *onewayclutch.Params) { p.Shaft = "" }), "unknown shaft preset"},
		{"shaft name is exact", with(func(p *onewayclutch.Params) { p.Shaft = "605zz" }), "unknown shaft preset"},
		{"zero direction", with(func(p *onewayclutch.Params) { p.Freewheel = 0 }),
			"onewayclutch: Freewheel must be Clockwise or CounterClockwise"},
		{"unknown direction", with(func(p *onewayclutch.Params) { p.Freewheel = 3 }),
			"Freewheel must be Clockwise or CounterClockwise"},
		{"negative direction", with(func(p *onewayclutch.Params) { p.Freewheel = -1 }), "Freewheel must be"},
		{"clearance too small", with(func(p *onewayclutch.Params) { p.BoreClearance = 0 }), "BoreClearance must be"},
		{"clearance negative", with(func(p *onewayclutch.Params) { p.BoreClearance = -0.1 }), "BoreClearance must be"},
		{"clearance too large", with(func(p *onewayclutch.Params) { p.BoreClearance = 0.6 }), "BoreClearance must be"},
		{"clearance NaN", with(func(p *onewayclutch.Params) { p.BoreClearance = math.NaN() }), "BoreClearance must be"},
		{"clearance +Inf", with(func(p *onewayclutch.Params) { p.BoreClearance = math.Inf(1) }), "BoreClearance must be"},
		{"clearance -Inf", with(func(p *onewayclutch.Params) { p.BoreClearance = math.Inf(-1) }), "BoreClearance must be"},
		{"invalid placement", with(func(p *onewayclutch.Params) { p.Placement = &r3.Frame{} }),
			"Placement must be a valid frame"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			require.Error(t, err)
			require.ErrorContains(t, err, "onewayclutch: ")
			require.ErrorContains(t, err, tc.message)
			_, err = tc.params.Derive()
			require.Error(t, err)

			doc := onewayclutch.NewDocument()
			_, err = onewayclutch.BuildParts(t.Context(), doc, tc.params)
			require.ErrorContains(t, err, tc.message)
			_, err = onewayclutch.Build(t.Context(), doc, tc.params)
			require.ErrorContains(t, err, tc.message)
			_, err = onewayclutch.BuildAdapter(t.Context(), doc, tc.params)
			require.ErrorContains(t, err, tc.message)
			require.Empty(t, doc.Bodies(), "a rejected build adds no body")
		})
	}

	t.Run("first failure wins", func(t *testing.T) {
		p := onewayclutch.Params{Shaft: "nope", Freewheel: 0, BoreClearance: math.NaN(), Placement: &r3.Frame{}}
		require.ErrorContains(t, p.Validate(), "unknown shaft preset")
		p.Shaft = "608ZZ"
		require.ErrorContains(t, p.Validate(), "Freewheel must be")
		p.Freewheel = onewayclutch.Clockwise
		require.ErrorContains(t, p.Validate(), "BoreClearance must be")
		p.BoreClearance = 0.35
		require.ErrorContains(t, p.Validate(), "Placement must be a valid frame")
		p.Placement = nil
		require.NoError(t, p.Validate())
	})

	t.Run("limits that stay valid", func(t *testing.T) {
		p := onewayclutch.DefaultParams()
		p.BoreClearance = 0.5
		require.NoError(t, p.Validate(), "605ZZ takes the largest clearance")
		for _, name := range []string{"608ZZ", "608ZZ-gearbox"} {
			p.Shaft = name
			require.NoError(t, p.Validate(), "%s takes the largest clearance", name)
		}
		for _, shaft := range onewayclutch.Shafts() {
			p := onewayclutch.DefaultParams()
			p.Shaft = shaft.Name
			require.NoError(t, p.Validate(), shaft.Name)
		}
	})

	t.Run("nil context and document", func(t *testing.T) {
		var nilCtx context.Context
		doc := onewayclutch.NewDocument()
		_, err := onewayclutch.BuildParts(nilCtx, doc, onewayclutch.DefaultParams())
		require.ErrorContains(t, err, "context must not be nil")
		_, err = onewayclutch.Build(nilCtx, doc, onewayclutch.DefaultParams())
		require.ErrorContains(t, err, "context must not be nil")
		_, err = onewayclutch.BuildAdapter(nilCtx, doc, onewayclutch.DefaultParams())
		require.ErrorContains(t, err, "context must not be nil")
		_, err = onewayclutch.BuildAdapter(t.Context(), nil, onewayclutch.DefaultParams())
		require.ErrorContains(t, err, "document must not be nil")
		_, err = onewayclutch.BuildParts(t.Context(), nil, onewayclutch.DefaultParams())
		require.ErrorContains(t, err, "document must not be nil")
		_, err = onewayclutch.Build(t.Context(), nil, onewayclutch.DefaultParams())
		require.ErrorContains(t, err, "document must not be nil")
		require.Empty(t, doc.Bodies())
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		doc := onewayclutch.NewDocument()
		_, err := onewayclutch.Build(ctx, doc, onewayclutch.DefaultParams())
		require.ErrorIs(t, err, context.Canceled)
		require.Empty(t, doc.Bodies())
	})

	t.Run("direction helpers", func(t *testing.T) {
		require.Equal(t, "clockwise", onewayclutch.Clockwise.String())
		require.Equal(t, "counter-clockwise", onewayclutch.CounterClockwise.String())
		require.Equal(t, -1.0, onewayclutch.Clockwise.Sign())
		require.Equal(t, 1.0, onewayclutch.CounterClockwise.Sign())
		require.Zero(t, onewayclutch.Direction(0).Sign())
		_, ok := onewayclutch.LookupShaft("608ZZ")
		require.True(t, ok)
		_, ok = onewayclutch.LookupShaft("608zz")
		require.False(t, ok)
		shafts := onewayclutch.Shafts()
		shafts[0].Name = "changed"
		require.Equal(t, "605ZZ", onewayclutch.Shafts()[0].Name, "Shafts returns a copy")
	})
}

func TestOneWayClutchSketches(t *testing.T) {
	for _, tc := range dirCases() {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := buildCase(t, tc.params)
			require.NotEmpty(t, m.Sketches)
			// Hub, ring tooth tool, two meridians, two plate profiles and the plates' inner tools come first, then one
			// fastener tool per bolt for each plate; none may be missing.
			require.GreaterOrEqual(t, len(m.Sketches), 6+2*len(m.Bolts))
			for i, s := range m.Sketches {
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
			}
		})
	}
}

func TestOneWayClutchMirror(t *testing.T) {
	for _, shaft := range onewayclutch.Shafts() {
		t.Run(shaft.Name, func(t *testing.T) {
			ccwParams := onewayclutch.DefaultParams()
			ccwParams.Shaft, ccwParams.Freewheel = shaft.Name, onewayclutch.CounterClockwise
			cwParams := ccwParams
			cwParams.Freewheel = onewayclutch.Clockwise
			ccw, _ := buildCase(t, ccwParams)
			cw, _ := buildCase(t, cwParams)

			require.Equal(t, 1.0, ccw.Dimensions.Sign)
			require.Equal(t, -1.0, cw.Dimensions.Sign)

			// The hub and the ring change hand: equal volume, bounds mirrored in x, y and z unchanged.
			for name, pair := range map[string][2]*decad.Body{"hub": {cw.Hub, ccw.Hub}, "ring": {cw.Ring, ccw.Ring}} {
				vc, bc := volumeOf(t, pair[0], name)
				vm, bm := volumeOf(t, pair[1], name)
				require.InDelta(t, vm, vc, bc+bm+1e-6, name+" volume")
				a, b := boxOf(t, pair[0], name), boxOf(t, pair[1], name)
				tol := a.Bound.Base() + b.Bound.Base() + 1e-3
				require.InDelta(t, -b.Max.X, a.Min.X, tol, name+" min x")
				require.InDelta(t, -b.Min.X, a.Max.X, tol, name+" max x")
				require.InDelta(t, b.Min.Y, a.Min.Y, tol, name+" min y")
				require.InDelta(t, b.Max.Y, a.Max.Y, tol, name+" max y")
				require.InDelta(t, b.Min.Z, a.Min.Z, tol, name+" min z")
				require.InDelta(t, b.Max.Z, a.Max.Z, tol, name+" max z")
			}

			// The plates, the bolts and the nuts are the same in both directions.
			same := map[string][2]*decad.Body{
				"back plate": {cw.BackPlate, ccw.BackPlate}, "cover": {cw.Cover, ccw.Cover},
			}
			for i := range cw.Bolts {
				same[boltName(i)] = [2]*decad.Body{cw.Bolts[i].Body, ccw.Bolts[i].Body}
				same[nutName(i)] = [2]*decad.Body{cw.Nuts[i].Body, ccw.Nuts[i].Body}
			}
			for name, pair := range same {
				vc, bc := volumeOf(t, pair[0], name)
				vm, bm := volumeOf(t, pair[1], name)
				require.InDelta(t, vm, vc, bc+bm+1e-6, name+" volume")
				a, b := boxOf(t, pair[0], name), boxOf(t, pair[1], name)
				tol := a.Bound.Base() + b.Bound.Base() + 1e-3
				require.InDelta(t, b.Min.X, a.Min.X, tol, name+" min x")
				require.InDelta(t, b.Max.X, a.Max.X, tol, name+" max x")
				require.InDelta(t, b.Min.Y, a.Min.Y, tol, name+" min y")
				require.InDelta(t, b.Max.Y, a.Max.Y, tol, name+" max y")
				require.InDelta(t, b.Min.Z, a.Min.Z, tol, name+" min z")
				require.InDelta(t, b.Max.Z, a.Max.Z, tol, name+" max z")
			}
			require.Equal(t, cw.Sites, ccw.Sites)

			// The pawl layout of Dimensions is the counter-clockwise one for both: only Sign differs.
			d1, d2 := cw.Dimensions, ccw.Dimensions
			d1.Sign, d2.Sign = 0, 0
			require.Equal(t, d1, d2)
		})
	}
}

func TestOneWayClutchFreewheelDirection(t *testing.T) {
	const turn = 2.0 // degrees: past the 0.40 mm head gap in the drive direction, short of it in the free direction
	for _, dir := range []onewayclutch.Direction{onewayclutch.Clockwise, onewayclutch.CounterClockwise} {
		t.Run(dir.String(), func(t *testing.T) {
			p := onewayclutch.DefaultParams()
			p.Freewheel = dir
			doc := onewayclutch.NewDocument()
			parts, err := onewayclutch.BuildParts(t.Context(), doc, p)
			require.NoError(t, err)
			sigma := dir.Sign()
			dm, err := p.Derive()
			require.NoError(t, err)
			toothTip := dm.ToothTip

			// The hub is turned, not the ring: a ring made by a Boolean is a mesh that Intersect cannot take, and turning the hub
			// by an angle equals turning the ring by its opposite.
			rotated := func(angle float64) *decad.Body {
				hub, err := decad.Copy(parts.Hub).Do(t.Context())
				require.NoError(t, err)
				xf, err := r3.Rotation(r3.Vec{Z: 1}, units.Degrees(angle))
				require.NoError(t, err)
				hub, err = decad.Transform(hub, xf).Do(t.Context())
				require.NoError(t, err)
				return hub
			}

			// The ring turned by -sigma*turn (the hub by +sigma*turn) closes the head gap: the lock faces cross and the
			// bodies overlap. The turned bodies are meshes that
			// Intersect cannot take, so the test counts hub vertices inside the ring's mesh.
			ring := tessellate(t, parts.Ring, "ring")
			locked := tessellate(t, rotated(sigma*turn), "locked hub")
			require.Positive(t, countInside(locked.Vertices(), ring, toothTip), "the locked hub reaches into the ring")

			// The ring turned by +sigma*turn (the hub by -sigma*turn): the heads ride up the ramps and stay clear. The
			// design requires 0.05 mm left after 2 degrees and at 1.5 degrees. Measured on the 605ZZ hub: 0.188 mm at 1.5
			// degrees and 0.126 at 2.0, in both directions; on the 8 mm hubs 0.176 and 0.112 (BoreClearance 0.05), and at
			// least 0.157 and 0.093 (0.5).
			for _, free := range []struct{ angle, clearance float64 }{{1.5, 0.05}, {turn, 0.05}} {
				hub := rotated(-sigma * free.angle)
				gap, clear, err := meshOf(t, hub, "free hub").Exceeds(meshOf(t, parts.Ring, "ring"), free.clearance)
				require.NoError(t, err)
				t.Logf("free way, hub turned %g degrees: proven gap %.3f mm", free.angle, gap)
				require.True(t, clear, "the hub turned %g degrees the free way clears the ring by %g mm (gap bound %g)",
					free.angle, free.clearance, gap)
				require.Zero(t, countInside(tessellate(t, hub, "free hub").Vertices(), ring, toothTip))
			}

			// The unturned ring clears the hub by at least the head gap's smaller share.
			gap, clear, err := meshOf(t, parts.Ring, "ring").Exceeds(meshOf(t, parts.Hub, "hub"), 0.2)
			require.NoError(t, err)
			require.True(t, clear, "the seated ring clears the hub by 0.2 mm (gap bound %g)", gap)
		})
	}
}

func TestOneWayClutchPlacement(t *testing.T) {
	frames := map[string]struct{ origin, x, y r3.Vec }{
		// A quarter turn about Z and a move.
		"turned": {r3.Vec{X: 12, Y: -7, Z: 3}, r3.Vec{Y: 1}, r3.Vec{X: -1}},
		// A flip about X, so +Z points back along -Z.
		"flipped": {r3.Vec{X: -4, Y: 6, Z: 30}, r3.Vec{X: 1}, r3.Vec{Y: -1}},
	}
	local, _ := buildDefault(t)
	for name, fr := range frames {
		t.Run(name, func(t *testing.T) {
			frame, err := r3.NewFrame(fr.origin, fr.x, fr.y)
			require.NoError(t, err)
			xf, err := r3.FromFrame(frame)
			require.NoError(t, err)
			p := onewayclutch.DefaultParams()
			p.Placement = &frame
			placed, err := onewayclutch.Build(t.Context(), onewayclutch.NewDocument(), p)
			require.NoError(t, err)

			// moved returns the bounds of the local box under the frame (exact for the axis-aligned frames used here).
			moved := func(box decad.Box) (r3.Vec, r3.Vec) {
				lo, hi := xf.Apply(box.Min), xf.Apply(box.Max)
				return r3.Vec{X: math.Min(lo.X, hi.X), Y: math.Min(lo.Y, hi.Y), Z: math.Min(lo.Z, hi.Z)},
					r3.Vec{X: math.Max(lo.X, hi.X), Y: math.Max(lo.Y, hi.Y), Z: math.Max(lo.Z, hi.Z)}
			}
			want, got := partsOf(local), partsOf(placed)
			require.Len(t, got, len(want))
			for part, body := range want {
				a, b := boxOf(t, body, part), boxOf(t, got[part], part)
				lo, hi := moved(a)
				tol := a.Bound.Base() + b.Bound.Base() + 1e-3
				require.InDelta(t, lo.X, b.Min.X, tol, part+" min x")
				require.InDelta(t, lo.Y, b.Min.Y, tol, part+" min y")
				require.InDelta(t, lo.Z, b.Min.Z, tol, part+" min z")
				require.InDelta(t, hi.X, b.Max.X, tol, part+" max x")
				require.InDelta(t, hi.Y, b.Max.Y, tol, part+" max y")
				require.InDelta(t, hi.Z, b.Max.Z, tol, part+" max z")
				va, ba := volumeOf(t, body, part)
				vb, bb := volumeOf(t, got[part], part)
				require.InDelta(t, va, vb, ba+bb+1e-6, part+" volume")
			}
			for i, site := range local.Sites {
				w := placed.Sites[i]
				require.InDelta(t, 0, xf.Apply(site.Head).Sub(w.Head).Len(), 1e-9, "site %d head", i)
				require.InDelta(t, 0, xf.ApplyDir(site.Dir).Sub(w.Dir).Len(), 1e-9, "site %d dir", i)
				require.InDelta(t, 0, xf.ApplyDir(site.Corner).Sub(w.Corner).Len(), 1e-9, "site %d corner", i)
			}
			for i, s := range placed.Sketches {
				require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "sketch %d", i)
			}
		})
	}
}

// TestOneWayClutchDerivedRadii checks the radii Derive grows with the shaft: the walls round the bolts, the slot margin
// and the arm and tooth relations hold for every preset and clearance, and the 605ZZ clutch keeps its former size.
func TestOneWayClutchDerivedRadii(t *testing.T) {
	const eps = 1e-9
	for _, shaft := range onewayclutch.Shafts() {
		for _, c := range []float64{0.05, 0.3, 0.326, 0.35, 0.5} {
			t.Run(fmt.Sprintf("%s c %g", shaft.Name, c), func(t *testing.T) {
				p := onewayclutch.DefaultParams()
				p.Shaft, p.BoreClearance = shaft.Name, c
				dm, err := p.Derive()
				require.NoError(t, err)
				require.GreaterOrEqual(t, dm.CoreRadius-dm.BoltCircle-m2fastener.HoleRadius, onewayclutch.PressWall-eps,
					"hole to core")
				pocket := math.Max(m2fastener.HeadPocketRadius, m2fastener.NutCavityRadius*math.Sqrt(3)/2)
				require.GreaterOrEqual(t, dm.BoltCircle-pocket-dm.BoreRadius, onewayclutch.PressWall-eps,
					"bore to the bolt pockets")
				for i, a := range onewayclutch.BoltAngles {
					centre := r3.Vec{X: dm.BoltCircle * math.Cos(a*math.Pi/180), Y: dm.BoltCircle * math.Sin(a*math.Pi/180)}
					require.GreaterOrEqual(t, distanceToBore(centre.X, centre.Y, dm.BoreRadius, dm.BoreFlat)-pocket, onewayclutch.PressWall-eps,
						"bolt %d: bore to its pockets", i+1)
				}
				require.GreaterOrEqual(t, dm.SlotWidth-dm.HeadTravel, 0.25, "slot margin")
				require.InDelta(t, onewayclutch.ArmThickness, dm.ArmOuter-dm.ArmInner, eps)
				require.InDelta(t, onewayclutch.LooseFit, dm.ToothTip-dm.ArmOuter, eps)
				require.Greater(t, dm.HubChamfer, 0.0)
				require.InDelta(t, dm.EarReach, dm.AdapterRadius, eps)
				require.InDelta(t, shaft.Diameter/2+onewayclutch.AdapterShaftGap, dm.AdapterInner, eps)
				require.InDelta(t, dm.CoreRadius+onewayclutch.BossStep, dm.BossRadius, eps)
				if shaft.Name != "605ZZ" {
					return
				}
				want := map[string]float64{
					"CoreRadius": 10.1, "SlotWidth": 1.6, "ArmOuter": 13.2, "ToothTip": 13.75, "ToothRoot": 16.1,
					"RingCounter": 17.1, "RingOuter": 19.1, "EarCircle": 21.6, "PlateRadius": 16.7, "BossRadius": 10.4,
					"HubChamfer": 0.25, "BoltCircle": 7.15, "AdapterInner": 3.5, "AdapterRadius": 25.7, "Growth": 0,
				}
				got := map[string]float64{
					"CoreRadius": dm.CoreRadius, "SlotWidth": dm.SlotWidth, "ArmOuter": dm.ArmOuter, "ToothTip": dm.ToothTip,
					"ToothRoot": dm.ToothRoot, "RingCounter": dm.RingCounter, "RingOuter": dm.RingOuter,
					"EarCircle": dm.EarCircle, "PlateRadius": dm.PlateRadius, "BossRadius": dm.BossRadius,
					"HubChamfer": dm.HubChamfer, "BoltCircle": dm.BoltCircle, "AdapterInner": dm.AdapterInner,
					"AdapterRadius": dm.AdapterRadius, "Growth": dm.Growth,
				}
				for name, v := range want {
					require.InDelta(t, v, got[name], 1e-6, name)
				}
			})
		}
	}
}
