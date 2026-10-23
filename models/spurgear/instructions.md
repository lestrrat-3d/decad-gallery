# Spur gear construction instructions

Use this contract to derive [steps.md](steps.md), then generate the Go package
and tests. This file owns geometry, dimensions, input rules, and case inputs.
The steps own interfaces, Sketch/Decad calls, and acceptance assertions.
Existing spur Go files are deliverables, not prerequisites for understanding
the construction.

## Inputs

Build one cylindrical gear with straight involute teeth. Use millimeters for
lengths and radians for angles. Anchor the gear with a reference point and a
reference plane, given as one `r3.Frame`: its origin is the reference point,
its `U` and `V` axes span the reference plane, and its normal is `U × V`.
The gear axis passes through the reference point along the normal. The lower
cap lies on the reference plane and the upper cap lies `h` above it along the
normal. Center tooth zero on `U`. All construction below is stated in this
frame's coordinates (`x` along `U`, `y` along `V`, `z` along the normal). With
no placement, the frame is the world origin with `U=+X` and `V=+Y`, so the
axis is world +Z and the caps are at z=0 and z=`h`.

| Symbol | Parameter | Type | Default | Meaning |
| --- | --- | --- | --- | --- |
| `m` | `Module` | float64 | 1 | Tooth module |
| `N` | `Teeth` | int | 17 | Tooth count |
| `α` | `PressureAngle` | float64 | `20π/180` | Pressure angle |
| `h` | `Thickness` | float64 | 10 | Axial thickness |
| `S` | `Samples` | int | 15 | Fit points per flank |
| `b` | `BoreDiameter` | float64 | 0 | Centered through-bore diameter; zero omits bore |
| `c` | `CapChamfer` | float64 | 0 | Equal-distance outer cap setback; zero omits chamfer |
| — | `Placement` | `*r3.Frame` | nil | Reference point and plane; nil selects the world XY frame at the origin |

Use finite geometric inputs. Do not interpret zero fields as default requests;
provide defaults through a separate constructor. A nil `Placement` is the one
exception: it means the world frame described above.

## Derived dimensions

Compute without rounding:

| Dimension | Formula |
| --- | --- |
| Pitch radius `Rp` | `mN/2` |
| Base radius `Rb` | `Rp cos α` |
| Root radius `Rr` | `(mN - 2.5m)/2` |
| Tip radius `Rt` | `(mN + 2m)/2` |
| Tooth angular pitch | `2π/N` |
| Root fillet radius `Rf` | `0.9 Rr (π/N - 2(tan α - α))/2` |

## Parameter table

The gear's inputs and derived dimensions are named entries of the document's
parameter table, so a generator can read their names and expressions. Every
feature value that follows is read from this table, not computed in place.

| Name | Unit | Definition |
| --- | --- | --- |
| `module` | mm | input `m` |
| `teeth` | none | input `N` |
| `pressure_angle` | rad | input `α` |
| `thickness` | mm | input `h` |
| `samples` | none | input `S` |
| `bore_diameter` | mm | input `b` |
| `cap_chamfer` | mm | input `c` |
| `tool_margin` | mm | `1` |
| `pitch_radius` | mm | `module * teeth / 2` |
| `base_radius` | mm | `pitch_radius * cos(pressure_angle)` |
| `root_radius` | mm | `(module * teeth - 2.5 * module) / 2` |
| `tip_radius` | mm | `(module * teeth + 2 * module) / 2` |
| `involute_angle` | none | `tan(pressure_angle) - deg(pressure_angle) * pi / 180` |
| `fillet_radius` | mm | `0.9 * root_radius * (pi / teeth - 2 * involute_angle) / 2` |
| `bore_tool_length` | mm | `thickness + 2 * tool_margin` |

The definitions are written as string literals, one call each, so that the
names and expressions can be read from the source. The root-disc circle radius
is bound to `root_radius` and the bore circle diameter to `bore_diameter`. The
tooth count, the extrusion distances, the fillet radius, the bore tool length
and the chamfer setback are read from the table.

