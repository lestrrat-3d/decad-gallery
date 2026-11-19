# Turbine fan Decad steps

Follow instructions.md for the API, the dimensions, the presets and the validation order. The package has seven files:
`params.go` (constants, `Hand`, `Params`, the presets, `Validate`, `Derive`, `Dimensions`), `size.go` (the rotor size
constants, `Size`, `SizeFor`, `Size.Beta`), `document.go`
(`NewDocument`), `blade.go` (the blade and its joint), `sketches.go` (the build run, the hub's outline sketch, the nut slot sketch and the tools) and
`turbinefan.go` (package doc, container names, `Parts`, `Model`, `BuildParts`, `Build`, the hub and the blades). Every error
message starts with `turbinefan: `.

No nut is embedded: the set screw's nut, the blade nuts and the adapter nuts sit in hex slots open to a face (AGENTS.md "Fasteners");
instructions.md "Fasteners" says why each slot works and why a back-face slot does not for the blade nuts.

## 1. Validate and derive (size.go, params.go)

`SizeFor(D_r)` (size.go) makes no geometry calls. With `s = D_r / 128`: `HubRadius = min(14 s, 18.9)`, `j = HubRadius / 14`,
`HubLength = 20 s`, `HubMid = HubLength / 2`, `TipRadius = D_r / 2`, `RootRadius = HubRadius + 0.4`, `RefRadius = 48 s`,
`HoldRadius = 21.5 s`, `BladeSocketStart = 5.7 j`, `BladeStudStart = BladeSocketStart + RegularFit + 0.005`, `BladeStudRadius = 3 j`,
`BladeSocketRadius = BladeStudRadius + RegularFit + 0.005`, `BladeStudEnd = 16 j`, `BladeBoltAt = 9.5 j`, `BladeBoltLength` = the longest
`m2bolt.StandardLengths()` entry `L` with
`HubLength - m2fastener.HeadPocketDepth - L >= BladeBoltTipFloor - 1e-9`, `BladeBoltCavity = BladeBoltLength - 0.4`.
Keep the subtraction order and apply the 1e-9 mm allowance only to this comparison.
`SizeFor(140.79999999999998)` must return `BladeBoltLength == 18`.
Implement `func (s Size) BladeSocketZ() float64 { return s.HubMid + 2.5 }`. The method is a derived
dimension in mm and adds no field to the public `Size` layout. Use it for the socket plane, assembled blade plane
and socket-to-slot wall estimate. Keep the physical hub bounds at [0, HubLength], with midpoint HubMid.
`SizeFor` fails for a diameter outside [128, 224] or not finite, when no length fits, and when `HubLength - BladeBoltCavity` is
above `BladeSocketZ() - BladeSocketRadius - 1`. `Size.Beta(r, pitch) = 5 deg + atan(tan(25 deg - 5 deg) RefRadius / max(r,
HoldRadius)) + (pitch - 25 deg)`.

Write Go documentation on the exported size constants, `Size` and its public fields, `SizeFor`, `Beta` and
`BladeSocketZ`. Use the local-frame dimensions in the “Size” table of instructions.md. Document the following meanings.

- `BaseRotorDiameter` sets the blade station table's reference size. `DefaultRotorDiameter` sets the preset size.
  `MinRotorDiameter` and `MaxRotorDiameter` bound the diameter inclusively. `BaseHubRadius` is the hub radius
  at the base size. `MaxHubRadius` leaves room for the 605ZZ clutch's ear bolt heads and a 1.5 mm hex key at the
  21.6 mm ear circle. `BladeBoltTipFloor` is the minimum blade-bolt tip height above the hub's back face.
- `Size` holds the derived local dimensions in mm, with dimensionless `Scale` and `HubScale`. Obtain it from
  `SizeFor`, which requires a finite diameter within the inclusive limits. Explain the formula and physical meaning
  beside each field. Distinguish the geometric midpoint `HubMid` from the socket-axis height `BladeSocketZ()`.
  Identify `RefRadius` as the rated-pitch station and `HoldRadius` as the inboard limit of twist.
- Identify the full circular stud's outer station at `14.6 * HubScale`, the legacy `BladeStudEnd` at
  `16 * HubScale`, and `BladeSocketStart` as the radial socket floor. `BladeBoltAt` is the radial bolt position.
  `BladeBoltCavity` measures from the bolt head's top face to the nut slot's near face.
- `Beta` returns the angle in radians from the plane of rotation at radius r in mm, with pitch in radians at
  `RefRadius`. State the inboard angle hold and the pitch adjustment to the blade sections.
  `BladeSocketZ` returns the assembled socket and stud axis height in mm from the hub's back face.

In `params.go`, document `RegularFit` as the 0.40 mm base allowance from crankhandle. Both the actual
radial stud clearance and axial stud-tip gap include the 0.005 mm addition in `SizeFor` and measure 0.405 mm.

Blade construction checks that the supplied dimensions match every field returned by `SizeFor`. Implement the private
method in `size.go` with exact Go struct equality.

```go
func (s Size) valid() bool {
	want, err := SizeFor(s.RotorDiameter)
	return err == nil && s == want
}
```

`Derive` makes no geometry calls. It runs the rules of instructions.md in order, stores `Size`, and fills the bore values
and the set screw: `BoreRadius`, `BoreFlat`, `BoreFlatWidth = 2 sqrt(r_b^2 - f_b^2)`, `BoreLoop`, `ShaftFlatWidth = 2
sqrt(d (D - d))`, `SetScrewTip = D/2 - d - c`, `SetScrewBolt` = the longest standard length `L` with `SetScrewTip + 2 + L <=
sqrt(HubRadius^2 - 1.9^2)`, `SetScrewHead = SetScrewTip + 2 + SetScrewBolt`, `CounterboreFloor = SetScrewHead - 2 - 0.4`.
`SetScrewSite` is `m2fastener.Site{Head: (0, -SetScrewHead, 4), Dir: (0, 1, 0), Corner: (0, 0, 1), Length: SetScrewBolt,
Cavity: SetScrewHead - f_b - 1.6 - 2}`. `BladeSites[k]` is `Head: BladeBoltAt u_k + (0, 0, HubLength), Dir: (0, 0, -1), Corner:
u_k, Length: BladeBoltLength, Cavity: BladeBoltCavity`. `AdapterSites[k]` (k = 0, 1, `u` at `EarBoltAngle(k+2)`, 30 and 150 degrees) is `Head: 9.5 u + (0, 0, -6.25), Dir: (0, 0,
1), Corner: u, Length: 10, Cavity: 7.75`, both validated; `Derive` fills it and the set screw's values in both mounts.

## 2. Sketches (sketches.go)

The private `run` and these helpers belong to `sketches.go`. The parts assembler shares the same run throughout
one build. Use the following field names and callable signatures.

```go
type run struct {
    ctx      context.Context
    doc      *decad.Document
    p        Params
    dm       Dimensions
    sketches []*sketch.Sketch
}

func across(z float64) r3.Frame
func newSketch(f r3.Frame) (*sketch.Sketch, error)
func (r *run) sketchOn(f r3.Frame) (*sketch.Sketch, error)
func (r *run) profile(s *sketch.Sketch, holes int) (*sketch.Profile, error)
func (r *run) extrudeOutline(s *sketch.Sketch, holes int, length float64) (*decad.Body, error)
func (r *run) hubSketch() (*sketch.Sketch, error)
func (r *run) nutSlotSketch(f r3.Frame, reach float64) (*sketch.Sketch, error)
func (r *run) counterboreTool() (*decad.Body, error)
func (r *run) setScrewSlotTool() (*decad.Body, error)
func (r *run) slotTool(site m2fastener.Site, u r3.Vec, reach float64) (*decad.Body, error)
```

`across` returns the plane at local z with U = +X and V = +Y. `sketchOn` creates and records a sketch.
`profile` solves it, verifies its constraints and selects a single valid profile with the requested hole count.
`extrudeOutline` uses that profile and extrudes along the plane normal. These run helpers record their driving
sketches. In `BuildParts`, record each hub cutter's driving sketch once, including cutters from
`m2fastener`, and append the fourteen construction sketches from the single blade before patterning it.

Each sketch is on a fresh `sketch.NewWorld()` plane made from an `r3.Frame`, recorded in the run, solved, and checked with
`sketchcheck.VerifyFullyConstrained` before its profile is used.

`hubSketch()` on the plane z = 0 (U = +X, V = +Y). Make every point with its exact initial coordinate.

- `disc = CreateCircle(O, HubRadius)` with `NewRadius`.
- The D bore: `qR = (w_b/2, -f_b)`, `qL = (-w_b/2, -f_b)`; `bore = CreateArc(O, qR, qL)` (counter-clockwise, the long way
  round over +Y); `flat = CreateLine(qL, qR)`; `NewRadius(bore, r_b)`, `NewHorizontal(flat)`,
  `NewDistancePointLine(O, flat, f_b)`.
- For each blade a circle of radius 1.3 whose centre, `BladeBoltAt` from the origin along `u_k`, is placed by
  `NewHorizontalDistance` and `NewVerticalDistance` from the origin, with `NewRadius`. Select the profile with four holes.

`nutSlotSketch(f, reach)` on the plane `f`, with `R = 2.51` and `h = R sqrt(3)/2`: points `A = (reach, -h)`, `B = (-R/2, -h)`,
`C = (-R, 0)`, `D = (-R/2, h)`, `E = (reach, h)` and the lines AB, BC, CD, DE, EA; `NewHorizontal(AB)`, `NewHorizontal(DE)`,
`NewVertical(EA)`, `NewDistancePointLine(O, line, h)` for AB, BC, CD and DE, `NewDistancePointLine(O, EA, reach)`,
`NewAngle(AB, BC, 60)` and `NewAngle(BC, CD, 60)`.

Tools, each recording its sketch:

1. `counterboreTool()`: a revolved meridian on the plane `r3.NewFrame((0, -rho_f, 4), U = (0, -1, 0), V = (1, 0, 0))`. Vertices
   (u, v) placed with `sketchcheck.DimensionPoint` and joined in order: `(0, 0)`; the nine round points `(rr + rr cos t_i,
   Rc - rr + rr sin t_i)`, `t_i = pi - (pi/2)(i/8)`, `rr = 0.5`, `Rc = 2.2`; `(a_out, Rc)`; `(a_out, 0)` with `a_out = HubRadius + 1 -
   rho_f`. Solve, verify, select the profile with no holes, then `decad.Revolve` about the axis from `(0, -rho_f, 4)` along
   `(0, -1, 0)` with `decad.FullRevolution{}`.
2. `setScrewSlotTool()`: `nutSlotSketch(r3.NewFrame((0, -(f_b + 1.6), 4), U = (0, 0, -1), V = (1, 0, 0)), 4 + 1)`, extruded 2
   (`NutCavityHeight`) along its normal, -Y. Then `decad.Fillet` of both caps' outer loops by `NutSlotRound`.
3. `slotTool(site, u, reach)` (formerly `bladeSlotTool(k)`): `nutSlotSketch(r3.NewFrame((site.Head.X, site.Head.Y, z), U = u,
   V = (-u.Y, u.X, 0)), reach)`, extruded 2 along +Z, with `z = site.Head.Z - site.CavityFar()` for a `Dir` -Z site and `z =
   site.Head.Z + site.Cavity` for a `Dir` +Z site. Blade k calls it with its site, `u_k` and reach `HubRadius + 1 - BladeBoltAt`, so the plane
   is z = `HubLength - BladeBoltCavity - 2` (2.4 at 128, 3.4 at 192). Adapter bolt k calls it with its site, its `Corner` and the reach `HubRadius + 1 - 9.5`, so the plane is z = 1.5. Then fillet the three edges along Z through the hexagon's corners B, C and D (C at 6.99 `u_k`), each picked by
   `decad.EndpointAt` at its stored position and joined with `.Or`, `Exactly(3)`, by `NutSlotRound`.

## 3. Parts (turbinefan.go)

Provide the following assembly entry points and local helpers in `turbinefan.go`. Use the `run` from “Sketches”.

Write a Go package comment beginning `Package turbinefan` immediately before the package clause. Describe
three replaceable blades, the 128–224 mm rotor range, the 192 mm default and the hub-radius cap for clutch bolt access.
Describe the D bore, the radial set screw seated in an open nut slot and the axial M2 bolts retaining the blades'
radial studs. In the default size these are M2 × 10 and M2 × 25 respectively. Describe the two common adapter
holes at 30° and 150°, the two M2 × 10 bolts and the 180° frame rotation for the clutch mount. The caller builds
the adapter and clutch. State the local frame: origin at the hub back face on the shaft, +Z toward the front,
−Y toward the shaft flat, first blade along +Y, and the other blades at 210° and 330°. Viewed from the front,
the right hand turns clockwise and the left hand counterclockwise to blow forward.

Give the following exported declarations Go documentation at their definitions.

- Document the body container constants `HubContainerName`, `BladesContainerName`, `BoltsContainerName`
  and `NutsContainerName`, whose literal values are `Hub`, `Blades`, `Bolts` and `Nuts` respectively.
- `Parts` contains the placed printed hub and three blades. Document `Sketches` as their 29 fully constrained
  construction sketches, `Dimensions` as the local dimensions, and `BladeSites`, `SetScrewSite` and `AdapterSites`
  as world-frame sites available in both mounts.
- `Model` embeds `Parts` and adds the three blade bolt and nut pairs. In `ShaftMount`, `SetScrew` and
  `SetScrewNut` are present and the entries in `AdapterBolts` and `AdapterNuts` are nil. In `ClutchMount`,
  both adapter pairs are present and the set screw fields are nil. Fastener models own their construction sketches.
- `BuildParts` adds and returns the placed hub and three blades. The caller controls their containers.
  `Build` files the complete fan including its mount-specific hardware under the four named containers.
  State on each function that it requires a nonnil context and document. State on `Build` that the caller
  builds the clutch and adapter separately.

```go
type Parts struct {
    Hub          *decad.Body
    Blades       [BladeCount]*decad.Body
    Sketches     []*sketch.Sketch
    Dimensions   Dimensions
    BladeSites   [BladeCount]m2fastener.Site
    SetScrewSite m2fastener.Site
    AdapterSites [2]m2fastener.Site
}

type Model struct {
    Parts
    BladeBolts  [BladeCount]*m2bolt.Model
    BladeNuts   [BladeCount]*m2nut.Model
    SetScrew    *m2bolt.Model
    SetScrewNut *m2nut.Model
    AdapterBolts [2]*m2bolt.Model
    AdapterNuts  [2]*m2nut.Model
}

func BuildParts(ctx context.Context, doc *decad.Document, p Params) (*Parts, error)
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error)
func (r *run) hub() (*decad.Body, error)
func (r *run) socketTools() ([]*decad.Body, error)
func (r *run) bladeSlotTool(k int) (*decad.Body, error)
func (r *run) ring(body *decad.Body) ([]*decad.Body, error)
func (r *run) blades() ([]*decad.Body, error)
func placeSite(s m2fastener.Site, f *r3.Frame) (m2fastener.Site, error)
```

`BuildParts` calls `p.Derive()` for the dimensions and `p.buildFrame(dm.Size)` for the optional placement.
Initialize `run{ctx: ctx, doc: doc, p: p, dm: dm}`. `bladeSlotTool(k)` takes a one-based blade index.
`ring` returns the input body first, then its 120° and 240° copies. `socketTools` returns that sequence for
the rounded cutter, and `blades` returns it for blade 1 after assembly placement.
`BuildParts` verifies and places each solid using `placement.Place(ctx, body, frame)`. It returns the
placed hub, three placed blades, collected sketches, dimensions and sites. `placeSite` with a nil frame
returns the original site. With a frame it transforms `Head` as a point and `Dir` and `Corner` as directions.

For hub finishing use
`edgefinish.Chamfer(ctx, body, normal, offsets, edgefinish.Long(edgefinish.All), ChamferSize)`;
it returns the new body, matched edge count and error. Both `m2fastener.PocketTool(ctx, doc, site)` and
`m2fastener.HoleTool(ctx, doc, site)` return the cutter, its driving sketch and an error.

1. Hub: extrude the outline `HubLength` along +Z (profile with four holes); `edgefinish.Chamfer` of both faces by 0.5 over the long
   loops; then `decad.Cut`, in this order, the socket tool and its two pattern copies, for each blade site `m2fastener.PocketTool`
   then `bladeSlotTool` (a call of `slotTool`), then `counterboreTool`, `m2fastener.HoleTool(SetScrewSite)` and `setScrewSlotTool`, then for each adapter site
   `m2fastener.HoleTool(AdapterSites[k])` and `slotTool(AdapterSites[k], Corner, HubRadius + 1 - 9.5)`. The socket tool is
   `BladeSocketTool` on `r3.NewFrame((0, BladeSocketStart, BladeSocketZ()), (0, 0, 1), (1, 0, 0))`; its copies come from
   `decad.PatternCopies` with `decad.CircularPattern{Axis: (0, 0, 1), Count: 3}`. Record every tool's sketch.
2. Blades: `BuildBlade(ctx, doc, BladePitch, Size)` (`BuildBladeMirrored` when `Hand` is `LeftHand`); record its
   sketches; `decad.Transform` by `r3.FromFrame(r3.NewFrame((0, 0, BladeSocketZ()), (-1, 0, 0), (0, 0, 1)))`; `decad.PatternCopies` with
   the same circular pattern. A blade-local point (x, y, z) on blade 1 becomes
   (−x, z, BladeSocketZ() + y). Apply this placement once, after the individual blade and its bolt hole are complete.
   The axis of that hole remains coaxial with the axial hub bolt. Build the head pockets and nut slots at their
   specified hub-face depths. Check the stud in its socket at `BladeSocketZ()`, including socket corner sections at
   `BladeSocketZ() ± 3*HubScale`. Retain the separate 30%, 50% and 70% hub-height clearance checks.
3. Require each part solid, then `placement.Place(ctx, body, f)`, where `f` is `Placement` moved back from the placement point
   to the local origin: origin `Placement.Apply((0, 0, -anchor))` with `Placement`'s U and V (anchor `HubLength` in `ShaftMount`,
   `ClutchAnchorZ` in `ClutchMount`; `f` is nil when `Placement` is nil). Carry each site's `Head` through `f` with
   `r3.FromFrame(*f).Apply`, and `Dir` and `Corner` through `ApplyDir`.

