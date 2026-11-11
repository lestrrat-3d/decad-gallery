# Hand-crank siren steps

Each step maps a requirement of `instructions.md` to a Sketch or Decad operation. Every sketch is solved, verified
fully constrained and recorded in `Assembly.Sketches`. All dimensions are in `instructions.md`; this file gives the
operations, their order and the checks.

## 1. Frames, parameters and helpers

- `params.go`: `Ratio` (10), `Params`, `DefaultParams`, `Validate`, `PitchHz`, `State`, `Motion`. `document.go`:
  `NewDocument` with a chain tolerance of 0.01 mm. `fits.go`: `tightFit` 0.25, `regularFit` 0.40, `looseFit` 0.55,
  `seatClearance` 0.05.
- `build.go`: `Build` validates, builds every part at crank angle 0, poses it by its motion, records it and returns
  every sketch. The builder keeps the first error (`fail`) and wraps it as `hand-crank siren <part>: <operation>: %w`.
  `part(name, group, motion, create)` poses through `motionTransform`, which rotates about the line through the axis
  point along −Y.
- `sections.go`: `frame`, `section` (new sketch on a frame, recorded), `profile` (solve, verify fully constrained,
  select the one valid profile with the wanted hole count), `extrude`, `revolve`, `cylinder`, `cut`, `union` (folds
  left to right), `turn`, `roundedPolygon`, `meridianPolygon`, `port`, `dBoreTool`, `dShaft`.
- `meridianPolygon(s, pts)` draws a closed polygon in (u, v). A segment with equal v is horizontal with a
  distance-to-origin of |v|. A segment with equal u is vertical with a distance of |u|. A vertex between a horizontal and
  a slanted segment gets a horizontal distance from the origin; between a vertical and a slanted segment a vertical
  distance; between two slanted segments `sketchcheck.DimensionPoint`. A vertex between two axis-aligned segments gets
  nothing more. It returns an error when an axis-aligned segment passes through the origin.
- `port(s, rIn, rOut, half)` draws the chord-sided port centred on +U with points p1 (rIn cos h, −rIn sin h),
  p2 (rIn cos h, +rIn sin h), p3 (rOut cos h, −rOut sin h), p4 (rOut cos h, +rOut sin h) and lines p1–p2, p3–p4, p1–p3,
  p2–p4. Constraints: p1–p2 and p3–p4 vertical at distances `rIn cos h` and `rOut cos h` from the origin; the origin on
  lines p1–p3 and p2–p4; `NewVerticalDistance` of ∓`rIn sin h` on p1 and p2. DOF 0.
- `dBoreTool(axis, from, length, clearance)`: a cylinder of radius 3.75 + clearance minus a box starting 3.0 +
  clearance from the axis toward +Z, ±5 across, 2 longer than the cylinder. Cut the box from the cylinder, then the tool
  from the host.
- `dShaft(axis, from, to)`: a cylinder of radius 3.75 minus a box beyond 3.0 toward +Z along its whole length, with 1 of
  stock at each end. Cut before any union with a collar.
- Check: `TestValidate`, `TestMotion`, `TestPitch`.

## 2. Gears (`gears.go`)

1. `spurgear.Variant{Define: gearroot.Define(gearroot.Clearance(1.5), 0.02)}.Build` with `spurgear.DefaultParams()`,
   Module 1.5, the teeth, Thickness 6, Backlash 0.15, BoreDiameter 0, Samples 15, `CapChamfer` 0.2 (50T, 48T) or 0
   (16T, 15T), Placement origin on the axis at the plane's front Y, V = (0, 1, 0) × U.
2. U per gear as in the table of `instructions.md` (the driven gears turned by π/16 and π/15 about +Y).
3. `dBoreTool(axis, planeFront − 1, 8, 0.25)` cut from each gear.
4. Mark holes: for each gear, one section at the bore's start per hole, one circle of radius 0.75 at 6.6 from the
   axis (25° apart, symmetric about +V), extruded 8 and cut. 1 hole for the 50T and 16T gears, 2 for the 48T and 15T.
