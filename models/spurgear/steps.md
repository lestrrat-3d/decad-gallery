# Spur gear Decad construction steps

Generate `models/spurgear/*.go` from this recipe and [instructions.md](instructions.md).
Use dependency versions in repository `go.mod`, `internal/sketchcheck`, and upstream
public APIs. Do not require existing spur source to interpret a step. The contract
owns formulas, defaults, numeric input guards, and case inputs; use its named sections.

## Package and API

Generate package `spurgear` at
`github.com/lestrrat-3d/decad-gallery/models/spurgear`.

| File | Responsibility |
| --- | --- |
| `involute.go` | Dimension derivation, planar rotation, involute sampling |
| `spurgear.go` | Parameters, profiles, root disc, one tooth loft, pattern and join for both branches, root fillet, bore, cap chamfer |
| `tooth_section.go` | Reusable ordinary tooth section |
| `spurgear_test.go` | Public body and finishing acceptance tests |
| `spurgear_profile_test.go` | Internal profile, root contact, and constraint tests |
| `tooth_section_test.go` | Public signed tooth-section tests |
| `placement_test.go` | Public placed-gear tests |

Declare `Params` using the contract's Inputs table. Expose these signatures:

```go
func DefaultParams() Params
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error)
func BuildProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
    p Params) (*sketch.Sketch, *sketch.Profile, error)
func BuildToothSection(ctx context.Context, world *sketch.World, plane *sketch.Plane,
    p Params, angle float64) (*sketch.Sketch, *sketch.Profile, error)
func InvoluteFlanks(pitch, base, tip, effectiveTeeth float64, samples int,
    angle float64) ([][2]float64, [][2]float64, error)
func CutThroughBore(ctx context.Context, doc *decad.Document, body *decad.Body,
    p Params) (*decad.Body, error)
func ChamferOuterCaps(ctx context.Context, gear *decad.Body,
    distance float64) (*decad.Body, error)
func (p Params) RejectPlacement(model string) error
```

`Params` ends with `Placement *r3.Frame`; nil selects the world frame. Add a
private `placement(p Params) (r3.Frame, error)` returning `*p.Placement` or
`r3.NewFrame(r3.NewVec(0,0,0),r3.NewVec(1,0,0),r3.NewVec(0,1,0))`, a private
`shifted(f r3.Frame, dist float64) (r3.Frame, error)` returning
`r3.NewFrame(f.Origin().Add(f.N().Scale(dist)),f.U(),f.V())`, and a private
`toWorld(f r3.Frame, xy point) r3.Vec` returning
`f.Origin().Add(f.U().Scale(xy.x)).Add(f.V().Scale(xy.y))`.

- Return left and right base-to-tip arrays, respectively, from `InvoluteFlanks`.
- Return solved, verified sketches and their profiles from both profile APIs.
- Require nonnil context/document for `Build` and nonnil context/world/plane for
  both profile APIs. Return errors before construction when these guards fail.
- Apply Input checks and limits through one private parameter validator for
  `Build`, `BuildProfile`, and `BuildToothSection`. It rejects a non-nil
  `Placement` whose `IsValid()` is false.
- `RejectPlacement` returns `fmt.Errorf("%s does not support Placement", model)`
  when `p.Placement != nil`, otherwise nil.
- For `BuildToothSection`, separately reject nonfinite angles and embedded flanks. The embedded-flank error text contains `embedded`.
- Validate `InvoluteFlanks` arguments according to Reusable constructions.
- Treat `CutThroughBore` and `ChamferOuterCaps` as finishing helpers with the
  preconditions in Reusable constructions; they do not invoke the gear validator.
- Wrap operation errors with `%w` and stage names. Include tooth index for
  flank failures and actual/expected counts for topology failures.

## Internal test interfaces

Preserve these package-private names so existing independent tests compile:

