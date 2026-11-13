# Spiral spring Decad steps

Follow instructions.md for the API, the symbols and the governing dimensions. Emit three files: `params.go`
(constants, `Params`, `DefaultParams`, `Validate`, `Derive`, `Dimensions`), `profile.go` (the sketch) and
`spiralspring.go` (package doc, `Model`, `Build`, 3D features, placement). Every error message starts with
`spiralspring: `.

## 1. Validate and derive (params.go)

`params.go` makes no geometry calls. One unexported `derive(p Params) (Dimensions, error)` runs the 14
validation rules of instructions.md in their stated order, then computes every `Dimensions` field from the
symbol table; the field-to-symbol mapping is in instructions.md (`TailLength` is `L`). `Validate` returns its error; `Derive` returns both values. `WindStrain` and `ActiveLength` use the
formulas of instructions.md. `BoreFlatDistance` is 0 when `ArborFlatDepth` is 0. The float finite check names
the field: `<Field> must be finite`. Messages:

- rule 2: `spiralspring: Thickness must be at least 1.5 mm`
- rule 11: `spiralspring: OuterDiameter leaves no room for one turn`
- rule 12: `spiralspring: Turns %d needs a gap below MinGap; at most %d turns fit`
- rule 14: `spiralspring: Placement must be a valid frame`

Every other rule gets a message naming the field and the limit.

## 2. Sketch the profile (profile.go)

Signature: `buildProfile(ctx context.Context, p Params, dm Dimensions) (*sketch.Sketch, *sketch.Profile,
[5]r3.Vec, error)`. The corners are returned in the order C1, C3, C4, P_i, P_o as solved sketch coordinates
with Z = 0.

Use a fresh `sketch.NewWorld()`, `world.CreateSketch(world.XY())` and `A = s.Origin()`. Create every point at the
coordinate given; the solver starts there, and two-solution constraints pick the side of the start point.
With `R, g, d, w, rho0` from `Dimensions` and `t = p.Thickness`:

1. Center `B = (d, 0)`. Constraints `NewHorizontalPoints(A, B)` and `NewHorizontalDistance(A, B, d)`.
2. Root. Points `C1 = (-R, 0)`, `C2 = (-(R + g + t), 0)`, `C3 = (d - sqrt(rho0^2 - w^2), -w)`,
   `C4 = (-sqrt(R^2 - w^2), -w)`.
   Entities: `hub = CreateArc(A, C4, C1)`, `L1 = CreateLine(C1, C2)` (root end face) and
   `L2 = CreateLine(C3, C4)` (spoke underside).
   Constraints: `NewRadius(hub, R)`, `NewHorizontal(L1)`, `NewHorizontalPoints(A, C2)`, `NewHorizontal(L2)`,
   `NewDistancePointLine(A, L2, w)`.
3. Coil, for `k = 0 .. 2N-1`. The center `c_k` is `B` for even `k` and `A` for odd `k`.
   - Inner end point `J_{k+1}`: x = `d + r_k` (even `k`) or `-r_k` (odd `k`), y = 0. Outer end point
     `O_{k+1}`: x = `d + r_k + t` (even `k`) or `-(r_k + t)` (odd `k`), y = 0. `r_k = rho0 + k*d`.
   - `inner[k] = CreateArc(c_k, J_k, J_{k+1})` with `J_0 = C3`. `outer[k] = CreateArc(c_k, O_k, O_{k+1})` with
     `O_0 = C2`. Arcs run counter-clockwise from start to end.
   - `NewHorizontalPoints(A, J_{k+1})` and `NewHorizontalPoints(A, O_{k+1})`.
   - After the loop: `NewRadius(inner[0], rho0)` and `NewRadialOffset(inner[0], outer[0], t)`.
   - Every later radius follows from the shared junctions. Adding any other radius dimension is redundant.
4. Tail and eye. Let `J = J_{2N}`, `O = O_{2N}`, `y_c = -L + sqrt(R_e^2 - t^2/4)`. Points:
   `E = (E.x, -L)`, `P_i = (E.x + t/2, y_c)`, `P_o = (E.x - t/2, y_c)`.
   Entities: `tailIn = CreateLine(J, P_i)`, `tailOut = CreateLine(O, P_o)`, `eye = CreateArc(E, P_o, P_i)` (the
   long way round through the bottom) and `hole = CreateCircle(E, D_hole/2)`.
   Constraints: `NewVertical(tailIn)`, `NewVertical(tailOut)`, `NewRadius(eye, R_e)`,
   `NewVerticalDistance(J, E, -L)`, `NewHorizontalDistance(O, E, t/2)`, `NewDiameter(hole, D_hole)`.