- Check: the `mesh` and `marks` subtests.

## 3. Shafts and bearings (`shafts.go`, `bearings.go`)

- `dShaft` for each shaft. Union the collars last (rotor-shaft [10.1, 12.1], crank-shaft [19.1, 20.1], radius 5.5).
- `ballbearing.Build` with `DefaultParams()` and Placement origin at the bearing's start Y on its axis, normal +Y.
  Append `Model.Sketch` to the sketches. `BearingSites()` lists the six bearings with their holders.
- Check: the `clearances` subtest rows T4, T5, T6, T7, T9, T10, L3.

## 4. Gearbox (`gearbox.go`)

1. gearbox-front: rounded outline (`roundedPolygon`, radius 13) extruded 10 from Y 0. Cut seat pockets and reliefs on
   R, I and C, the R through hole, then `edgefinish.Chamfer(body, +Y, {0, 10}, Long(All), 0.5)`, then the fastener cuts.
2. gearbox-rear: the same from Y 27.6, seats open to the front, C through hole, offsets {27.6, 37.6}.
3. gearbox-frame: one sketch with the outline and the cavity (radius 6), profile with 1 hole, extruded 17.1, offsets
   {10.25, 27.35}, then the bolt holes and nut cavities.
4. gearbox-front also gets the hole and cavity for each `grip-bolt-*`, after the drum-bolt cuts.
- Check: `fasteners` subtest; rows T1, S1, S2, L2, L4.

## 5. Siren head (`drum.go`, `rotor.go`, `stator.go`, `collector.go`, `horn.go`)

Meridian sketches of the drum and rotor sit on origin (0, 0, 84), U = +X, V = +Y, normal +Z, and revolve about
`ConstructionAxis{(0, 0, 84), (0, 1, 0)}`. The horn's meridian has its own origin and axis (see `horn.go`).