## Input checks and limits

- Reject `m <= 0`, `h <= 0`, `N < 3`, or `S < 3`.
- Reject `α <= 0` or `α >= π/2`.
- Reject negative `b` or `c`.
- Reject a non-nil `Placement` that is not a valid frame.
- Reject `Rr <= 0`, `Rb <= 0`, or `Rt <= Rb`.
- Reject `b >= 2Rr` or a nonfinite `Rf`.
- Select filleting only when `Rf > 0`; nonpositive `Rf` is permitted.
- Return the geometry error when an otherwise admissible input cannot produce
  the specified profile, fillet, cut, or chamfer. No universal maximum chamfer
  or successful-build guarantee for the entire numeric range is specified.
- Select the ordinary branch at `Rb == Rr`. Its radial connectors collapse to
  zero length; the acceptance cases do not establish engine behavior at equality.
- Do not promise uniform rejection of nonfinite fields: current numeric guards
  explicitly check finiteness only for `Rf`. Behavior outside the finite-input
  contract requires a separate validation change.

## Tooth flank

Sample the original involute from base to tip, including both endpoints:

1. For `i=0..S-1`, use `rᵢ = Rb + (Rt-Rb)i/(S-1)`.
2. Set `t = tan(acos(Rb/rᵢ))`.
3. Compute `I(rᵢ) = (Rb(cos t + t sin t), Rb(sin t - t cos t))`.
4. Mirror `I` across X by negating its Y coordinate.
5. Evaluate the same involute at `Rp`, mirror it to `(px,py)`, and compute
   `β = π/(2N) - atan2(py,px)`.
6. Rotate each mirrored sample by `β` to make the left flank.
7. Mirror that rotated sample across X to make the right flank.
8. Rotate both flanks by `θ = 2πk/N` for tooth `k=0..N-1`.

Use `Rotθ(x,y) = (x cos θ-y sin θ, x sin θ+y cos θ)`. Both arrays run
base-to-tip. Their pitch crossings before tooth rotation are at `±π/(2N)`.
Interpolate each full array with one native fit spline. Do not replace splines
with chords, resample from the root, or alter their shape during trimming.

## Body

### Ordinary construction

Use this branch when `Rb >= Rr`. The gear is one root disc and one tooth
repeated `N` times about the gear axis, joined into one solid.

- Root disc: a circle of radius `Rr` about the gear axis, extruded from the
  lower cap plane to the upper cap plane.
- Tooth zero is drawn once, on the lower cap plane at angle 0, as the six-curve
  tooth section described below, and extruded from the lower cap plane to the
  upper cap plane into one straight tooth solid. Repeat that one tooth solid
  `N` times with a circular pattern about the gear axis through the placement
  origin; do not draw a separate sketch for any other tooth.
- Scale each flank's base point by `Rr/Rb` to locate its root contact. Evaluate
  each coordinate as `x*Rr/Rb`, multiplying before dividing, not as `x*(Rr/Rb)`.
  The helical join rejects the one-bit coordinate difference.
- Traverse each tooth section from right root to right base by line, along the
  right spline to its tip, along the tip-circle arc to the left tip, along the
  left spline back to its base, by line to the left root, and by the short
  root-circle arc back to the right root.
- Each tooth section has two splines, two radial lines, one tip arc and one
  root arc, and the root arc lies on the root-disc circle. Preserve unrounded
  root corners.
