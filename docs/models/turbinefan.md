# Turbine fan

The model in `models/turbinefan` constructs a fan of three replaceable blades on one printed hub. `Params.RotorDiameter`
(128 to 224 mm, 192 by default) sizes it. The hub is a plain round cylinder, radius 18.9 and 30 long at 192 mm (14 and 20 at
128 mm), with a D bore for a D-flat shaft. One radial M2 x 10 bolt (M2 x 6 at 128 mm) acts as a set screw: it threads
through a nut in a hex slot open to the back face and its tip presses the shaft's flat, and its head floats in a counterbore
in the round face. Each blade is this model's blade, with its round stud in a radial socket in the hub, held by one
M2 bolt and a nut in a hex slot open to the round face. The same hub has two blind M2 bolt holes in its back face, each with a
nut slot open to the round face, for mounting it on the one-way clutch's adapter ring.

## Construction and API

`Params` holds the shaft diameter and flat depth, the bore clearance, the rotor diameter, the blade pitch, the `Hand` and an
optional `Placement` frame. `Params605ZZ()` and `Params608ZZ()` read the shaft diameter from `ballbearing` and the bore
clearance from `ballbearing.PressFitClearance`, and set `RotorDiameter` to `DefaultRotorDiameter` (192).
`DefaultParams()` returns `Params605ZZ()`. `Params.Derive()` returns the first failure in the order `instructions.md`
lists, or `Dimensions`: `Size`, the bore, the set screw's length, tip, head and counterbore floor, and the set screw's,
blade bolts' and adapter bolts' `m2fastener.Site`s.

`SizeFor(rotorDiameter)` returns the `Size` of a rotor: the hub's radius and length, the blade's tip, root, reference and
hold radii, the stud and socket, the blade bolt's circle, length and nut cavity. The blade scale is the rotor diameter over
128. Every airfoil section, the hub length and the pitch reference radius scale with it. The hub radius is 14 times that
scale but stops at `MaxHubRadius` (18.9), so the clutch mount keeps hex-key access to its ear bolts; the round blade root,
the stud, the socket and the blade bolt circle scale by the hub radius over 14. `Size.Beta(r, pitch)` is the blade angle.
The blade bolt and the set screw lengths are the longest `m2fastener.BoltSize.StandardLengths` entry that fits. `Size` replaces the
fixed hub and blade-joint constants this package exported before `RotorDiameter` existed (`HubRadius`, `HubLength`,
`HubMid`, `BladeBoltAt`, `BladeBoltLength`, `BladeBoltCavity`, `BladeStud...`, `BladeSocketRadius`, `BladeSocketStart`,
`SetScrewBolt`, `BladeRootRadius()`, `BladeTipRadius()` and `Beta`).

Build in a document from `NewDocument()`, which carries a chain tolerance of 0.1 mm; the blade's stud union refuses a plain
document. `BuildParts(ctx, doc, params)` adds the hub and the three blades and files them in no container. `Build(ctx, doc,
params)` adds those, the bolts and the nuts (four of each in `ShaftMount`, five in `ClutchMount`), in the `Hub`, `Blades`, `Bolts`
and `Nuts` containers. A rejected
build adds no body.

The model owns the blade (`BuildBlade(ctx, doc, pitch, size)`, `BuildBladeMirrored(ctx, doc, pitch, size)` and
`BladeSocketTool(ctx, doc, frame, size)`, each taking a `Size` from `SizeFor`, and the joint constants), which
`models/handcrankfan` imports at `RotorDiameter` 128, and reads `PressWall`, `ChamferSize` and `RegularFit` from `crankhandle`. It copies no geometry.

The hub's outline holds the circle, the D bore and the three blade bolt holes. It is extruded `Size.HubLength`, then `edgefinish.Chamfer`
takes the long loops of both faces: the outer circle, and the D loop at the 608ZZ bore. Then it is cut by the three sockets,
each blade's head pocket and hex nut slot, the set screw's counterbore, hole and hex nut slot, and each adapter bolt's blind
hole and hex nut slot. The counterbore is a
revolved meridian with a rounded floor; each nut slot is a hexagon with two straight walls running out through a face, with
its corners and edges rounded by 0.5.

## Print pose and nose rib

Each blade prints lying on the bed, leading edge down, at its rest tilt. `BuildBlade` and `BuildBladeMirrored` cut a
flat on the stud's inner end (`StudFlatDepth` 0.5 into the stud), so the blade stands on that flat and on its leading
edge. `RestTilt(ctx, blade, size, pitch, hand)` finds the tilt at which the loft's lowest point reaches the bed with the
flat on it: 13.10, 12.93 and 13.41 degrees at 128, 192 and 224 mm, at every pitch and hand. It fails when the round root
would stand less than `StudFlatRootClearance` (0.2) over the bed. `PrintPoseFor(size, pitch, hand, tilt)` returns the
frame that moves the blade into the print frame, with the flat on the bed and the stud end corner the flat cut away
`StudFlatDepth` under it.

