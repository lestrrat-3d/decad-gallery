# Screw gear Decad construction steps

Generate `models/screwgear/*.go` from this recipe and [instructions.md](instructions.md). Use the dependency
versions in the repository `go.mod`, `internal/sketchcheck`, `internal/placement`, and the decad capabilities
listed under "Decad facts". The contract owns formulas, defaults, input checks and case inputs; this file owns
the Sketch and Decad calls and the acceptance assertions.

## Decad facts

These were measured against the decad branches the gallery builds with. They decide how the steps are written.

| Fact | Consequence for the steps |
| --- | --- |
| `decad.LoftSections` admits 3 to 1024 sections | A ribbon uses `10·N + 1` sections: 681 at 68 teeth, 161 at 16 teeth |
| The twisted fitted-edge route needs section planes parallel to world XY with the twist axis at the plane origin, strictly ascending Z, and the same fit-point count in every section | Each ribbon is built in its own frame along +Z and moved onto its gear afterwards. Every section has 11 fit points |
| The long ribbon takes about 12 s and 1.3 GB | Build the two ribbons one after another. A default build takes about 40 s and a 16-tooth build about 15 s |
| Booleans with the long ribbon are refused | The ribbons are never operands of a boolean. They are built whole and only moved |
| `decad.Sweep` with `WithSweepTwist` admits off-axis sections and any twist angle. Earlier it refused 35 to 55 degrees | The bore tool is one sweep with the twist `(sOut − sIn)/Λ` of about 80 degrees |
| `decad.Cut` of a twisted tool needs the tool to overshoot the wall by at least 0.75 mm each side | The bore cut overshoots by 1 mm at each end (`sIn` and `sOut`) |
| The six chained cuts on one curved tube need the document tolerance of `WithChainTolerance`, 0.1 mm. A window cut over a bore rim is why | `NewDocument` sets `decad.WithChainTolerance(units.Millimeters(0.1))`. `Build` needs a document made by it |
| `decad.Verify` with clearances took 108 s on the 16-tooth model, found no interference, measured 0.548 ± 0.027 mm between the ribbons, and left both ribbon-versus-cage pairs undecided | Clearance checks use `DistanceToPoint` on the cage, not `decad.Verify` |

These capabilities landed in decad in pull requests 2629 to 2634. The gallery `go.mod` pins decad at
`v0.0.0-20261027045205-cfe5a7e9e746` or later, the first revision that has all of them.

## Package and API

Generate package `screwgear` at `github.com/lestrrat-3d/decad-gallery/models/screwgear`.

| File | Responsibility |
| --- | --- |
| `params.go` | `Params`, `DefaultParams`, derived values, `Validate` |
| `frame.go` | Pair frame and the two gears' frames |
| `ribbon.go` | Ribbon section sketch, whole-length loft, placement onto a frame |
| `sleeve_geometry.go` | Sleeve tube, bore openings, roof side, section in the wall, wall separation |
| `window_search.go` | Window hexagon search and `room` |
| `sleeve_build.go` | Tube, bore tools, window tools, and the cut order |
| `build.go` | `Mechanism`, `NewDocument`, `Build` |
| `*_test.go` | Tests named under "Acceptance tests" |
| `export_test.go` | Package-private values the external tests read |

Expose these:

```go
type Params struct { /* the Inputs table of instructions.md, ending in Placement *r3.Frame */ }
type Mechanism struct{ GearA, GearB, Cage *decad.Body }

func DefaultParams() Params
func (p Params) Validate() error
func (p Params) Lambda() float64
func (p Params) Beta() float64
func (p Params) AxisOffset() float64
func (p Params) Length() float64
func (p Params) Travel() float64
func (p Params) AxialWindow() float64
func (p Params) SleeveInner() float64
func (p Params) SleeveOuter() float64
func (p Params) BoreHalfWidth() float64
func (p Params) BoreHalfThickness() float64
func (p Params) BoreCorner() float64
func (p Params) SleeveCut() (sIn, sOut float64)
func NewDocument(opts ...decad.DocumentOption) *decad.Document
func Build(ctx context.Context, doc *decad.Document, p Params) (*Mechanism, error)
```

- `DefaultParams` returns the defaults of the Inputs table. `ToothSlant` is `25.8·π/180`, `CrossAngle` is
  `80·π/180`, and the mounting angles are 0.
- `NewDocument` returns `decad.New` with `decad.WithChainTolerance(units.Millimeters(0.1))` applied first, then
  the caller's options.
