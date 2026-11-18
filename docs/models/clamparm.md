# Clamp arm

The model in `models/clamparm` constructs a printed desk-edge clamp with a four-joint friction arm and a mount interface
with one placeholder adapter: the C-shaped frame, a pad, a swivel, two links, a head, the adapter, four joint knobs, two
screw knobs, and the M5 bolts and nuts (25 bodies). The interface at the free end is what later holders build against.

## Construction and API

`Params` holds `LinkLength`, `Opening`, the four joint angles `Pan`, `Shoulder`, `Elbow` and `Wrist`, and an optional
`Placement` frame. `DefaultParams()` returns the defaults. `Params.Derive()` checks the fields in the order
`instructions.md` lists and returns the first failure, or `Dimensions`: the pose frames, the joint points and axes,
every bolt's `m5fastener.Site`, the knob and nut frames and `ScrewKnobGap`. It calls no geometry.

Build in a document from `NewDocument()`, which carries a chain tolerance of 0.01 mm. `BuildParts(ctx, doc, params)` adds
the 13 printed bodies, posed, and files them in no container. `Build(ctx, doc, params)` adds those and the six bolts and
six nuts, in the `Frame`, `Pad`, `Swivel`, `Links`, `Head`, `Adapter`, `Knobs`, `Screw Knobs`, `Bolts` and `Nuts`
containers. A rejected build adds no body.

`InterfaceRecessTool` builds the cutter for the recess and stud hole of an adapter, and the interface constants
(`BossBaseRadius`, `BossHeight`, `RecessRadius`, `RecessClearance`, `StudProtrusion`, `AdapterNutMin`, `AdapterNutMax`,
`AdapterStudClear`) are exported so a holder imports them. The Mount interface section of `instructions.md` lists the
rules an adapter follows.

## Items used

- `models/m5bolt`: the six hex head bolts (M5 x 35, three M5 x 25, M5 x 40 and M5 x 70).
- `models/m5nut`: the six M5 nuts.

`internal/m5fastener` places them from each bolt's `Site`. The model references their constructions and does not copy
their geometry.

## Build and images

```sh
go test ./models/clamparm
go -C tools/render test -count=1 -run 'TestClampArm' .
go -C tools/render run . -model clamparm -shot readme -out ../../models/clamparm/images
```

The readme shot writes `hero.png`, `side.png`, `back.png`, `top.png`, `poses.png`, `exploded.png`, `clamp-section.png`,
`joint-section.png`, `interface-section.png`, `interface.png`, `index.png`, and `parts/frame.png`, `pad.png`,
`swivel.png`, `link.png`, `head.png`, `adapter.png`, `knob.png`, `screw-knob.png`, `bolt-25.png`, `bolt-35.png`,
`bolt-40.png`, `bolt-70.png` and `nut.png`. See [Rendering](../rendering.md) for the presets.

## Verification

`go test ./models/clamparm` checks:

- `TestClampArmBuildCases`, `TestClampArmBuildContainerNames`, `TestClampArmBuildParts`, `TestClampArmBuildVerifyAll`,
  `TestClampArmBuildDefaultBounds`, `TestClampArmBuildPlaced` and `TestClampArmBuildInvalid`: every representative case
  builds 25 bodies in the named containers, every sketch is fully constrained, the placed case matches the default, and
  an invalid input adds no body;
- `TestClampArmParams*`: the defaults, the validation errors in order, that every limit is valid, the derived frames,
  `ScrewKnobGap` and the screw's tip;
- `TestClampArmClamp*`: the frame's bounds, the J1 gap, the jaw-root rounds, the pad and its socket fit, the screw knob
  and the jaw nut with the screw;
- `TestClampArmSwivel`, `TestClampArmHead` and `TestClampArmJoint*`: the swivel and head, the link and joint
  construction, that no printed parts overlap, and the link edge finishing;
- `TestClampArmInterface*`: the constants, the recess tool's sketch, the adapter build, the boss's clearance in the
  recess, the floor over the recess and the stud's tip;
- `TestClampArmRangeOfMotion` and `TestClampArmRangeLimitsMatchCases`: at least 0.5 mm of clearance at each joint's
  limits;
- `TestClampArmFits*`: every fit in the Part fits table;
- `TestClampArmFasteners*`: each head flush and inside its pocket, each nut inside its pocket, no fastener overlapping the
  parts it passes through, and the press fit of the screw knobs' pockets;
- `TestClampArmCorners` and `TestClampArmEdgeFinishing`: `cornercheck` on every printed part with its print axis, and the
  0.5 mm chamfers;
- `TestClampArmMarks`: the knob has 1 hole and the screw knob 2, each through the part;
- `TestClampArmCapacity`: the geometric inputs of the joint capacity estimate.

`go -C tools/render test -count=1 -run 'TestClampArm' .` checks the render presets: every body appears once in the
exploded order and the parts table, no exploded bounding boxes intersect, and the exploded parts fill at least 80% of the
image's width or height.