`BuildBladeHarness(ctx, doc, blade, size, hand, tilt)` builds the nose rib for a posed blade: a wedge-shaped wall on the
bed whose top follows the leading-edge nose `RibGapZ` (0.16) under it, `RibTopWidth` (1.2) wide, from `BrimClearance`
past the leading-edge bed spot to `RibTipMargin` before the tip. It is the intersection of a sheared wedge and a profile
slab, with two fully constrained sketches. `BuildBladePrint(ctx, doc, params)` does all of this and files the blade in
`Blades` and the rib in `Harness` (`HarnessContainerName`). The rib is a print aid, exempt from "Corners and roots";
`instructions.md` "Print pose and nose rib" gives the construction and the constants.

`TestPrintPose`, `TestStudFlat` and `TestStudFlatRootClearance` cover the pose and the flat. `TestBuildBladeHarness`,
`TestBladeHarnessVerifyAll`, `TestBladeHarnessSeparate`, `TestBladePrintBedContact`, `TestCradleFirstLayer`,
`TestBladeBrimClearance`, `TestBladeHarnessOverhangs`, `TestCradleLayers`, `TestRibGap`, `TestRibLayerGap` and
`TestBuildBladePrint` cover the rib, on six cases built once each. `TestTurbineFanCorners` reads each blade lying in its
pose. The rib builds in 40 to 90 ms.

## Clutch mount

`Params.Mount = ClutchMount` bolts the hub to the standard adapter ring (`onewayclutch.BuildAdapter`) instead of gripping the
shaft; the zero value `ShaftMount` is the fan above. The hub is the same plain hub in both modes, with no flange and no clamp
ring. `ParamsForClutch(onewayclutch.Params)` returns a matched fan: the clutch's shaft, the hand that its `Freewheel` drives
forward (`Hand.Freewheel()` is the reverse mapping), the default rotor diameter (`WithRotorDiameter` sets another), the default pitch and `Placement`
`MountFrame(c.Placement)`, which seats the fan on the adapter turned `ClutchTurn` (180 degrees) so each ear bolt lies
between two blades. `Build` adds two M2 x 10
adapter bolts and nuts (`Model.AdapterBolts`, `AdapterNuts`) and no set screw. It does not build the adapter or the clutch:
the caller builds the clutch and the adapter in a `onewayclutch.NewDocument()` and the fan in its own, because both models
file their hub in a container named `Hub`.

The presets `turbinefan/clutch`, `clutch-back`, `clutch-cutaway` and `clutch-exploded` render the fan on the default clutch
with its adapter; the readme shot also writes `clutch.png`, `clutch-back.png`, `clutch-cutaway.png`, `clutch-exploded.png` and
`parts/adapter-bolt.png` and `ear-bolt.png`. The tests `TestTurbineFanClutchParams`, `ClutchBuild`, `ClutchSketches`,
`ClutchHub`, `ClutchAdapterBolts`, `ClutchAccess`, `TestTurbineFanAdapterCuts` and `TestTurbineFanOnClutch` (the fan on the
real clutch and adapter: face interface, stack, every pair) cover it.

## Build and images

```sh
go test ./models/turbinefan -timeout 40m
go -C tools/render run . -model turbinefan -shot readme -out ../../models/turbinefan/images
```

The readme shot writes `hero.png`, `index.png`, `front.png`, `back.png`, `side.png`, `setscrew.png`, `joint.png`,
`exploded.png`, `exploded-back.png`, `sizes.png`, `print.png`, `print-side.png`, `print-end.png`, `print-bed.png` and
`parts/hub.png`, `blade.png`, `bolt.png`, `setscrew.png`, `nut.png` and `harness.png`. The presets `turbinefan/hero`,
`front`, `back`, `side`, `setscrew`, `joint`, `exploded`, `exploded-back`, `sizes`, `print`, `print-side`, `print-end`
and `print-bed` render one view each.

## Verification

Tests build eight cases: both presets at 192 mm, a 0.35 mm bore clearance, a 40 degree and a 10 degree pitch, a placed fan, a
`compact` fan (the 608ZZ preset at 128 mm, which rebuilds the fan as it was before `RotorDiameter` existed) and a `large`
fan (the 605ZZ preset at 224 mm and 40 degrees). Bore tests run on the five bore cases and blade tests on the five pitch
cases; `TestBuildBlade` also builds the blade at 128, 192 and 224 mm and at 10, 25 and 40 degrees, and `TestSizeFor` checks
every size against the 128 mm constants and the design numbers. For each they check the sketches, the twelve bodies and
their containers, both adapter cuts, the bounds, the blade envelope, the bore and its wall, the edge chamfers, the set screw and the blade bolts
with their slots, access to the set screw's path and each blade nut's path, the Regular fits, the blades' pitch and hand,
the copies, the replacement paths and `cornercheck` on the hub and each blade. One extra test builds the left-hand fan and
proves its blades are the right-hand blades reflected in x = 0; the other checks run for the right hand only, because the
hub, bolts and nuts do not depend on the hand.
