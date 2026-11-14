# Crank handle

## Purpose and source

Construct a printed crank handle that turns a D-flat shaft by hand. The crank is one printed solid: a flat plate
whose hub has a D bore, a tapered arm, and a round grip standing on the arm's end. An M2 set screw enters the hub's
flat tail face, passes a captive nut, and presses on the shaft's flat.

The shaft is not modeled. It belongs to the mechanism the crank drives, and its diameter equals the bore of the
purchased bearing it runs in ([Part clearances](../../docs/model-pipeline.md#part-clearances)). The presets take that
bore from the bearing presets in [`models/ballbearing`](../ballbearing/). The bolt and the nut are
[`models/m2bolt`](../m2bolt/) and [`models/m2nut`](../m2nut/), placed with `internal/m2fastener`.

All dimensions are in millimeters.

## Symbols

| Symbol | Meaning | Formula |
| --- | --- | --- |
| `D` | shaft diameter | `ShaftDiameter` |
| `d` | depth of the shaft's flat | `FlatDepth` |
| `c` | radial bore clearance | `BoreClearance` |
| `r_b` | bore radius | `D/2 + c` |
| `f_b` | distance from the axis to the bore's flat | `D/2 - d + c` |
| `w_b` | half the width of the bore's flat | `sqrt(r_b^2 - f_b^2)` |
| `L_b` | length of the bore's outline | `2 r_b (pi - acos(f_b / r_b)) + 2 w_b` |
| `R_h` | hub radius | `HubDiameter/2` |
| `L_s` | set screw length | 6, the shortest of `m2bolt.StandardLengths()` |
| `t` | distance from the axis to the tail face | `f_b + 2 + L_s` (head height 2) |
| `w_t` | half the width of the tail face | `sqrt(R_h^2 - t^2)` |
| `T` | plate thickness | `Thickness` |
| `z_s` | height of the set screw's axis | `T/2` |
| `rho` | grip root fillet radius | `min(T, GripDiameter)/2` |
| `R_e` | radius of the arm's end about the grip axis | `GripDiameter/2 + rho + 1.5` |
| `r_t` | round on the grip's top edge | `min(2, GripDiameter/4)` |
| `k` | rim chamfer of the head pocket floor and the nut cavity | `0.28` |

## API

Package: `models/crankhandle`. Exported surface:

- Constants `DefaultFlatDepth = 0.5`, `PressWall = 1.6`, `BoltLength = 6`, `CavityDepth = 4`, `RimChamfer = 0.28`,
  `EndLedge = 1.5`, `GripTipRound = 2`, `MaxBoreClearance = 0.5`, and the container names `CrankContainerName`
  (`Crank`), `BoltsContainerName` (`Bolts`) and `NutsContainerName` (`Nuts`).
- `Params` with the float fields `ShaftDiameter`, `FlatDepth`, `BoreClearance`, `HubDiameter`, `Thickness`, `Throw`,
  `GripDiameter`, `GripLength` and `Placement *r3.Frame` (nil selects the local frame).
- `Params605ZZ()`, `Params6082RS()` and `DefaultParams()`, which returns `Params605ZZ()`.
- `Params.Validate() error` and `Params.Derive() (Dimensions, error)`.
- `Dimensions` with `BoreRadius` (`r_b`), `BoreFlat` (`f_b`), `BoreLoop` (`L_b`), `HubRadius` (`R_h`), `TailDistance`
  (`t`), `TailHalf` (`w_t`), `ScrewHeight` (`z_s`), `RootRadius` (`rho`), `EndRadius` (`R_e`), `TipRound` (`r_t`)
  and `Site`, the set screw's `m2fastener.Site` in the local frame.
- `Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error)`. `Model` exposes `Crank *decad.Body`,
  `Bolt *m2bolt.Model`, `Nut *m2nut.Model`, the sketches `Plate` and `Screw`, `Dimensions`, and `Site` in the world
  frame. Build adds exactly three bodies.

## Presets and defaults

| Field | `Params605ZZ` | `Params6082RS` |
| --- | --- | --- |
| `ShaftDiameter` | 5, `ballbearing.Params605ZZ().BoreDiameter` | 8, `ballbearing.Params6082RS().BoreDiameter` |
| `FlatDepth` | 0.5 | 0.5 |
| `BoreClearance` | 0.05, `ballbearing.PressFitClearance` | 0.05 |
| `HubDiameter` | 24 | 28 |
| `Thickness` | 10 | 10 |
| `Throw` | 50 | 50 |
| `GripDiameter` | 14 | 14 |
| `GripLength` | 30 | 30 |

A 608ZZ has the same 8 mm bore as a 608-2RS. The 0.5 mm flat is the planetary gearbox shafts' flat.

Derived values:

| Value | 605ZZ | 608-2RS |
| --- | --- | --- |
| `r_b` | 2.55 | 4.05 |
| `f_b` | 2.05 | 3.55 |
| width of the bore's flat, `2 w_b` | 3.03 | 3.90 |
| `L_b` | 15.81 | 25.28 |
| `t` | 10.05 | 11.55 |
| `w_t` | 6.557 | 7.912 |
| `z_s`, `rho`, `R_e`, `r_t` | 5, 5, 13.5, 2 | 5, 5, 13.5, 2 |

## Validation

Every message starts with `crankhandle: `. Derive checks in this order and returns the first failure:

1. Every float field is finite and positive: `<Field> must be finite and positive`.
2. `d <= D/4`.
3. `ballbearing.PressFitClearance <= c <= MaxBoreClearance`.
4. `w_b >= m2fastener.HoleRadius` (1.3), so the bolt's hole opens on the bore's flat and not on its round.
5. `R_h > t`, then `w_t >= m2fastener.HeadPocketRadius + 1` (3.2), so the head pocket keeps 1 mm of material on the
   tail face. The message gives the smallest hub diameter.
6. `R_h - r_b >= PressWall`.
7. `T >= 2 (m2fastener.NutCavityRadius + 1)` (7.02), so 1 mm of material lies above and below the nut cavity.
8. `GripLength >= rho + r_t + 1`.
9. `Throw >= R_h + R_e`, so the arm's end circle does not reach into the hub circle.
10. `Placement` is nil or a valid frame.

Build rejects a nil context and a nil document, derives the dimensions, and checks `ctx.Err()` before it creates any
geometry. A rejected build adds no body.

## Local frame

The origin is on the shaft axis in the crank's back face, the face toward the mechanism. +Z runs along the shaft
axis away from the mechanism. The arm runs along +Y to the grip axis at `(0, Throw)`. The bore's flat, the tail face
and the set screw face -Y. A `Placement` maps this frame onto the given one after the last operation.

## Geometry

1. **Plate outline.** One fully constrained sketch in the XY plane. The hub is a circle of radius `R_h` about the
   origin, cut by the tail face, a line at `y = -t`. The arm's end is a circle of radius `R_e` about `(0, Throw)`.
   Two lines tangent to both circles join them, so the outline is convex and has no inside corner. Extrude it `T`
   along +Z.
2. **Bore.** A D-shaped hole in the same sketch: an arc of radius `r_b` about the origin the long way round through
   +Y, closed by a flat at `y = -f_b`. Every point of the bore lies `c` outside the shaft's D section, on the round
   and on the flat.
3. **Set screw channel.** The set screw lies on the line through `(0, -t, z_s)` along +Y. Its head's top face is
   flush with the tail face, and its tip ends on the bore's flat, at distance `2 + L_s` from the head. One revolved
   tool cuts, from the tail face inward: the head pocket (radius 2.2, depth 2), the clearance hole (radius 1.3), the
   nut cavity (radius 2.51, from 4 to 6 from the head) and the clearance hole again, into the bore. A 45 degree
   chamfer of leg `k` fills the concave rim of the pocket floor and both concave rims of the cavity. The tool's
   meridian runs from 1 mm before the tail face to 1 mm inside the bore.
4. **Surface edge finishing.** Chamfer the outer loop of the back and front faces by 0.5 mm. Chamfer the bore's rim
   on both faces as well when `L_b >= 20` (the 608-2RS crank).
5. **Grip.** A round boss of diameter `GripDiameter` and height `GripLength` stands on the front face at
   `(0, Throw, T)`, with a root fillet of radius `rho` (`decad.Boss` with `RootFillet`). Its top edge is rounded with
   a fillet of radius `r_t`.
6. **Fasteners.** `m2fastener.PlaceBolt` and `m2fastener.PlaceNut` place an M2 x 6 bolt and an M2 nut at the site,
   carried through `Placement`. The nut's corners lie along X, so it stands on a flat when it is dropped in.
7. **Containers.** The crank goes in `Crank`, the bolt in `Bolts` and the nut in `Nuts`, each a child of the document
   root.

The set screw channel is cut before the faces are chamfered (see `steps.md`, "Known decad limit").

## Fasteners

The set screw holds the crank on the shaft along its axis and takes up the bore's clearance toward the flat. The
flat carries the torque. The set screw meets the AGENTS.md fastener rules:

- The head pocket is 2.2 mm in radius (head 1.9 plus 0.3) and 2 mm deep, and the head's top face is flush with the
  tail face. The bolt is modeled with its tip on the bore's flat; tightened, it presses on the shaft's flat and its
  head sinks up to `c` below flush.
- The nut cavity is 2.51 mm in radius and 2.0 mm long. The crank closes it on every side except the bolt hole: 2 mm of
  material lies between it and the pocket floor, 2 mm between it and the bore's flat, `z_s - 2.51` above and below,
  and at least `w_t - 2.51` to each side.
- The clearance hole is 2.6 mm across, and the tip stays inside the bore.

A clamp hub, a slotted hub closed by a cross bolt, was not chosen: the slot weakens the hub, and the bolt crosses the
slot horizontally in the print just as the set screw does.

## Part fits

| Fit | Class | Clearance |
| --- | --- | --- |
| D bore round the shaft | Press fit, the crank turns with the shaft | `c` = 0.05 radial, on the round and on the flat |
| Set screw head in its pocket | Fastener rule | 0.3 radial |
| Bolt shank in its hole | Fastener rule | 0.3 radial |
| Nut in its cavity | Fastener rule | 0.2 past the corners, 0.2 at each face |

`BoreClearance` can be raised to 0.5 for a crank that slides on, so the model is **Mixed**: the bore fit is adjustable
and the bolt and nut keep their nominal size.

The crank is a part pressed on a shaft end outside the frame. [Arbor collars](../../docs/model-pipeline.md#arbor-collars)
asks the consuming model's shaft for a collar between the frame's outer face and the crank's back face.

## Valid range

- `FlatDepth` from just above the width the bolt hole needs (rule 4) to `D/4`.
- `HubDiameter` from 21.1 (605ZZ) or 24.0 (608-2RS) up; Derive names the limit.
- `Thickness` at least 7.02.
- `Throw` at least `R_h + R_e`. Decad currently refuses the face chamfer for some throws (see `steps.md`).

## Representative cases

- `605ZZ`: the default.
- `608-2RS`: an 8 mm bore whose rim is chamfered.
- `regular-fit`: `Params605ZZ` with `BoreClearance` 0.35.
- `long-thin`: `Params605ZZ` with `Throw` 100, `Thickness` 8, `GripDiameter` 18, `GripLength` 45 and `FlatDepth` 1;
  the bore's corners turn more than 45 degrees.
- Placed: the frame with origin `(10, -5, 7)`, U `(0, 1, 0)` and V `(0, 0, 1)`.

## Retained sharp corners

The rule is "Corners and roots" in `docs/model-pipeline.md`. The crank prints on its back face with the shaft axis
vertical, as the press-fit rule asks.

| Corner | Reason |
| --- | --- |
| The two corners of the D bore, where the flat meets the arc | Mating corners: they match the sharp edges of the shaft's flat. They turn `acos(f_b / r_b)`: 36.5 degrees at 605ZZ and 28.8 at 608-2RS, so they need no round there. A deeper flat, such as `long-thin`, turns them more than 45 degrees, and the corner test lists them. |

The section through the set screw's axis turns 45 degrees at each chamfered rim and 90 degrees outward elsewhere, so
it has no inside corner. The grip's root is a stressed root, and its fillet radius `rho` is half the thinner of the
plate and the grip. The tip round `r_t` sits on an outward edge between layers.

The set screw's hole breaks the wall round the bore at the flat. Elsewhere the wall is `R_h - r_b` round the arc and
2 mm between the nut cavity and the flat, both above `PressWall`.

## Printing

Print the crank on its back face with no supports. Pause the print at `z_s + 0.5` (5.5 mm for the presets), drop the
nut into its cavity on a flat, and resume. The layers above close the cavity. Screw the set screw in from the tail
face after the print.

## Checks and images

Check every case above: both sketches fully constrained, the three bodies and their containers, the bore's clearance
round the nominal shaft and its wall, the set screw's head, tip and nut against their pockets, no overlap between the
fasteners and the crank, the chamfers, the grip's root fillet and round, and `internal/cornercheck`. Check the
default's bounds and a `VerifyAll` mesh, and the placed crank against the local one. Register the `crankhandle`
presets in `tools/render`, add the model to `models/README.md` and the gallery image, and run
`scripts/check_repository.py`.