`Build` then calls `m2fastener.PlaceBolt` and `PlaceNut` at each placed blade site and, in `ShaftMount`, at the set screw site
or, in `ClutchMount`, at each placed adapter site, and adds each body to `doc.Root().ChildNamed(name)` with `AddBody`: the hub
in `Hub`, the blades in `Blades`, blade bolts 1 to 3 then the set screw or adapter bolts 1 and 2 in `Bolts`, and their nuts in
the same order in `Nuts`. Return the 29 `Parts.Sketches` collected by `BuildParts` (15 hub outline and cutter
sketches plus 14 blade sketches). Each placed bolt owns its `Profile` and `Socket` sketches, and each placed
nut owns its `Hexagon`, `LowerChamfer` and `UpperChamfer` sketches. Verify these through their respective
fastener fields, separately from `Parts.Sketches`: 20 in `ShaftMount` and 25 in `ClutchMount`.

## 4. Blade (blade.go)

Generate the right-hand blade and stud at the requested pitch as one solid with
`decad.LoftSectionsWithCircularCorrespondence`. Return the four circular sketches and all airfoil sketches in
ascending radial height, followed by the bolt-hole sketch. The nine airfoil stations give fourteen sketches per blade. Validate pitch and size before creating bodies. Include the station height and operation in errors.