- The whole-gear outline of the same teeth (the root arc of each tooth joined
  to the next tooth's right root, one outline for all `N` teeth) remains
  available to callers as a plane profile; the ordinary gear build does not use it.

### Embedded profile

Use this branch when `Rb < Rr`.

- Draw all original base-to-tip flanks and each right-to-left tip arc.
- Add one full root circle at the gear origin.
- Union all bounded sketch regions into one hole-free outline.
- Require `4N` boundary curves: two trimmed splines, one tip arc, and one
  root-circle segment per tooth.
- Require certified spline trim parameters at the root-circle intersections.
- Save both actual trimmed root contacts per tooth for fillet selection.

### Embedded tooth

Use this branch when `Rb < Rr`. The gear is one root disc and one natural tooth
repeated `N` times, as in the ordinary branch, but the tooth flanks cross the
root circle instead of ending on it.

- Let `h(r)` be the angle between the tooth axis and the left flank where the
  flank meets radius `r`, for `Rb <= r <= Rr`. Adjacent teeth stay apart at
  radius `r` only while `h(r) < π/N`.
- Choose the closing radius `Rc`. If `h(Rb) < π/N`, use `Rc = Rb`. Otherwise
  find `Rq` in `[Rb, Rr)` with `h(Rq) = π/N` (`h` falls as `r` grows) and use
  `Rc = (Rq + Rr)/2`, so the closing arc lies strictly inside the root disc and
  clear of the neighbouring teeth. If `h(Rr) >= π/N`, no closing radius exists:
  return an error that names the root circle.
- The tooth section has four curves: the right flank spline from radius `Rc`
  to the tip, the tip arc, the left flank spline from the tip back to radius
  `Rc`, and an arc about the gear axis at radius `Rc` that closes it. The flank
  samples run linearly in radius from `Rc` to `Rt`, using the same involute
  and rotation as the ordinary tooth. Nothing is scaled to the root circle.
- Each flank crosses the root circle once. The tooth, its copies, and the root
  disc are joined in one union; the join cuts the flanks at the root circle.
- The root fillet corners are the `2N` concave axial edges of the joined body.

### Feature order

Ordinary gear (`Rb >= Rr`):

1. Extrude the root disc, and extrude tooth zero from its one section.
2. Pattern tooth zero `N` times about the gear axis, then join the disc and all
   `N` teeth in one union into one straight solid.
3. If `Rf > 0`, fillet all `2N` axial root edges together with radius `Rf`.
4. If `b > 0`, extrude a centered circular tool from below z=0 to above z=`h`,
   then cut it from the gear.
5. If `c > 0`, chamfer both outer cap loops with setback `c`.

Embedded gear (`Rb < Rr`):

1. Extrude the root disc, and extrude the embedded tooth from its one
   section.
2. Pattern the tooth `N` times about the gear axis, then join the disc and all
   `N` teeth in one union into one straight solid.
3. If `Rf > 0`, fillet the `2N` concave axial edges together with radius `Rf`.
4. Steps 4 and 5 of the ordinary order apply unchanged.

The root disc and the bore tool use extrusion. Keep bore rims sharp during
chamfering. Require the returned gear body to be one solid. A fresh document
contains one body afterward (the disc and teeth are consumed by the join); a
document with existing bodies must also accept a new gear.

## Sketch constraints

- Use the sketch origin as the gear center; do not add a separately fixed center.
  Every sketch lies on a plane built from the placement frame, so its origin is
  the reference point (or its offset along the normal).
- Locate computed points with signed horizontal and vertical driving dimensions
  from that origin. Reuse endpoint objects for adjoining curves.
- Give every right-flank point both coordinate dimensions. Give each left-flank
  point both dimensions except its tip, which receives one coordinate dimension.
  In an embedded tooth section, the left flank's first point (on the closing
  arc) also receives only one coordinate dimension.
- In ordinary profiles, give each right root both dimensions and each left root
  one dimension. The joining arc supplies the remaining radius relation.
- For a point receiving one dimension, dimension Y when `abs(x) >= abs(y)`;
  otherwise dimension X. Its initial coordinates select the intended circle branch.
- Do not add radius constraints to tip or ordinary root arcs: their centers and
  fully dimensioned opposite endpoints already determine their radii.
- Dimension the root-disc circle and the separate embedded root circle by `Rr`
  and the bore circle by `b`.