- `Build` returns an error when `ctx` or `doc` is nil, then calls `Validate`. It wraps stage errors with `%w` and
  names the gear (`gear A`, `gear B`) or `sleeve`.
- `Mechanism` holds the three bodies. After `Build`, a fresh document holds exactly three bodies and each is
  solid.
- `Validate` follows "Input checks and limits" of `instructions.md`, in its order. The error text names the input
  by its dialog id (`ribbonWidth`, `cageRadius`, `collarWall`) and its bound.

The private pair frame holds `Params`, the placement frame, and the two gears. A gear holds `origin`, `ex`, `ey`,
`ez` (the section `û`, the section `v̂`, and the axis direction), `mount` and `phase`. Its angle at station `s` is
`s/Λ + mount`. Gear A is `origin = (0, 0, −A/2)`, `ex = +Z`, `ez = (cos Σ/2, sin Σ/2, 0)`. Gear B is
`origin = (0, 0, +A/2)`, `ex = −Z`, `ez = (cos Σ/2, −sin Σ/2, 0)`, `phase = AssemblyPhase`. For both,
`ey = ez × ex`. A nil placement is the frame `NewFrame((0,0,0), (1,0,0), (0,1,0))`.

## Step 1 — ribbon section sketch

For gear `g` at station `s`, draw the cross-section on the plane `z = s` of the gear's own frame. Use the world
XY plane when `s == 0`, and `CreateOffsetPlane(XY, s)` otherwise. `Sketch.Origin()` is the twist axis, so each
section's origin is the point where the gear's axis pierces it.

Let `(sin, cos) = Sincos(θ(s))` and `at(u, v)` be `sketchcheck.DimensionPoint(sk, u·cos − v·sin, u·sin + v·cos)`.
`DimensionPoint` creates the point and adds a horizontal and a vertical distance constraint from the origin.

1. `back0 = at(−W/2, −T/2)`.
2. For `j = 0…10`: `v = −T/2 + T·j/10`; `fit[j] = at(Utooth(v, s), v)`, with
   `Utooth(v, s) = W/2 − H/2 + (H/2)·cos(2π (s + tan(Slant)·v − Z_g)/P) − Bow·v²`.
3. `back1 = at(−W/2, T/2)`.
4. Lines `back0 → fit[0]`, `fit[10] → back1` and `back1 → back0`. One `CreateFitSpline(fit[0] … fit[10])`.
5. Solve the sketch, then require `sketchcheck.VerifyFullyConstrained`. All 13 points carry two distance
   constraints, so `DOF == 0`.
6. Select the profile with `Valid`, four outer edges (three lines and the spline) and no holes. Return it as a
   `decad.LoftSection`. Return an error naming the station when there is none.

Check: the section's profile has four outer edges and the same 11 fit points at every station.

## Step 2 — whole-length ribbon loft

For gear `g`, with `count = 10·N + 1`, `step = L/(count − 1)` and `first = −L/2 + Z_g`:

1. Create one `sketch.NewWorld()` for the ribbon. Build the section at `first + i·step` for `i = 0…count−1`, by
   Step 1. The sections are strictly ascending in Z and 10 per tooth pitch.
2. Call `decad.LoftSections(ctx, doc, sections...)` once. This is one loft through all sections with the twisted
   fitted edge. Wrap its error with `loft ribbon`.

Check: `body.IsSolid()`. The volume equals the integral of the section area over the length, within the bound
`Volume` reports. The twist turns the section rigidly and leaves its area alone, so the area at each station is
`∫ (Utooth(v, s) + W/2) dv` over `|v| ≤ T/2`.

## Step 3 — move the ribbon onto its gear

Build `r3.NewFrame(g.origin, g.ex, g.ey)`. Its normal is `ex × ey = ez`. Convert it with `r3.FromFrame` and apply
`decad.Transform`. The ribbon's own +Z axis becomes the gear's axis. This runs after the loft, since the loft
needs world-XY section planes.

## Step 4 — tube

1. Create a sketch on the world XY plane. Add two circles centred on `s.Origin()` with radii `Ri` and `Ro`, each
   with `sketch.NewRadius`. Solve and verify.
2. Select the profile with `Valid` and exactly one hole. This is the ring.
3. `decad.Extrude` the ring with `decad.Distance{D: 2·CageRise, Dir: decad.Along}`.
4. `decad.Transform` it by `r3.Translation(0, 0, −CageRise)`, so it spans `−CageRise` to `+CageRise`.

