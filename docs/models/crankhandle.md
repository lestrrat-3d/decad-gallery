# Crank handle

The model in `models/crankhandle` constructs a printed hand crank for a D-flat shaft: an arm, a pin, a handle that turns
on the pin, and the M2 bolt and nut that clamp the pin. `models/handcranksiren` builds its crank from it.

## Construction and API

`Params` holds the shaft diameter and flat depth, the bore clearance, the throw and an optional `Placement` frame. The
arm, pin and handle keep fixed sizes, exported as constants with the handle's meridian `HandleProfile`.
`Params605ZZ()` and `Params6082RS()` read the shaft diameter from `ballbearing.Params605ZZ()` and
`ballbearing.Params6082RS()`, and the bore clearance from `ballbearing.PressFitClearance`. `DefaultParams()` returns
`Params605ZZ()`. `Params6082RS()` is the siren's crank. `Params.Derive()` returns the first failure in the order
`instructions.md` lists, or `Dimensions`: the bore's radius, flat and outline length, and the pin bolt's
`m2fastener.Site`.

`BuildParts(ctx, doc, params)` adds the arm, the pin and the handle and returns them with every sketch and the pin
bolt's site in the world frame. It files them in no container, so an assembly files them with its own parts.
`Build(ctx, doc, params)` adds those three, the M2 x 10 bolt and the nut, in the `Arm`, `Pin`, `Handle`, `Bolts` and
`Nuts` containers. A rejected build adds no body.

The arm is an extruded stadium with both faces' outlines chamfered, then cut by a blind D bore, the hexagonal pocket
for the pin's stub and the bolt's head pocket. The pin joins a hexagonal stub, a round shank and a head, then takes the
bolt's hole and the nut's cavity. The handle is a revolved meridian.

## Build and images

```sh
go test ./models/crankhandle
go -C tools/render run . -model crankhandle -shot readme -out ../../models/crankhandle/images
```

The readme shot writes `hero.png`, `index.png`, `side.png`, `back.png`, `cutaway.png`, `exploded.png`, `sizes.png` and
`parts/arm.png`, `pin.png`, `handle.png`, `bolt.png` and `nut.png`. The presets `crankhandle/hero`, `side`, `back`,
`cutaway`, `exploded` and `sizes` render one view each.

## Verification

Tests build four cases: both presets, a 0.35 mm bore clearance, and an 80 mm throw. For each they check every sketch's
full constraint, the five bodies and their containers, and `cornercheck` on the arm, pin and handle with the hexagonal
pocket's corners listed. They cut sections of the arm and require every point of the bore to lie `BoreClearance`
outside the shaft's D section, the bore to stop at its floor, and the wall round it to be at least 1.6 mm. They measure
the stub's gap in its pocket (Tight), the handle's gap round the shank (Regular), and the handle's gaps to the pin and
the arm in a cut through the pin axis (Regular). They require the bolt's head to be flush with the arm's back face, its
tip inside the pin, the nut inside its cavity, and no overlap between the fasteners and the parts. The default's bounds
and `VerifyAll` meshes, the placed crank and the validation errors are checked too. The siren's tests check its crank
through the assembly.

## Known decad limit

The 608-2RS bore's rim in the arm's back face is not chamfered: Decad refuses the chamfer. `steps.md` quotes the error.