```go
type point struct { x, y float64 }
type dimensions struct { pitch, base, root, tip, fillet float64 }
func derive(p Params) dimensions
func buildGearProfile(ctx context.Context, world *sketch.World, plane *sketch.Plane,
    p Params, d dimensions) (*sketch.Sketch, *sketch.Profile, []point, error)
func buildBoreProfile(ctx context.Context, world *sketch.World, p Params,
    margin float64) (*sketch.Sketch, *sketch.Profile, error)
func filletRoots(ctx context.Context, body *decad.Body, frame r3.Frame, roots []point,
    radius float64) (*decad.Body, error)
func filletRadius(p Params, d dimensions) float64
```

`filletRoots` is the ordinary-branch root-matching fillet of Step 7. The embedded
fillet is a separate private helper.

Return actual root contact coordinates from `buildGearProfile`, including exact
trim contacts in the embedded branch. Keep the bore sketch separately inspectable
through `buildBoreProfile`. Other private helper names are implementation choices.

## Step 0 — parameter table

Implement a private `defineParameters(t *param.Table, p Params) error`
(`github.com/lestrrat-3d/sketch/param`). It writes the Parameter table of
`instructions.md` with exactly these calls, in this order, each with string
literals only, no concatenation, no loops and no variables for names or
expressions. Check every error and wrap it with the parameter name:

```go
t.SetValue("module", units.Millimeters(p.Module))
t.SetNumber("teeth", float64(p.Teeth))
t.SetValue("pressure_angle", units.Radians(p.PressureAngle))
t.SetValue("thickness", units.Millimeters(p.Thickness))
t.SetNumber("samples", float64(p.Samples))
t.SetValue("bore_diameter", units.Millimeters(p.BoreDiameter))
t.SetValue("cap_chamfer", units.Millimeters(p.CapChamfer))
t.SetExpr("tool_margin", "1", units.Millimeter)
t.SetExpr("pitch_radius", "module * teeth / 2", units.Millimeter)
t.SetExpr("base_radius", "pitch_radius * cos(pressure_angle)", units.Millimeter)
t.SetExpr("root_radius", "(module * teeth - 2.5 * module) / 2", units.Millimeter)
t.SetExpr("tip_radius", "(module * teeth + 2 * module) / 2", units.Millimeter)
t.SetExpr("involute_angle", "tan(pressure_angle) - deg(pressure_angle) * pi / 180", units.One)
t.SetExpr("fillet_radius", "0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2", units.Millimeter)
t.SetExpr("bore_tool_length", "thickness + 2 * tool_margin", units.Millimeter)
```

Implement `readDimensions(doc *decad.Document) (dimensions, error)` that reads
`pitch_radius`, `base_radius`, `root_radius`, `tip_radius` and `fillet_radius`
with `doc.Param(name)` and keeps each `Value.Base()` in `dimensions`
(`pitch`, `base`, `root`, `tip`, `fillet`). `build`-side code reads counts with
`doc.ParamInt("teeth")` and `doc.ParamInt("samples")`.

## Step 1 — derive points

- Implement `derive(p Params) dimensions` on a scratch document: `decad.New()`,
  `defineParameters(doc.Params(), p)`, then `readDimensions(doc)`. It returns a
  zero `dimensions` when either call fails. Implement `filletRadius(p, d)` as
  returning `d.fillet`.
- Implement Tooth flank using `math` functions. Convert counts before division;
  preserve both endpoint samples and the stated mirror/rotation order.
- Use one internal sampler accepting real-valued effective teeth. Whole-gear
  sampling supplies `float64(p.Teeth)`; `InvoluteFlanks` exports `[2]float64` pairs.
- Retain all original samples even when the root circle intersects a flank.

## Step 2 — constrain sketch points

Reuse these existing `internal/sketchcheck` helpers:

| Helper | Required behavior |
| --- | --- |
| `DimensionPoint(s,x,y)` | `s.CreatePoint(x,y)` and signed `NewHorizontalDistance(s.Origin(),point,x)` plus `NewVerticalDistance(s.Origin(),point,y)` |
| `DimensionArcEnd(s,x,y)` | Create point and add only the dimension selected by Sketch constraints |
| `VerifyFullyConstrained(ctx,s)` | Enforce every report condition in Sketch constraints and return `report.Check()` |