### Package declarations and return values

Use these package-level declarations in `package turbinefan`. All Go pitch values are radians.

```go
const (
    designAttack          = 5 * math.Pi / 180
    refPitch              = 25 * math.Pi / 180
    DefaultBladePitch     = refPitch
    MinBladePitch         = 10 * math.Pi / 180
    MaxBladePitch         = 40 * math.Pi / 180
    BladeSocketFloorRound = 0.5
)

type Blade struct {
    Body     *decad.Body
    Sketches []*sketch.Sketch
}

type bladeStation struct{ r, chord, ratio, camber, ellipse float64 }
```

`size.go` owns `func (s Size) Beta(r, pitch float64) float64` and uses the unexported constants above.
Both builders take the following signatures and have the same two-result contract.

```go
func BuildBlade(ctx context.Context, doc *decad.Document, pitch float64, size Size) (*Blade, error)
func BuildBladeMirrored(ctx context.Context, doc *decad.Document, pitch float64, size Size) (*Blade, error)
```

On success return `&Blade{Body: body, Sketches: sketches}, nil`. On error return `nil, err`. Require a nonnil context and
document, a finite pitch within the inclusive radian limits, and a size equal to the result of `SizeFor`. `body` is the final drilled solid
in the requested hand. `sketches` contains its thirteen section sketches in radial order and its bore sketch last.
The assembly appends the sketches once, places the body and patterns that placed body into its other blade positions.
The socket cutter has the distinct three-result signature given under “Socket cutter”.