Check: bounds are `±CageRise` along Z and `Ro` along X.

## Step 5 — bore tools and the four cuts

For each bore in the order gear A `−R`, gear A `+R`, gear B `−R`, gear B `+R` (`sign = −1` then `+1` for each
gear):

1. The span `(s0, s1)` is `(sIn, sOut)` for `sign = +1` and `(−sOut, −sIn)` for `sign = −1`.
2. Compute the opening `(hw, vLo, vHi)` from `boreOpening`: `roofSide` is `+1` when
   `−sin θ(sc)·ex.Z + cos θ(sc)·ey.Z > 0` with `sc = sign·CageRadius`. Then `(vLo, vHi) = (−ht, ht + a)` for
   `+1` and `(−ht − a, ht)` for `−1`.
3. On the world XY plane, create four points at the opening's corners `(−hw, vLo)`, `(hw, vLo)`, `(hw, vHi)` and
   `(−hw, vHi)`, each turned by `θ(s0)` and created with `DimensionPoint`. Join them with four lines in order.
   Solve and verify. The sketch has one profile.
4. `decad.NewPath(r3.Vec{}, decad.LineTo{End: (0, 0, s1 − s0)})`.
5. `decad.Sweep(ctx, doc, sk, profile, path, decad.WithSweepTwist(units.Radians((s1 − s0)/Λ)))`.
6. Move the tool with `r3.NewFrame(g.origin + ez·s0, ex, ey)` as in Step 3.
7. `decad.Cut(ctx, cage, tool)`, which returns the cage for the next bore.

The tool starts in air at `s0` and ends in air at `s1`, 1 mm past the material. Check: after the four cuts the
cage is one solid.

## Step 6 — window search

Run the search of "The windows" in `instructions.md` for each facing in `windowFacings(p)`:
`(0, 1, 0)` and `(0, −1, 0)` when `Σ ≤ π/2`, otherwise `(1, 0, 0)` and `(−1, 0, 0)`.

The search needs these private pieces:

| Piece | Computation |
| --- | --- |
| `sectionInWall(g, s)` | Opening rectangle turned by `θ(s)`, clipped to `x` between `(−CageRise − origin.Z)/ex.Z` and `(CageRise − origin.Z)/ex.Z`, then to the two strips of `y` between `near` and `far`. Returns two convex polygons of at most eight corners. Returns none at `|s| ≥ Ro` |
| `channelSections(b)` | Station table of `sectionInWall` plus, per piece, the mean of its corners and the largest distance to a corner. Stations every 0.002 mm across the bore's span, and every 0.0001 mm between two neighbours that differ in which pieces exist |
| `wallGap(b, q, reach)` | Least distance from `q` to the bore's channel in the wall, or `reach`. Walks stations outward from the one nearest `q` along the axis, both ways, and stops each way when `(sq − s)² ≥ best`. Skips a piece whose bounding circle cannot beat `best` |
| `wallCorners(b, fn)` | Every corner of every piece at stations every 0.001 mm of the span |
| `channelTop()` | Largest distance from the mid plane reached by any channel in the wall, at 0.01 mm stations |
| `windowEnd(w, far, dir)` | The 24-step bisection with `need = CollarWall + 0.1/√2 + 0.005` |
| `clipCorners`, `dropRepeats` | Hexagon clipping and removal of corners within 0.001 mm of the one before |

Check: `room()` returns `"the flanking bores leave no band between them"` when `hi ≤ lo`,
`"the far bores leave the band no length"` when `right ≤ left`, there are fewer than three corners, or the area is
not positive, and `""` otherwise. Run the search at the defaults and compare with the numbers in "Acceptance
cases" of `instructions.md`.

## Step 7 — windows

For each window whose `room()` is `""`, in facing order:

1. Build the frame `r3.NewFrame(r3.Vec{}, across, (0, 0, 1))` and a plane from it with
   `world.CreatePlaneFromFrame`. The frame's normal is the window's facing `d`. Create a sketch on the plane.
2. Create a point at each hexagon corner `(t, z)` with `DimensionPoint` and join consecutive points with lines.
   Solve and verify. Require exactly one valid profile.
3. `decad.Extrude` the profile with `decad.Distance{D: Ro + 1, Dir: decad.Along}`. This pushes the prism toward
   `d`.