- Solve, then verify every sketch before reading profiles or returning it.
  Require `sketch.FullyConstrained`, `DOF == 0`, no redundant constraints, no
  conflicts, and a successful verification report check.
- Apply these rules to the root-disc sketch, every tooth section on both cap
  planes, both embedded end profiles, standalone tooth sections, and bore
  tools. Do not fix computed points to remove freedom.

## Reusable constructions

- Expose the whole unfilleted outline on a caller-supplied plane for other lofts.
- Expose one ordinary tooth at an arbitrary finite signed angle. Close it with
  the short root arc from its right root to its left root. The six-curve profile
  traverses this root arc in reverse to close the region.
- Reject embedded standalone tooth requests; the complete embedded gear profile
  remains a separate supported route.
- Expose flank sampling with caller-supplied `pitch`, `base`, `tip`, real-valued
  `effectiveTeeth`, `samples`, and signed `angle`. Use Tooth flank with those
  radii and `effectiveTeeth` in place of `N`; never round it.
- For that sampling API, reject nonfinite scalar inputs, `base <= 0`,
  `pitch <= base`, `tip <= pitch`, `effectiveTeeth <= 0`, or `samples < 3`.
- Expose through-bore and outer-cap chamfer operations for assembled gears.
  Bore callers supply a body spanning z=0 through `h`, a positive diameter, and
  valid context/document/body arguments. Chamfer callers supply a positive
  setback and a body whose two cap references identify its outer loops.
- Expose `Params.RejectPlacement(model string) error`. It returns an error
  naming `model` when `Placement` is non-nil. Gear models that reuse `Params`
  without placement support call it so a placement is never silently ignored.

## Acceptance cases

Apply these overrides to Inputs defaults. This table owns case inputs; the
[acceptance tests](steps.md#acceptance-tests) own their detailed assertions.

| Case | Overrides | Required result |
| --- | --- | --- |
| Small end-to-end | `N=8`, `h=2`, `S=5` | Root-disc and tooth sketches verified, one solid, 16 root fillets, verified mesh |
| Default | None | Ordinary body, 17 teeth, 34 native root fillets |
| Bore tool | `b=3`; use a 10×10×10 box spanning z=0..10 | Remove a centered cylindrical volume |
| Bore and chamfer | `b=3`, `c=0.1` | Gear with sharp bore rims and chamfered outer caps |
| Chamfer only | `c=0.1` | Both cap areas and total volume decrease beyond error bounds |
| Embedded, no fillet | `m=0.5`, `N=30`, `α=30π/180`, `h=2`, `S=5`, `c=0.02` | Embedded tooth join, no root fillet, chamfered caps |
| Embedded, with fillet | `m=0.5`, `N=24`, `α=30π/180`, `h=2`, `S=5` | Embedded tooth join, 48 native root fillets |
| Signed tooth section | `N=8`, `h=2`, `S=5`; angles `0`, `±14.5π/180` | Six-curve profiles with the requested rotation |
| Placed ordinary | Default with `b=3`, `c=0.1`; frame origin `(10,-5,7)`, `U=+Y`, `V=+Z` | Same volume as the unplaced gear; bounds follow the frame |
| Placed embedded | `m=0.5`, `N=60`, `h=2`, `S=15`, `b=3`, `c=0.02`; same frame | Same volume as the unplaced gear; bounds follow the frame; root fillets present |
| Invalid placement | Default with a zero-value `r3.Frame` | Error naming the placement |

## Verification

Run the Small end-to-end case first. Check every placed case against the same
gear built without placement. Check constraints independently for the
root disc, the ordinary tooth sections on both cap planes, the whole-gear outline,
embedded profiles, bore tools, and signed tooth sections.
Confirm tooth curves, exact root trims, final topology, cap positions, and
feature surfaces. An odd tooth count need not reach `±Rt` on both axes.
Check shared construction through real helical, herringbone, and bevel consumers.
Render the real Decad body through `tools/render/`.