Provide `func stationsFor(size Size) []bladeStation` for the original source sizing table used by the package's
`export_test.go`. Its first two rows are
`{size.RootRadius, 8*size.HubScale, 1, 0, 1}` and
`{size.RootRadius + 1.9*size.HubScale, 8*size.HubScale, 1, 0, 1}`.
Append the eight original “Airfoil stations and print face” rows in source order. Multiply each source radius
and chord by `size.Scale` and retain its thickness ratio, camber and ellipse weight. The eight source rows also
drive construction of the working sections. Apply the physical radial mapping and add the rated-radius section
when constructing the loft. A test that measures thickness at physical radius z must compare with the table at
its corresponding source coordinate, or interpolate directly between the actual working stations.

### Socket cutter

Implement `BladeSocketTool(ctx, doc, f, size) (*decad.Body, *sketch.Sketch, error)` for the supplied frame f.
Reject an invalid size. Create a fresh sketch world, a plane from f, and a sketch on that plane. Create a circle
at the sketch origin with radius `size.BladeSocketRadius`. Constrain it with `sketch.NewRadius` at that radius,
solve it, and require `sketchcheck.VerifyFullyConstrained(ctx, s)`. Require exactly one valid profile with no holes.

Extrude that profile along the positive frame normal. The distance is
`size.HubRadius + 1 - size.BladeSocketStart` mm. The given hub frame points radially outward, so the cutter
starts at the socket floor and ends 1 mm beyond the hub radius. For the 128 mm rotor this distance is 9.3 mm.
Use these calls, checking each error before the next operation.

