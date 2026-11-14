# Crank handle

## Purpose and source

Construct a printed hand crank for a D-flat shaft. It is the hand-crank siren's crank, moved here so that the siren and
any other mechanism build it from one construction. The parts are:

- an arm, a stadium pressed onto the shaft's D-flat;
- a pin, whose hexagonal stub sits in a pocket in the arm and whose head holds the handle on. It prints as two halves
  split lengthwise ([Split shafts](../../docs/model-pipeline.md#split-shafts));
- a handle, a barrel that turns freely on the pin's shank;
- an M2 x 10 bolt from the arm's back face and an M2 nut in the pin's stub, which clamp the pin in the arm;
- two M2 x 6 cross bolts and M2 nuts, which clamp the pin's two halves together.

The shaft is not modeled. It belongs to the mechanism, and its diameter equals the bore of the purchased bearing it runs
in ([Part clearances](../../docs/model-pipeline.md#part-clearances)). The presets read that bore from
[`models/ballbearing`](../ballbearing/). The bolt and nut are [`models/m2bolt`](../m2bolt/) and
[`models/m2nut`](../m2nut/), placed with `internal/m2fastener`. [`models/handcranksiren`](../handcranksiren/) builds its
crank with `BuildParts(Params6082RS())`.

All dimensions are in millimeters.

## Symbols

| Symbol | Meaning | Formula |
| --- | --- | --- |
| `D`, `d`, `c` | shaft diameter, flat depth, bore clearance | `ShaftDiameter`, `FlatDepth`, `BoreClearance` |
| `r_b` | bore radius | `D/2 + c` |
| `f_b` | distance from the axis to the bore's flat, toward the pin | `D/2 - d + c` |
| `L_b` | length of the bore's outline | `2 r_b (pi - acos(f_b / r_b)) + 2 sqrt(r_b^2 - f_b^2)` |
| `L` | throw, axis to pin axis | `Throw` |

## API

Package: `models/crankhandle`. Exported surface:

- The fixed dimensions as constants: `FitMargin`, `TightFit` (0.25), `RegularFit` (0.40), `ArmRadius` (10),
  `ArmThickness` (8), `BoreFloor` (1), `PinHoleR` (1.3), `PocketFlats` (7.53), `PocketFrom` (3.25), `PocketLength`
  (5.75), `StubFlats` (7.03), `StubFrom` (3.5), `StubLength` (5.8), `ShankR` (5.15), `ShankFrom` (8.25), `RootR`
  (3.5), `ShankTo` (52.75), `HeadR` (6.45), `HeadFrom` (55), `HeadLength` (3), `HandleFrom` (7.4), `PinBolt` (10),
  `PinCavity` (4.75), `PressWall` (1.6), `ChamferSize` (0.5), `CrossBolt` (6), `CrossFlat` (4.0), `CrossFlatHalf`
  (3.25), `CrossCavity` (5.0), `ConeAngle` (30), the cross bolts' positions `CrossZ` (17, 49), and `HandleProfile`,
  the handle's meridian.
- The container names `ArmContainerName` (`Arm`), `PinContainerName` (`Pin`), `HandleContainerName` (`Handle`),
  `BoltsContainerName` (`Bolts`) and `NutsContainerName` (`Nuts`).
- `Params` with `ShaftDiameter`, `FlatDepth`, `BoreClearance`, `Throw` and `Placement *r3.Frame` (nil selects the local
  frame).
- `Params605ZZ()`, `Params6082RS()` and `DefaultParams()`, which returns `Params605ZZ()`.
- `Params.Validate() error` and `Params.Derive() (Dimensions, error)`. `Dimensions` holds `BoreRadius`, `BoreFlat`,
  `BoreLoop`, `Site`, the pin bolt's `m2fastener.Site`, and `CrossSites`, the two cross bolts' sites, in the local
  frame.
- `NewDocument(opts ...decad.DocumentOption)` returns `decad.New(WithChainTolerance(0.01 mm), opts...)`. The pin's
  cuts after its union need a chain tolerance of 0.005 mm or more; build in such a document.
- `BuildParts(ctx, doc, p) (*Parts, error)` adds the arm, the pin and the handle and puts them in no container. `Parts`
  holds `Arm`, `Pin`, `Handle`, every `Sketches`, `Dimensions`, and in the world frame the pin bolt's `Site`, the cross
  bolts' `CrossSites` and `SplitNormal`, the unit normal of the plane that splits the pin.
- `Build(ctx, doc, p) (*Model, error)` adds the parts, the bolts and the nuts, each in its container. `Model` embeds
  `Parts` and holds `Bolt *m2bolt.Model`, `Nut *m2nut.Model`, `CrossBolts [2]*m2bolt.Model` and
  `CrossNuts [2]*m2nut.Model`.

## Presets and defaults

| Field | `Params605ZZ` (default) | `Params6082RS` (the siren's) |
| --- | --- | --- |
| `ShaftDiameter` | 5, `ballbearing.Params605ZZ().BoreDiameter` | 8, `ballbearing.Params6082RS().BoreDiameter` |
| `FlatDepth` | 0.5, the planetary gearbox shafts' flat | 0.75, the siren shafts' flat |
| `BoreClearance` | 0.05, `ballbearing.PressFitClearance` | 0.05 |
| `Throw` | 55 | 55 |

Derived: `r_b` 2.55 and 4.05, `f_b` 2.05 and 3.30, `L_b` 15.8 and 25.3. A 608ZZ has the same bore as a 608-2RS.

## Validation

Every message starts with `crankhandle: `. Derive checks in this order and returns the first failure:

1. Every float field is finite and positive: `<Field> must be finite and positive`.
2. `d <= D/4`.
3. `ballbearing.PressFitClearance <= c <= 0.5`.
4. `ArmRadius - r_b >= PressWall`, so the arm keeps a 1.6 mm wall round the bore (`D` at most 16.7).
5. `L >= 2 ArmRadius`.
6. `Placement` is nil or a valid frame.

BuildParts and Build reject a nil context and a nil document, derive the dimensions, and check `ctx.Err()` before any
geometry. A rejected build adds no body.

## Local frame

The origin is on the shaft axis in the arm's back face, the face toward the mechanism. +Z runs along the shaft axis away
from the mechanism, the arm runs along +Y to the pin axis at `(0, L)`, and the bore's flat faces +Y. A `Placement` maps
this frame onto the given one after each part's last operation.

## Geometry

1. **Arm.** A sketch on z = 0: arcs of radius `ArmRadius` about the origin and about `(0, L)`, joined by two tangent
   lines, with a circle of radius `PinHoleR` about `(0, L)`. Extrude it `ArmThickness` along +Z, then chamfer the outer
   loop of both faces by `ChamferSize`. Cut the blind D bore, a sketch on z = -1 extruded `ArmThickness` (to z = 7): an
   arc of radius `r_b` about the axis the long way round through -Y, closed by its flat at `y = f_b`. Cut the hexagonal
   pocket, a hexagon `PocketFlats` across flats about `(0, L)` with its corners on X, from z = `PocketFrom` for
   `PocketLength`, out through the front face. Cut the pin bolt's head pocket.
2. **Pin.** About `(0, L)`:
   - Join a hexagon `StubFlats` across flats, corners on X, from z = `StubFrom` for `StubLength` (to z 9.3), and a
     revolved meridian: a cone of half angle `ConeAngle` from radius `RootR` at z = `ShankFrom` out to `ShankR` at
     z 11.11, the shank to z = `ShankTo`, a cone out to `HeadR` at z = `HeadFrom`, and the head to z 58. The root cone
     starts inside the stub's flats and grows past its corners (4.06) by z 9.22, inside the stub.
   - Cut each cross bolt's double flat: everything beyond `CrossFlat` (4.0) from the axis on both the +Y and the -Y
     side, over z = `CrossZ[i]` ± `CrossFlatHalf`, so 13.75 to 20.25 and 45.75 to 52.25.
   - Cut the pin bolt's hole and nut cavity with `m2fastener.RoundedHoleTool` and `RoundedCavityTool`: radius 1.3 to
     z 13 and radius 2.51 from z 4.75 to 6.75, each end corner rounded by 0.55 mm.
   - Cut each cross bolt's clearance hole, head pocket and nut cavity with `m2fastener.HoleTool`, `PocketTool` and
     `CavityTool`.
3. **Handle.** Revolve `HandleProfile` about the pin axis. The sketch plane holds the axis, with its origin at
   `(0, L, HandleFrom)`, U along +Z and V the radius. The handle runs from z = 8.4 to 58.4: a bore of radius 5.55, which
   ends in a 30° cone 0.4 mm off the pin's head cone, a counterbore of radius 7.05 for the pin's head, a waist and a
   flared outer end. Every corner of the meridian is cut by a small chamfer.
4. **Fasteners.**
   - The pin bolt's head lies flush with the arm's back face at `(0, L)` and the bolt runs along +Z: head pocket 2 deep
     in the arm, hole 2.6 across through the arm and into the pin, nut cavity from 4.75 to 6.75 in the stub, and the
     tip at z = 12, inside the shank. `m2fastener.PlaceBolt` and `PlaceNut` place an M2 x 10 bolt and an M2 nut.
   - Each cross bolt's site has its head at `(0, L + CrossFlat, CrossZ[i])`, `Dir` (0, -1, 0), `Corner` (1, 0, 0),
     `Length` 6 and `Cavity` 5. Its head lies flush with the +Y flat in a pocket 2 deep (y from L + 4 to L + 2), its
     hole runs through both halves, its nut cavity lies in the -Y half from y = L - 1 to L - 3, and its tip ends at the
     -Y flat (y = L - 4). `PlaceBolt` and `PlaceNut` place an M2 x 6 bolt and an M2 nut at each.
5. **Containers.** `Build` files the arm, pin and handle in their containers, the three bolts in `Bolts` and the three
   nuts in `Nuts`, each container a child of the document root.

## Split pin

The pin prints as two halves split by the plane through its axis and local X ([Split
shafts](../../docs/model-pipeline.md#split-shafts)). X is the direction of the hand's driving force, so that force
loads the halves side by side and not across the seam. The plane passes through two corners of the stub's hexagon.
`Parts.SplitNormal` is its normal, local +Y, along the arm toward the pin. The model keeps the pin whole; the export to
Fusion cuts it.

- The +Y half (away from the shaft) holds the cross bolts' head pockets. The -Y half holds their nut cavities, which it
  encloses on all sides but the bolt hole, with 1 mm of pin between each cavity and the seam and 1 mm to the -Y flat.
- The two M2 x 6 cross bolts clamp the halves. One sits past the pin bolt's hole and one short of the head cone, so
  the halves act as one section over the span the hand bends.
- The seam cuts the pin bolt's hole and nut cavity in half. The clamped halves enclose its nut, and the stub's place in
  the arm's pocket closes the halves round it too.
- The pin bolt is still needed: it alone holds the pin in the arm, whose pocket opens through the arm's front face.
- Apart from the pockets and cavities round each cross bolt, the halves are mirror images. Each stands 6.45 mm above
  its cut face.

| Wall round a cross bolt | mm |
| --- | --- |
| Pin round the nut cavity at its far face | 1.68 |
| Flat round the head pocket | 1.04 across, 1.05 along the pin |
| Pin under the head pocket to the seam | 2.0 |
| Pin between the nut cavity and the seam | 1.0 |
| Pin between the nut cavity and the -Y flat | 1.0 |

## Part fits

| Fit | Class | Clearance |
| --- | --- | --- |
| D bore round the shaft | Press fit, the arm turns with the shaft | `c` = 0.05 radial, on the round and on the flat |
| Pin stub in the arm's pocket | Tight | 0.25 on each flat |
| Handle on the pin's shank | Regular | 0.40 radial (bore 5.55, shank 5.15) |
| Handle's inner end to the arm's front face | Regular | 0.40 |
| Handle's bore cone to the pin's head cone | Regular | 0.40 normal to the cones |
| Handle's counterbore round the pin's head | | 0.6 radial, 0.4 above the head |
| Bolts, nuts | Fastener rules | head pocket 0.3, hole 0.3, cavity 0.2 |

The pin bolt's nut cavity has 0.55 mm rounds at its end corners. At the nut's faces they leave 0.07 mm of the 0.2 mm
radial clearance past the nut's corners.

The handle is a solid of revolution about the pin axis, so it turns freely when every gap in a cut through that axis
clears its class. The pin's head (radius 5.8) is wider than the handle's bore (radius 4.9), so the handle cannot slide
off the pin. The model is **Mixed**: the bore fit is adjustable and the bolt and nut keep their nominal size.

The arm is held on the shaft by the press fit alone, as on the siren. Nothing clamps it along the shaft.

## Valid range

- `ShaftDiameter` up to 16.7 at the default clearance; `FlatDepth` up to `D/4`; `BoreClearance` 0.05 to 0.5.
- `Throw` from 20 up.

## Representative cases

- `605ZZ`: the default.
- `608-2RS`: the siren's crank.
- `regular-bore`: `Params605ZZ` with `BoreClearance` 0.35.
- `long-throw`: `Params6082RS` with `Throw` 80.
- Placed: the frame with origin `(10, -5, 7)`, U `(0, 1, 0)` and V `(0, 0, 1)`.

## Retained sharp corners and edges

The rule is "Corners and roots" in `docs/model-pipeline.md`. The arm prints on its back face, each half of the pin on
its cut face, and the handle upright on its inner end. In the pin's halves the 30° cones join the stub, the shank and
the head without a sharp step.

| Part | Corner or edge | Reason |
| --- | --- | --- |
| Arm | The six corners of the hexagonal pocket | Joinery: they match the sharp corners of the pin's stub. |
| Arm | The two corners of the D bore | Mating corners of the shaft's flat. They turn 36.5 degrees (605ZZ) and 35.4 (608-2RS), so they need no round. |
| Arm | The bore's rim in the back face, at 608-2RS (`L_b` 25.3 mm, a long run) | Not chamfered: Decad refuses the chamfer (steps.md, "Known decad limit"). The siren's arm has the same sharp rim. |
| Pin | The pin bolt's nut cavity's 0.55 mm end rounds, in the layer 2 mm off the seam | That layer crosses the rounds near their edge, where they turn about 52°. They are the largest rounds that keep the nut's corner clearance. |

The pin carries the handle beside the arm's front face without a collar, as on the siren. [Arbor
collars](../../docs/model-pipeline.md#arbor-collars) asks for a collar there; the siren's crank predates the rule.

## Printing

Print the arm on its back face and the handle upright on its inner end. Print the pin as its two halves, each lying
on its cut face, with no supports. The half grooves of the pin bolt's hole and nut cavity on the cut face are short
bridges, at most 3.6 mm across, and so are the roofs of the -Y half's nut cavities, 5.02 mm across. Pause the -Y half's print at 3.0 mm, the top of its two nut cavities, drop an M2 nut
into each, and resume. The +Y half needs no pause.

To assemble:

1. Drop the pin bolt's M2 nut into its half pocket in one half, and lay the other half on it.
2. Push each M2 x 6 cross bolt through its head pocket in the +Y half into its nut in the -Y half, and tighten it.
3. Slide the handle onto the pin's shank from the stub end.
4. Push the stub into the arm's pocket and drive the M2 x 10 bolt in from the arm's back face.
5. Press the arm onto the shaft.

## Checks and images

Check every case above: every sketch fully constrained, the nine bodies and their containers, the bore's clearance
round the nominal shaft, its floor and wall, every fit between the printed parts with the handle's gaps in a cut
through the pin axis, each bolt's head, tip and nut against its pockets with no overlap, the walls round the cross
bolts, the face chamfers, `internal/splitcheck` on the pin with its split normal and the cross bolts' lines exempt, and
`internal/cornercheck` on the arm and handle (axis +Z) and on the pin (axis `SplitNormal`). Check the default's bounds
and `VerifyAll` meshes, and the placed crank against the local one. The siren's own tests check its crank through the assembly.