Use `s.Origin()` as center. Apply these helpers to the exact point classes in
Sketch constraints. Add circle constraints with
`s.AddConstraint(sketch.NewRadius(circle,radius))` or
`s.AddConstraint(sketch.NewDiameter(circle,diameter))`. Reuse endpoint pointers;
do not add coincident constraints to points that already are identical.

Call `s.Solve(ctx)` and check its error, then
`sketchcheck.VerifyFullyConstrained(ctx,s)`. Only then read `s.Profiles()` or
call `s.UnionProfiles(...)`. Repeat for every sketch, including tool sketches.

## Step 3 — build an ordinary outline

Implement `buildGearProfile`. Dispatch to Step 4 when `d.base < d.root`; otherwise:

1. Call `world.CreateSketch(plane)` and use its origin as center.
2. Iterate teeth in ascending index. Sample the rotated left/right flank arrays.
3. Create dimensioned sample points according to Step 2. Create both root
   points by scaling their base samples according to Ordinary profile; compute
   each coordinate as `x*d.root/d.base` (multiply, then divide), never with a
   precomputed `d.root/d.base` factor. Create the left root before the right root.
4. Save right root then left root coordinates for each tooth in a `2N`-entry list.
5. Except on tooth zero, call `s.CreateArc(center,previousLeftRoot,rightRoot)`.
   Save tooth zero's right root for final closure.
6. Call `s.CreateLine(rightRoot,rightPoints[0])`.
7. Call `s.CreateFitSpline(rightPoints...)` and check its error.
8. Call `s.CreateArc(center,rightPoints[S-1],leftPoints[S-1])` for the tip.
9. Call `s.CreateFitSpline(leftPoints...)` and check its error. Preserve its
   base-to-tip parameterization; the profile traverses this side in reverse.
10. Call `s.CreateLine(leftPoints[0],leftRoot)` and save the left root.
11. After the loop, call `s.CreateArc(center,lastLeftRoot,firstRightRoot)`.
12. Solve and verify. Require exactly one valid profile from `s.Profiles()`.

Test curve counts from Ordinary profile. Do not round sketch corners; Step 7
applies native solid fillets after lofting.

## Step 4 — build an embedded outline

1. Create a sketch on the requested plane. Draw every tooth's two original
   fit splines and tip arc with the samples and constraints from Step 3.
2. Omit radial lines and individual root arcs. Call
   `s.CreateCircle(s.Origin(),d.root)` and add its radius constraint.
3. Solve and verify before extracting regions.
4. Read all `s.Profiles()`. Reject invalid regions; collect every region index.
5. Call `s.UnionProfiles(indices...)` and check its error.
6. Require `profile.Valid`, no `profile.Holes`, and the boundary count from
   Embedded profile. Return this union profile rather than an arbitrary region.
7. For each `profile.Outer` edge whose `Entity` is `*sketch.FitSpline`, require
   `edge.Partial` and `edge.TExact`.
8. Evaluate `flank.Eval(edge.TStart)` on the original base-to-tip spline. Require
   `abs(hypot(x,y)-d.root) <= 1e-8` and save this actual root contact.
9. Require exactly `2*p.Teeth` saved contacts; return sketch, profile, and contacts.

Use native trim parameters, not polyline endpoints or reconstructed intersections.
If certified trimming fails, follow AGENTS.md Agent coordination with the real case.

## Step 4b — build an embedded tooth

Implement an internal `buildEmbeddedTooth(ctx,world,plane,p,d)` for the embedded
branch, with its closing-radius helper:

1. Compute `Rc` from the Embedded tooth rule in `instructions.md`. Evaluate
   the half-angle `h(r)` with the Tooth flank formulas: rotate the mirrored
   involute point at radius `r` by the same `beta` as the sampler and take
   `math.Atan2` of the rotated point. Bisect `[d.base,d.root]` for `Rq` with a
   fixed 200 iterations. Return an error naming the root circle when no `Rc`
   exists.
2. Sample the flanks from `Rc` to `d.tip` with the same involute and `beta` as
   `sampleFlanks`, at angle 0, with `p.Samples` samples.
3. Create every right point with both coordinate dimensions. Create every left
   point with both dimensions except the first and the last, which receive one
   coordinate dimension through `sketchcheck.DimensionArcEnd`.