```go
body, err := decad.Extrude(ctx, doc, s, profiles[0], decad.Distance{
    D:   units.Millimeters(size.HubRadius + 1 - size.BladeSocketStart),
    Dir: decad.Along,
})
// After checking err, select the single circle at the extrusion's start cap.
floorRim := decad.Edges(decad.OuterLoopOf(decad.CapStart(body))).Exactly(1)
body, err = decad.Fillet(ctx, body, floorRim, units.Millimeters(BladeSocketFloorRound))
```

`BladeSocketFloorRound` is 0.5 mm. Return the filleted cutter and its original sketch. The hub builder makes
the two circular-pattern copies and subtracts the three cutters.

### Circular sections

Use “Stud, socket and loaded root” for the dimensions. Give each section plane world +X and +Y axes and the same
physical XY origin O. Try (15j/8, 7j/4) first and (53j/32, 61j/32) second. A rational value is representable
when rounding to binary64 and converting the result back to an exact rational gives the original value.

1. Lift the actual binary64 fields R and S and the computed joint scale j into exact rationals. Start the axial search
   at the nearest binary64 value to S + 3/10, rounded to nearest with ties to even. Include the next four binary64
   values on each side using `math.Nextafter`. Order all nine by exact distance from S + 3/10, then by smaller value.