4. `decad.Cut(ctx, cage, tool)`.

The six chained cuts (four bores and two windows) need the document from `NewDocument`. A window without room is
skipped without an error.

## Step 8 — place the bodies

Call `placement.Place(ctx, body, &pf.frame)` on gear A, gear B and the cage. It converts the placement with
`r3.FromFrame` and applies `decad.Transform`. This is the last step, after every boolean, since the booleans fail
on bodies built at an offset. A nil `Placement` makes the pair frame the world frame, and the transform is the
identity.

## Failure checks

| Condition | Result |
| --- | --- |
| `ctx` or `doc` is nil | Error, before any geometry |
| `Validate` fails | The error from `Validate`, before any geometry |
| A section, tube, bore or window sketch is not fully constrained | Error naming the stage, `verify:` and the status |
| A section has no four-edge profile | Error naming the station |
| The loft, sweep, cut, extrude or transform fails | The decad error wrapped with the stage, the gear or bore name, or the window facing |

## Acceptance tests

Run `go test ./models/screwgear`. The tests named here read the values of "Acceptance cases" in
`instructions.md`.

| Test | Assertion |
| --- | --- |
| `TestParamsDefaultsAreTheSpecTable` | Every default, and `Validate` returns no error |
| `TestParamsDerivedValues` | `β` 43.6°, `A` 13.85, `L` 178.5, travel 141.5, `Ri` 12.5, `Ro` 18.5, `hw` 7.7, `ht` 2.075, `c` 8.151, `sIn` 8.477, `sOut` 19.5, axial window 8.77 |
| `TestParamsValidateRangeChecks` | Each check of "Input checks and limits" fails with an error naming its input, and non-finite fields and an unset placement fail |
| `TestParamsValidateOrder` | An input that fails two checks reports the earlier |
| `TestParamsValidateSleeveBounds` | `CageRise` accepted at `A/2 + c + CollarWall + 1e-6` (18.08) and refused 2e-6 below; `RoofAllowance = 0` accepted |
| `TestFramePairAtTheDefaults` | Axis origins `(0, 0, ∓6.925)`, axis directions at `±40°`, `û` is `±Z`, the angle between the axes is `Σ`, gear B's phase is −1.31 |
| `TestFrameAngleAndSectionCoordinates` | `θ = s/Λ + Φ`, and the section-to-world map and its inverse agree |
| `TestFramePlacementCarriesThePair` | The same relations hold in world coordinates on a placed frame |
| `TestFrameBoreRoofsAtTheDefaults` | Roof sides, `hw` 7.7, opening 4.75 mm through, and `vLo`, `vHi` of ∓2.075 and ±2.675 |
| `TestSleeveChannelSeparationAtTheDefaults` | 5.310 mm, between gear A `−R` and gear B `−R` |
| `TestSleeveChannelSeparationShrinksWithTheWall` | A 0.55 mm clearance lowers the separation |
| `TestSleeveWindowFacingsFollowTheCrossingAngle` | `±k̂` at 80° and 90°, `±ê` at 100° |
| `TestWindowSearchAtTheDefaults` | Window numbers and corners of "Acceptance cases", facings `+k̂` and `−k̂` |
| `TestWindowSearchKeepsCollarWallToTheFlankingBores` | `CollarWall = 3.5` narrows the band by `√2` mm |
| `TestWindowSearchRoomRefusesTheBandWithNoWidth` | `CollarWall = 7` leaves no band |
| `TestBuildDefaultMechanism` | Default build: three solid bodies, ribbon volumes, cage volume 17,088 mm³, cage bounds |
| `TestBuildPlacementMovesTheWholeMechanism` | Placed cage bounds and volume |
| `TestBuildBoresFollowTheRibbonSurface` | `DistanceToPoint` from the cage to ribbon points at all four crossings: 0.19 ± 0.03 mm, and 0.48 ± 0.05 mm at the roof face |
| `TestBuildBranches` | No roof allowance, `Σ = 100°`, and mounting angles of ±5° each build with both windows cut |
| `TestBuildRejectsInvalidParams` | `Clearance = 0` fails with `clearance`; a nil context fails |

The default build takes about 40 s. The build tests use 16 teeth in place of 68, which takes about 15 s. `TestBuildBoresFollowTheRibbonSurface` states that decad reports a
lower bound of zero for these distances, so it checks the measured values and does not prove clearance.

Run `python3 scripts/check_repository.py` after the files exist.