5. Bore. Flat bore (`ArborFlatDepth > 0`): `Q1 = (sqrt(r_b^2 - f^2), f)`, `Q2 = (-sqrt(r_b^2 - f^2), f)`,
   `bore = CreateArc(A, Q2, Q1)`, `flat = CreateLine(Q1, Q2)`. Constraints `NewRadius(bore, r_b)`,
   `NewHorizontal(flat)`, `NewDistancePointLine(A, flat, f)`. Round bore: `bore = CreateCircle(A, r_b)` and
   `NewDiameter(bore, 2*r_b)`.
6. `s.Solve(ctx)`, then `internal/sketchcheck.VerifyFullyConstrained(ctx, s)`.
7. `s.Profiles()`. Require exactly 3 profiles, all `Valid`. Select the one with `len(Holes) == 2`; the other two
   are the bore disc and the eye-hole disc. Anything else is an error.

Three rules:

- Do NOT use `NewPointOnLine` against a short construction axis. Its lever arm through `B` drives the sketch's
  conditioning below the verification gate.
- Do NOT use `sketch.CreateFillet`. The fillets are 3D (step 4).
- Do NOT add any radius dimension beyond those listed, and do NOT fix computed points to force `DOF == 0`.

Wrap every sketch error `spiralspring: <step>: %w` and return before any body is added.

## 3. Extrude (spiralspring.go)

Build, in order: nil ctx -> `spiralspring: context must not be nil`; nil doc -> `spiralspring: document must not
be nil`; `p.Derive()`; `ctx.Err()` wrapped `spiralspring: before creating profile: %w`; `buildProfile`.

`decad.Extrude(ctx, doc, s, profile, decad.Distance{D: units.Millimeters(h), Dir: decad.Along})`. The body spans
z from 0 to `h`.

## 4. Fillet the five concave corners

Selector: OR over the five corners of
`decad.Edges(decad.Concave(), decad.ParallelTo(r3.Vec{Z: 1}), decad.EndpointAt(corner))`, then `.Exactly(5)`.
Without `ParallelTo` the query matches 6 edges. `decad.Fillet(ctx, body, sel, units.Millimeters(r_f))`.

## 5. Edge finishing

Rule: docs/model-pipeline.md "Surface edge finishing". Use `internal/edgefinish`; do not write edge predicates.

- Skip this step when neither cap face is a planar face with `|N.Z| = 1` and area at least
  `edgefinish.LargeFaceArea` (`edgefinish.Chamfer` fails when it selects nothing). Measure the faces at z = 0
  and z = `h` with `Face.Area()`.
- Otherwise one call: `edgefinish.Chamfer(ctx, body, r3.Vec{Z: 1}, []float64{0, h}, keep,
  edgefinish.MinChamfer)`, where `keep(loop, edges)` is true for `edgefinish.Outer(loop, edges)` or
  `edgefinish.Long(edgefinish.Inner)(loop, edges)`. That chamfers the outer loop of each cap, and the bore
  loop and the eye-hole loop when the loop's edge run is at least `edgefinish.LongEdgeRun`. Decad cannot chamfer
  again after a whole-outline chamfer, so both caps go in this one call.

## 6. Place

`placement.Place(ctx, body, p.Placement)` from `internal/placement`. A nil frame leaves the body in place. Wrap
its error `spiralspring: place spring: %w`.

## 7. Return

Require `body.IsSolid()`, else an error. Return `&Model{Body: body, Profile: s, Dimensions: dm}`. Build adds exactly
one body to doc; decad replaces the intermediate bodies.

## Known decad limit

After the cap chamfer, `Mesh.VolumeVerified()` is false. Decad publishes no occupied-volume proof for a
cap-loop chamfer with circular walls (see `VolumeVerified` in decad's `tessellate_verification.go`).
`BoundaryVerified()` stays true. An unchamfered body verifies both. The chamfered mesh's `Mesh.Bound()` is
about 0.8 mm at a requested 0.02 mm, from the chamfer bands. The coil-gap test therefore slices the mid-height
section, where the walls are vertical cylinders and planes, and charges the chord tolerance (0.02 mm) plus
0.001 mm instead of the certified bound.