2. For each candidate Z1, compute c = Z1 − S and r0 = R − c exactly. Require 0 < c < R and representable r0.
   Store the accepted r0 and Z1 directly; recomputing c from a decimal chamfer constant can change the axial slope.
3. For each coordinate of the current pole seed, include its nearest binary64 value and eight successive
   neighbours in each direction. Order by exact distance to its nominal value, then by smaller value. Pick the
   first Ox or Oy for which Oi + R, Oi − R, Oi + r0 and Oi − r0 are all representable. If either search has
   no result, try the next axial candidate. Accept the first pair with Ox² + Oy² < r0². When none qualifies, return
   an error with S and R. Verify the accepted tuple on each build.
4. Subtract O from physical XY coordinates when authoring sketches. Keep the physical stud axis at (0,0).
   Verify the solved centre, endpoints, radii and equal axial and radial chamfer legs as exact rational values.
   For compact dimensions Z1 is approximately 6.405 mm. Use the accepted binary64 value for the chamfer rim.
5. Try each origin directly in the caller's document. A polar-angle refusal occurs before Decad registers a body.
   Retry that failure with the next origin and the same physical geometry. Propagate other errors. Keep the first
   admitted loft and cut its bolt bore once.
6. If both initial origins fail, construct half-planes from the solved world-coordinate sections and the loft
   interpolation. Sample u = 0, 1/8, …, 1 and h = 0, 1/8, 1/4, 1/2, 3/4, 7/8, 1. For a point C with nonzero
   perimeter tangent D, keep origins satisfying det(C−O,D)/|D| ≥ m at margin m. Start by clipping a regular
   96-gon of radius r0 at zero margin. Report the failing band and span if the polygon is empty.
   Bisect the margin between 0 and 1 mm fourteen times, clipping the 96-gon of radius r0−m each time.
   Set the lower bound to m when the clipped polygon is nonempty, and the upper bound to m otherwise. At 90% of the
   final lower bound, average the remaining vertices to get the third origin. Quantize it as above and require
   exact loft admission. Report the failing band and span when admission fails.