- `drum.go`: revolve the drum `meridianPolygon` (with the shaft hole's 0.5 chamfers); cut three
  windows (`port(14, 24, 40°)` at Y −16.50, U at 60°, extruded 6, `Patterned` Count 3, then `Cut`);
  `edgefinish.Chamfer(body, +Y, {−11.50, −15.50}, windows only (Long(Inner), more than one edge), 0.5)`; union three
  posts (radius 5 at radius 30, patterned); fastener cuts for drum-bolt and collector-bolt.
- `rotor.go`: disc radius 45 over [−27.50, −24.50]; chamfer both cap outer loops 0.5 before any cut; vanes patterned
  Count N; hub radius 9; `Union(vanes, hub)` then `Union(disc, that)`; ports `port(30, 42, π/(2N))` at Y −28.50,
  extruded 5, patterned Count N, cut; D-bore `dBoreTool(R, −25.50, 10, 0.25)`.
- `stator.go`: disc radius 58 over [−31.65, −28.65]; cap outer loops chamfered 0.5; the same ports; hole for each
  collector-bolt.
- `collector.go`: `collector()` in eight steps:
  1. outer prism `roundedPrism(rotor.across(−96.90), outline(−51, 51, 33, 135), {10,10,10,10}, 65)`;
  2. cut the bore, `cylinder(rotor.across(−40.90), 44, 10)`;
  3. cut the cavity, `roundedPrism(rotor.across(−91.90), outline(−46, 46, 38, 130), {7,7,7,7}, 52)`;
  4. `finish(body, −31.90, −96.90)`;
  5. `edgefinish.Chamfer(body, +Y, {−39.90}, Long(Inner), 0.5)` for the bore rim on the ceiling;
  6. cut the throat, `cylinder(frame((−45, throatY, 84), U (0,0,1), V (0,1,0)), 22, 7)`;
  7. `edgefinish.Chamfer(body, +X, {−51, −46}, Long(Inner), 0.5)` for the throat rim;
  8. hole and cavity for each collector-bolt, then pocket and hole for each throat-bolt.
- `horn.go`: the meridian sketch has origin O = (−50, throatY, 84), U = +Y, V = −X, normal +Z. `meridianPolygon` of the
  vertices A to L, then `revolve` about `ConstructionAxis{Origin: O, Dir: (−1, 0, 0)}` with `FullRevolution`, then the
  hole and cavity for each throat-bolt.
- Check: the `ports` and `outlet` subtests; rows T2, T3, T11, T5, T6, R1, R2, L1, L6.

## 6. Crank (`crank.go`)

- crank-arm: stadium on Y 38.6 (two arcs of radius 10 with `NewEqualRadius`, four `NewTangent` lines, pin circle radius
  1.3), extrude 8 selecting the profile with 1 hole, chamfer cap outer loops 0.5, `dBoreTool(C, 37.6, 8, 0.25)`, the
  hexagon pocket, the pin-bolt head pocket.
- crank-pin: hexagon, shank, head unioned in that order, then hole and cavity.
- crank-handle: `meridianPolygon` about the pin axis with origin 1 behind the inner end, revolved.
- Check: the `sweep` subtest; rows T5, T6, T8, R3, R4, L5.

## 6a. Grip (`grip.go`)

1. `section(frame((79, 0, 0), U (0,1,0), V (0,0,1)))`, then `roundedPolygon` of the outer outline
   {(−76.25, 5), (−0.25, 5), (−0.25, 125), (−76.25, 125)} with radii 10 and of the window
   {(−46.25, 12), (−16.25, 12), (−16.25, 118), (−46.25, 118)} with radii 3, both counter-clockwise in (u, v).
2. `extrude(s, 1, 26)`.
3. `edgefinish.Chamfer(body, +X, {79, 105}, Long(All), 2.0)`.
4. Pocket and hole for each grip-bolt. `grip()` records the part in the `Grip` container, then the hardware for each
   grip-bolt site. `gearboxFront` cuts the hole and cavity of each grip-bolt (step 4 of section 4).
- Check: rows T12, L7; the `grip`, `knuckle room` and `window height` subtests.

## 7. Fasteners (`fasteners.go`)

`fastenerSite` wraps `m2fastener.Site` with names, hosts and motion. `hardware(site)` adds the bolt part and then the nut
part, appending their sketches. `fastenerCuts` cuts the pocket, hole and cavity tools from `internal/m2fastener` per
host (the flags decide which). The tools follow the AGENTS.md clearances: head pocket radius 2.2, hole radius 1.3, round nut cavity of radius 2.51
and height 2.0. `Fasteners(p)` in `export_test.go` poses every site. Site order: gearbox-front,
gearbox-rear, drum, collector, throat, grip, crank-pin. The prefixes are `collector-bolt` (nut host collector),
`throat-bolt` (collector to horn) and `grip-bolt` (grip to gearbox-front).

## 8. Containers and pose (`containers.go`)

`moveBodies` runs last. For each part it calls `doc.Root().ChildNamed(group)` when it adds the body, so no empty
container exists. The grip goes into its own `Grip` container and the collector into `Housing`. Gears go into `Gear Pairs` with `NewUniqueChild("Pair")` for stage 1 and again for stage 2.

## 9. Checks

`TestHandCrankSirenDefault` runs `checkSiren` with the subtests `parts`, `sketches`, `containers`, `mesh`,
`clearances`, `fasteners`, `ports`, `outlet`, `grip`, `knuckle room`, `window height` (these three default only), `sweep`, `orientation`, `tessellate`, `edges` and
`overlap`.
`TestHandCrankSirenVariant` runs `Slots` 12 (CrankAngle 0) and 24 (CrankAngle 5π/4) with parts, sketches, containers,
ports, outlet, orientation, overlap, and rows R2 and L1. `TestValidate`, `TestMotion` and `TestPitch` check the parameters.
