# Turbine fan

The model in `models/turbinefan` constructs a fan of three replaceable blades on one printed hub. The hub is a plain round
cylinder, radius 14 and 20 long, with a D bore for a D-flat shaft. One radial M2 x 6 bolt acts as a set screw: it threads
through a nut in a hex slot open to the back face and its tip presses the shaft's flat, and its head floats in a counterbore
in the round face. Each blade is the hand-crank fan's blade with its round stud in a radial socket in the hub, held by one
M2 bolt and a nut in a hex slot open to the round face.

## Construction and API

`Params` holds the shaft diameter and flat depth, the bore clearance, the blade pitch, the `Hand` and an optional
`Placement` frame. `Params605ZZ()` and `Params608ZZ()` read the shaft diameter from `ballbearing` and the bore clearance
from `ballbearing.PressFitClearance`. `DefaultParams()` returns `Params605ZZ()`. `Params.Derive()` returns the first failure
in the order `instructions.md` lists, or `Dimensions`: the bore, the set screw's tip, head and counterbore floor, and the
set screw's and blade bolts' `m2fastener.Site`s.

Build in a document from `NewDocument()`, which carries a chain tolerance of 0.1 mm; the blade's stud union refuses a plain
document. `BuildParts(ctx, doc, params)` adds the hub and the three blades and files them in no container. `Build(ctx, doc,
params)` adds those, the four bolts and the four nuts, in the `Hub`, `Blades`, `Bolts` and `Nuts` containers. A rejected
build adds no body.

The model reuses `handcrankfan` by import (`BuildBlade`, `BuildBladeMirrored`, `BladeSocketTool` and the hub and blade-joint
constants) and reads `PressWall`, `ChamferSize` and `RegularFit` from `crankhandle`. It copies no geometry.

The hub's outline holds the circle, the D bore and the three blade bolt holes. It is extruded 20, then `edgefinish.Chamfer`
takes the long loops of both faces: the outer circle, and the D loop at the 608ZZ bore. Then it is cut by the three sockets,
each blade's head pocket and hex nut slot, and the set screw's counterbore, hole and hex nut slot. The counterbore is a
revolved meridian with a rounded floor; each nut slot is a hexagon with two straight walls running out through a face, with
its corners and edges rounded by 0.5.

## Build and images

```sh
go test ./models/turbinefan -timeout 20m
go -C tools/render run . -model turbinefan -shot readme -out ../../models/turbinefan/images
```

The readme shot writes `hero.png`, `index.png`, `front.png`, `back.png`, `side.png`, `setscrew.png`, `joint.png`,
`exploded.png`, `exploded-back.png`, `sizes.png` and `parts/hub.png`, `blade.png`, `bolt.png`, `setscrew.png` and `nut.png`.
The presets `turbinefan/hero`, `front`, `back`, `side`, `setscrew`, `joint`, `exploded`, `exploded-back` and `sizes` render one
view each.

## Verification

Tests build six cases: both presets, a 0.35 mm bore clearance, a 40 degree and a 10 degree pitch, and a placed fan. Bore tests
run on the three bore cases and blade tests on the three pitch cases. For each they check the sketches, the twelve bodies and
their containers, the bounds, the blade envelope, the bore and its wall, the edge chamfers, the set screw and the blade bolts
with their slots, access to the set screw's path and each blade nut's path, the Regular fits, the blades' pitch and hand,
the copies, the replacement paths and `cornercheck` on the hub and each blade. One extra test builds the left-hand fan and
proves its blades are the right-hand blades reflected in x = 0; the other checks run for the right hand only, because the
hub, bolts and nuts do not depend on the hand.