Draw east, north, west and south at distance r from the stud centre. Dimension the centre and east point from the
sketch origin. Give the north and south points the centre's horizontal coordinate and give west its vertical
coordinate. Draw the first three CCW quarter arcs about the common centre. For the south-to-east arc create a
second centre at the same coordinates and constrain only its horizontal coordinate. The arc's own radius equation
sets its vertical coordinate. This closes the loop with four equal radii and four independent arc equations.
Require all four solved arcs to have the same centre and radius, the expected cardinal endpoints, and a 90° sweep.

### Airfoil sections

1. Build the eight source stations and the additional physical rated-radius station in “Airfoil stations and print face”. For each section generate
   21 reference points. Start at the suction-side trailing edge, follow ten cosine-spaced chord positions
   `x_k = (1 − cos(kπ/10)) / 2`, pass the leading edge once and return along the pressure side. Evaluate `x = 1`
   as `1 − 10⁻⁸` before setting the flat endpoints. Construct the mean line using
   `yc(x)=m(2px−x²)/p²` for `x<p`, and `yc(x)=m(1−2p+2px−x²)/(1−p)²` otherwise, with `p=0.4`.
   For the signed half-thickness in the instructions, let `e = 0.4 + 0.1 w`, `a = c(e−x)`,
   `b = c(yc(x)−yc(e)) ± halfThickness`. At its physical station height, the canonical right-hand point is
   `(−a cos β + b sin β, −a sin β − b cos β)`. Generate this section once in the canonical hand.
2. Translate each section so the midpoint of its two trailing endpoints is `(R,Yt)` for the right hand.
   Set `Yt = 4 j sin(β(RootRadius, BladePitch))`, the trailing midpoint of the round reference section.
   For each opposed pair in the last 20% of the chord, retain the midpoint M.
   With D = (suction − pressure)/2, t = (x−0.8)/0.2 and h = 3t²−2t³, write
   `M ± ((1−h)D + h(0,−1 mm))`. Set the two trailing endpoint coordinates to the bed X and `Yt ± 1 mm` exactly.
3. For the first airfoil station round the transition to the flat. Number the 21 reference points
   from 0 at the suction-side end to 20 at the pressure-side end. On the suction side blend from 0 to 7 using point
   8 for the outgoing direction. On the pressure side blend from 20 to 16 using 15. For each side, write the
   intermediate reference points at uniformly spaced parameters on a quadratic Bézier with controls
   `endpoint, join + 2(join − next), join`. Limit the middle control to at least 1.2 mm inside the bed plane.
   At points 16–19 on the pressure side add `0.45 mm sin²(π(i−15)/5)` to Y. Keep these lengths fixed across rotor sizes.
4. Build one clamped, uniform cubic B-spline with 23 control points and 20 nonempty spans using `CreateSpline`.
   Form the natural chord-length cubic through those points. Represent each of its 20 source pieces as a cubic in
   its own parameter in [0,1]. Use a clamped uniform knot vector with interior knots at binary64 values j/20,
   j = 1…19. Set the two end controls to the source endpoints. Set each adjacent control from the corresponding
   source endpoint derivative divided by three, adding at the start and subtracting at the end. Solve the 19
   collocation equations at the interior knots for the remaining 19 controls so the destination curve passes through
   the interior reference points. Solve each coordinate with rational coefficients and round each final coordinate
   once to binary64. Generate the controls from each section's reference points. Author the spline from suction to
   pressure and verify its endpoints and edge tangents.
5. Make the two interior trailing-line joints at one and two thirds of the distance from the suction to the
   pressure endpoint. Use three whole lines to close the spline. Solve every sketch, require `FullyConstrained`,
   zero DOF and empty redundancy/conflict lists, and take the single valid profile with four outer edges and no holes.

### Loft, placement and checks

Use the explicit circular correspondence with `EmergingFlat = true` and `SmoothCircularBands = true`. On each
of the four circular quadrants set the six knot numerators to [0, 12, 24, 41, 56, 80] and the denominator to 80.
In ascending circular order, match each quadrant to five consecutive working spline spans in the CCW profile
traversal. Preserve the actual recorded spline knots. Match each of the three trailing lines to the same physical
+X tangent point on the last cylinder. The flat spans have zero width on that ring and positive width above it.
Those three planar patches produce the growing strip specified under “Stud, socket and loaded root”.

