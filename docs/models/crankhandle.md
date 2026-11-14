# Crank handle

The model in `models/crankhandle` constructs a printed crank handle for a D-flat shaft, with an M2 set screw and nut.

## Construction and API

`Params` holds the shaft diameter and flat depth, the bore clearance, the hub diameter, the plate thickness, the throw,
the grip's diameter and length, and an optional `Placement` frame. `Params605ZZ()` and `Params6082RS()` read the shaft
diameter from `ballbearing.Params605ZZ()` and `ballbearing.Params6082RS()`, and the bore clearance from
`ballbearing.PressFitClearance`. `DefaultParams()` returns `Params605ZZ()`. `Params.Derive()` returns the first failure
in the order `instructions.md` lists, or `Dimensions`: the bore's radius, flat and outline length, the tail face's
distance and width, the set screw's height and `m2fastener.Site`, the grip's root fillet, the arm end's radius and the
grip's top round.

`Build(ctx, doc, params)` returns a `Model` with the `Crank` body, the `Bolt` and `Nut` models, the `Plate` and
`Screw` sketches, `Dimensions`, and the set screw's `Site` in the world frame. It adds three bodies to `doc`, in the
`Crank`, `Bolts` and `Nuts` containers. A rejected build adds no body.

Build extrudes the plate sketch (hub, tangent arm, end circle and D bore), cuts the set screw channel with one revolved
tool, chamfers the outer loop of both faces (and the bore's rim when its outline is 20 mm or longer), raises the grip
with `decad.Boss` and a root fillet, rounds the grip's top edge, and applies the placement. `internal/m2fastener` places
the M2 × 6 bolt and the nut.

## Build and images

```sh
go test ./models/crankhandle
go -C tools/render run . -model crankhandle -shot readme -out ../../models/crankhandle/images
```

The readme shot writes `hero.png`, `index.png`, `side.png`, `back.png`, `tail.png`, `cutaway.png`, `exploded.png`,
`sizes.png` and `parts/crank.png`, `parts/bolt.png` and `parts/nut.png`. The presets `crankhandle/hero`, `side`, `back`,
`tail`, `cutaway`, `exploded` and `sizes` render one view each.

## Verification

Tests build four cases: both presets, a 0.35 mm bore clearance, and a long, thin crank with a 1 mm flat. For each they
check both sketches' full constraint, the three bodies and their containers, and `cornercheck` on the crank with the
D bore's corners listed when they turn more than 45 degrees. They cut sections of the crank and require every point
of the bore to lie `BoreClearance` outside the shaft's D section, and the wall round the bore to be at least 1.6 mm.
They require the bolt's head to be flush with the tail face and its tip on the bore's flat, the nut to lie inside its
cavity, material round the cavity on every side, and no overlap between the crank and the bolt or the nut. They check
the 0.5 mm face chamfers, the bore rim's chamfer when its outline is 20 mm or longer, and the grip's root fillet and
diameter. The default's bounds and `VerifyAll` mesh, the placed crank, and the validation errors are checked too.

## Known decad limit

Decad refuses the face chamfer after the set screw cut for some throws, such as 30 and 65 to 90 mm at the 605ZZ
preset; chamfering first makes Decad refuse the cut. `steps.md` quotes both errors.