4. Create, in order: the right fit spline, the arc `s.CreateArc(center,rightTip,
   leftTip)`, the left fit spline, and `s.CreateArc(center,rightFirst,leftFirst)`.
   Reuse the same point objects at every junction.
5. Solve and verify, require one valid profile and exactly four outer curves,
   and return sketch and profile.

## Step 5 — build a standalone tooth

Implement `BuildToothSection` after its API guards and ordinary-branch check:

- Sample one tooth at the supplied angle; apply no additional tooth-index rotation.
- Create the same dimensioned flank/root points as Step 3.
- Create right radial line, right spline, right-to-left tip arc, left spline,
  and left radial line in the same order.
- Close with `s.CreateArc(center,rightRoot,leftRoot)`. The bounded region uses
  this short arc in reverse; do not use the complementary long root arc.
- Solve and verify. Require one valid profile and exactly six outer curves.
- Return sketch/profile for the caller's loft; apply no solid features.

## Step 6 — build the gear

Implement `Build` using `sketch.NewWorld()`. After `validate`, create the
world and call `defineParameters` on both `doc.Params()` and `world.Params()`
(a sketch only binds to its world's table), then `readDimensions(doc)`, and read
the tooth count with `doc.ParamInt("teeth")`. Both branches then start the same way:

1. Resolve the placement frame. For the lower plane, use `world.XY()` when
   `p.Placement` is nil; otherwise call `world.CreatePlaneFromFrame` with the
   placement frame.
2. Create the upper frame with `shifted(placement, p.Thickness)`.
3. Call `world.CreatePlaneFromFrame(frame)` for the upper plane.

Ordinary branch (`d.base >= d.root`):

4. Build the root disc. Create a sketch on the lower plane, call
   `s.CreateCircle(s.Origin(),d.root)`, and add `sketch.NewRadius(circle,d.root)`;
   bind that radius with `s.Bind(radius,world.Params(),"root_radius")`.
   Solve and verify the sketch, require one valid profile, and call
   `decad.Extrude(ctx,doc,s,profile,decad.Distance{D: thickness,
   Dir: decad.Along})` with `thickness` from `doc.Param("thickness")`.
5. Build the tooth section on the lower plane with angle 0 and on the upper plane
   with angle 0 (the Step 5 construction, without re-validating `p`). Call
   `decad.Loft(ctx,doc,lowerSketch,lowerProfile,upperSketch,upperProfile)` and keep
   the returned tooth body.
6. Call `decad.PatternCopies(ctx,tooth,decad.CircularPattern{Center: frame.Origin(),
   Axis: frame.N(), Count: teeth})` for the other `N-1` teeth. Call
   `decad.Union(ctx,disc,append([]*decad.Body{tooth},copies...)...)` once, with the
   disc first and the tooth before its copies. Do not draw, rotate, or loft any other
   tooth, and do not chain `Union(gear,next)`. Require the returned body to be solid.
7. Save the root contacts for Step 7. For each tooth `k` in ascending order, rotate
   by `toothAngle(p,k)` the root coordinates of the unrotated right flank's first sample and
   of the left flank's first sample (scaled by `d.root/d.base` as in Step 3), and
   append right then left. The list has `2N` entries.

Embedded branch (`d.base < d.root`):

4. Build the root disc as in the ordinary branch.
5. Call `buildEmbeddedTooth` on both planes, and loft the two sections into one
   tooth body.
6. Pattern and join exactly as in the ordinary branch (`decad.PatternCopies`
   once, then one `decad.Union(ctx,disc,append([]*decad.Body{tooth},copies...)...)`).
   Return no saved root list.

Carry the returned body through Steps 7–9 and return the last body.

Implement `BuildProfile` by calling `buildGearProfile` after its guards and
discarding the root list. This API returns an unfilleted planar profile of the
whole gear for both branches; `Build` uses it only in the embedded branch.
Require the returned body to be solid. Do not reject a document because it
already contains bodies; the fresh-document body count is a test assertion.

## Step 7 — fillet root edges

Skip when the derived fillet radius is nonpositive. `radius` is the
`units.Value` from `doc.Param("fillet_radius")`, and `teeth` the count from
`doc.ParamInt("teeth")`. The private `filletRoots` keeps its float signature for
the existing tests and wraps its `radius` with `units.Millimeters`. Embedded branch: select
`decad.Edges(decad.ParallelTo(axis),decad.Concave()).Exactly(2*teeth)` and call
`decad.Fillet(ctx,body,selection,radius)`; the steps below
are the ordinary branch only. Ordinary branch:

1. Use `axis := placement.N()` and
   `decad.Edges(decad.ParallelTo(axis)).SelectEdges(body)`. Pass the placement
   frame into this step.
2. Read `edge.Start().Position().Value` for each axial edge. Match it to a
   saved root when `toWorld(placement, root)` is within `1e-6` of it. After the
   ordinary join, a corner vertex holds either the line's recorded end or the
   circle fragment's computed end, and the two differ by rounding.
3. Save the full 3D start once per matched edge. Require the number of matches
   to equal the saved root count before constructing the selector.
4. Start with `decad.Edges(decad.ParallelTo(axis),decad.EndpointAt(corners[0]))`.
   Add each remaining conjunction with `.Or(decad.ParallelTo(axis),decad.EndpointAt(corner))`.
5. Call `decad.Fillet(ctx,body,selection.Exactly(len(corners)),radius)`.

Use the returned body. Selecting every axial edge also selects flank joints;
include only edges matched to saved root contacts.

## Step 8 — cut a through-bore

In `Build`, call this operation only when `p.BoreDiameter > 0`. Implement
`CutThroughBore` and the internal build call through one shared routine:

1. Set a 1 mm tool margin (the `tool_margin` entry's value). Create a world and call `buildBoreProfile` with it.
2. In that helper, create the frame `shifted(placement, -margin)` from
   `p`'s placement, its plane, and its sketch. Use the frame creation APIs
   from Step 6. The tool then spans the placed gear along its normal.
3. Call `s.CreateCircle(s.Origin(),p.BoreDiameter/2)` and add
   `sketch.NewDiameter(circle,p.BoreDiameter)`, then bind that diameter with
   `s.Bind(diameter,world.Params(),"bore_diameter")`. `buildBoreProfile` keeps its
   signature, so it calls `defineParameters(world.Params(), p)` first.
4. Solve and verify, require exactly one valid profile, and return sketch/profile.
5. Call `decad.Extrude(ctx,doc,s,profile,decad.Distance{D: length,Dir: decad.Along})`
   to produce the tool, with `length` from `doc.Param("bore_tool_length")`.
   `CutThroughBore` calls `defineParameters(doc.Params(), p)` first, so other
   models can call it with a document that has not defined the table.
6. Call `decad.Cut(ctx,gear,tool)` and return its body.

## Step 9 — chamfer outer caps

In `Build`, call `ChamferOuterCaps` only when `p.CapChamfer > 0`:

1. Select `decad.Edges(decad.OuterLoopOf(decad.CapStart(gear))).Or(
   decad.OuterLoopOf(decad.CapEnd(gear)))`.
2. Call `decad.Chamfer(ctx,gear,selection,setback)`, where `setback` is
   `doc.Param("cap_chamfer")` when called from `Build`. The exported
   `ChamferOuterCaps` keeps its float signature and wraps `distance` with
   `units.Millimeters`; both share one private routine taking the `units.Value`.
3. Return its body or a contextual error.

Use cap references and outer-loop selectors rather than face indices or every
cap edge. The bore loop must not enter the query.

## Acceptance tests

Use Acceptance cases for numeric inputs. Start with Small end-to-end before
expanding to other cases. Use `t.Context()` and `decad.New()` for real geometry.
Do not infer success from an image or only a non-error return.

| Test | Assertions |
| --- | --- |
| Small end-to-end | Root-disc and tooth sketches fully constrained; complete `Build` solid; one document body; expected z bounds; `2N` fillet roles; verified mesh |
| Reused document | Build the Small end-to-end gear twice into one document; both calls return solid bodies and the document contains two bodies |
| Default body | `IsSolid()`; one `doc.Bodies()` entry; volume `2218.77` within `Volume().Bound.Base()+0.02`; volume bound below `0.5` |
| Default dimensions | z bounds 0 and thickness within `Bounds().Bound.Base()+1e-8`; max X strictly between `9.4` and `9.6` |
| Default fillets | Exactly `2N` distinct face origin roles starting with `fillet(0,` |
| Ordinary outline | `2N` saved roots at root radius within `1e-7`; `N` root arcs and `N` tip arcs; no other arc radius |
| Profile constraints | Independently call `Verify` on the root-disc, ordinary tooth, whole-outline and embedded sketches; assert all Sketch constraints report fields and `report.Check()` |
| Outline equivalence | Default `Build` without bore or chamfer against a loft of the `BuildProfile` outlines on both planes followed by the same root fillet: volumes agree within summed bounds plus `0.02` |
| Bore constraints | Inspect the separate bore sketch with the same report assertions |
| Bore tool | Removed box volume equals `π(b/2)²h` within both volume bounds plus `0.02`; final solid and one document body |
| Bore and chamfer | Solid; one document body; positive face-role count for `chamferCap(`; sharp bore rims on both caps |
| Chamfer only | Solid; one document body; volume and both cap areas decrease beyond summed respective error bounds |
| Embedded, no fillet | Solid; one document body; expected z bounds; positive chamfer-face count; one loop on each cap |
| Embedded, with fillet | Solid; one document body; `2N` distinct roles starting `fillet(`; one loop on each cap |
| Placed ordinary | Build the gear with and without the frame; volumes agree within summed bounds plus `0.02`; solid; one document body; bounds start at the frame origin along the normal and extend `h`; Y and Z extents are centered on the reference point with half-width within `0.1` of the tip radius |
| Placed embedded | Same assertions with root fillets, bore, and chamfer |
| Invalid placement | `Build` with `&r3.Frame{}` returns an error containing `placement` |
| Signed tooth section | Six curves and full constraints for lower angle zero and upper positive/negative angles; rotated centroid within `1e-6` |

- Inspect cap faces with `decad.Faces(decad.FaceCreatedBy(cap)).Exactly(1).SelectFaces(body)`.
  Use `decad.CapStart(body)` and `decad.CapEnd(body)`.
- Without a bore, require one cap loop. With a bore, require two loops and one
  edge in the inner loop. Require `decad.Circle3` with requested radius within
  `1e-10`; this checks sharp bore rims after chamfering.
- For the box, call `s.CreateRectangle(-5,-5,5,5)`. Dimension A relative to
  origin by signed X/Y, A→B by horizontal width, and B→C by vertical height.
  Solve and fully verify before extruding to the case's height.
- For signed sections, compute the arithmetic mean of all `edge.Polyline`
  coordinates in `profile.Outer`; compare upper mean with rotated lower mean.
  Use polylines only for this rotation measurement, never to replace curves.
- On Small end-to-end, Bore and chamfer, and Embedded with fillet, call
  `body.Tessellate(ctx,units.Millimeters(0.1),decad.WithVerification(decad.VerifyAll))`.
  Require no error, `mesh.BoundaryVerified()`, and `mesh.VolumeVerified()`.
- Assert invalid argument errors from Package and API and Input checks and limits.
  Numeric guards alone do not establish successful construction for every input.

## Downstream checks

| Consumer | Spur API boundary |
| --- | --- |
| `helicalgear` | Embeds `Params`; lofts `BuildToothSection` results; uses `CutThroughBore` and `ChamferOuterCaps` after assembly |
| `herringbonegear` | Uses `BuildToothSection` at lower and midpoint planes; calls helical assembly finishing |
| `bevelgear` | Uses `InvoluteFlanks` with noninteger virtual tooth count and rotation π on a back-cone plane |

Run `go test ./models/spurgear` as the focused generation gate. Exercise changed
interfaces through real consuming packages before claiming those connections
work; hand-authored arrays cannot prove this boundary. Use the repository pipeline
for isolated regeneration, broader CI, and rendering. Report engine failures
through AGENTS.md Agent coordination with parameters, last successful operation,
and the real failing build command.
