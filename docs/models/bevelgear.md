# Bevel gear package

The `models/bevelgear` package builds the pairs that the
[bevel gear pair gallery page](../../models/bevelgear/) shows. The construction rules come from
`fusion360-gear-generator/spec/bevelgear/instructions.md`; they are recorded in
[`instructions.md`](../../models/bevelgear/instructions.md), and the
[steps](../../models/bevelgear/steps.md) name the Sketch and Decad operations.

## Using the package

`Build(ctx, doc, Params)` constructs one gear member. A straight member takes any tooth count and
option. A spiral member takes equal tooth counts at Module 1 and any cutter radius, hand, face width,
base height, tooth spacing, toe extension, toe radius, pressure angle and spiral angle.
`DefaultParams()` selects Module 1, a 90° shaft angle, 35° spiral teeth, and an auto-calculated bore
for each member. `BuildPair` builds and places all four supported pairs. The document passed to `Build`
needs a chain tolerance for spiral members; the doc comment on `Build` in `bevelgear.go` explains why
and how long a spiral build takes.

Set `Params.Placement` (an `r3.Frame`) to anchor the pair. The frame origin is the common pitch apex, its
U and V axes span the plane that holds both shafts, and the driving shaft runs along U. A nil `Placement`
keeps the world origin and XY shaft plane.

## How a spiral member is built

Each spiral member has a revolved blank extended past its caps, one curved tooth trimmed past the toe and
heel, 31 patterned teeth joined to the blank, a single trim to the caps, and a 7.75 mm through-bore cut
after the join. The pinion and driving member share a pitch apex and follow the source's meshing
rotations. The tooth band and the smooth back face meet at a sharp root, because the source joins them
into one solid without a root fillet.

## Images

The [renderer](../rendering.md) builds the source-default pair and the 32/16 straight pair for the hero
image. It enlarges the Module 1 spiral pair fourfold so both pairs have the same pitch diameter.

## Checks

The real-body checks are slow, so the test file `bevelgear_test.go` lists each one with its command and
timeout. They check a closed outer shell before the bore, two placed solids with through-holes afterward,
and `VerifyAll` for each member.