Let Decad certify the exact circular and spline section records through
`LoftSectionsWithCircularCorrespondence`. Propagate admission errors with the band and span; a sampled scan cannot
replace the admission call. For each regular patch, the XY boundary measured from O is C = ((1−h) A + h B)/W
with positive common denominator W. Decad bounds the determinant det(C, ∂C/∂u) above zero on the closed unit
square. Its numerator has three Bernstein rows in h: det(A,A′), [det(A,B′)+det(B,A′)]/2 and det(B,B′).
Decad converts them to a common Bernstein degree in u, subdivides in u and h at one half, and accepts a cell
only when every coefficient is positive. A nonpositive corner or unresolved proof refuses the loft.
The birth patches have a single specified source point and lie in X = R. Their target endpoints must ascend in Y;
the exact positivity condition there is (R−Ox)(Yhi−Ylo) > 0. Width vanishes only at the birth. Preserve that
single shared apex on freezing and triangulation.

As a geometry check, compare adjacent endpoint positions exactly, and their tangents at corresponding u = 1 and 0.
On the 19 regular working joins require determinant zero and positive dot product for all h. On the two collinear
flat joins require these conditions for h > 0. At the two true flat endpoints require an outward turn for h > 0
and the common tangent limit at birth. Use the actual frozen surface derivatives for the loaded-root check.
With S(u,t) in blade coordinates, choose outward N = Su × St, and let I be the first fundamental form and
J the matrix with entries N·Suu, N·Sut, N·Stt. Prove |N| I − R J positive semidefinite on each regular loaded-root
patch and check all adjoining creases separately. A strict positive lower bound also suffices.

On a canonical near-bed cut x ≥ R − 0.01, isolate its regular side branches and bound their signed planar curvature
on the entire arc, including its limits at the apex. For a branch y(z), the absolute curvature is
|y″|/(1+y′²)^(3/2). The continuous geometry test requires a bound below 1.9/mm through the root and outboard intervals.
At x = R, check the planar footprint and any zero-area tangent generators separately. For each variant run
`cornercheck.CheckAt` along the actual bed normal and retain all findings outside proved geometry-specific
allowances. An insertion-chamfer hit must match its Tip kind, proven slice height, axial station and rim;
a position-only exemption could also cover a distinct inside corner. Freeze and tessellate the body at 0.05 mm
with `VerifyAll` and require boundary and volume verification.

Keep the completed loft in its authored frame. Changing section pitch leaves the bed, stud and hole axes fixed.
Create the verified 1.3 mm circle on frame origin
`(0,−(R+2),BladeBoltAt)`, U = +Z and V = +X. Extrude by `2(R+2)` along +Y and cut the bore. For the opposite hand
reflect the completed drilled right-hand body in local YZ. Verify the handed chord and the fixed physical bolt axis.
Build the matching hub cutter as specified under “Socket cutter”.

Place and copy the resulting blade as in “Parts”. Check the 0.05 mm verified mesh, bed-triangle area, full strip
width and extent, and absence of material below the bed. The nominal contact is 87.4 mm² on the compact rotor.
For each new case, check the actual full-circle stud and socket fit, the bolt clearance, chord pitch at the rated
radius, thickness, axial envelope, removal paths and the consuming hand-crank fan.

## 5. Clutch mount

`Mount = ClutchMount` changes only what `Build` adds. The hub, the blades and their sketches are the `ShaftMount` ones
(sections 1 to 3), so the hub has the set screw's cuts and the two adapter holes and slots in both mounts. See
instructions.md "Clutch mount" for the numbers.

- `MountZ` is 26.25 (`onewayclutch.AdapterTo + ClutchSeam`). `MountFrame` returns the placement point: origin (0, 0,
  `onewayclutch.CoverTo`) in the clutch frame, turned `ClutchTurn` (180 degrees) about Z. `ClutchAnchorZ` is `CoverTo - MountZ` = -7.75.
- `ParamsForClutch(c, opts...)` takes `WithBladePitch(pitch)` and `WithRotorDiameter(d)` to replace the defaults, then runs
  `Validate`.
- `BuildParts` places `SetScrewSite` and both `AdapterSites` in both modes. `Build` in `ClutchMount` places an M2 x 10 bolt and
  nut at each adapter site and no set screw. Bodies: 14. The build returns 29 sketches from `BuildParts`, as in `ShaftMount`.
- `Build` does not build the adapter or the clutch. The caller builds them with `onewayclutch.Build` and
  `onewayclutch.BuildAdapter` in the clutch's own `onewayclutch.NewDocument()`, and the fan in its own document, because both
  models file their hub in a container named `Hub`. The adapter's `FaceSites[1]` and `FaceSites[2]` are the fan's
  `AdapterSites[0]` and `[1]` through `MountFrame` and the anchor offset; `FaceSites[0]` stays empty.
