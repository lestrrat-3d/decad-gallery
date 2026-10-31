# Spinning-wheel arrangement audit

## Scope and evidence

Compare the two photographs in [George Weil's labelled image](https://www.georgeweil.com/wp-content/uploads/2025/09/spinning-wheel-parts-labelled-1.jpg)
with Gallery commit `71c2e50`. Complete this inventory before changing construction.
Use Ashford's Traditional assembly diagrams to resolve perspective and hidden joints.
The local reference manual is `.tmp/spinningwheel-reference/ashford-assembly.pdf`;
its pages 5–8 show the treadle and universal joint, and pages 9–11 show the single-drive head.
The photographs establish arrangement, not manufacturing dimensions.
`models/spinningwheel/instructions.md` owns the replacement dimensions and acceptance checks.

## Complete part inventory

| Reference part | Existing model | Required correction |
| --- | --- | --- |
| Flywheel | The hub is only 430 mm high; eight spokes and a wide head-to-wheel gap change the silhouette. | Raise the hub to a nominal 580 mm, shorten the horizontal head offset, and use six spokes. |
| Drive band | One band joins parallel pulley axes, but reaches a whorl behind the bobbin. | Keep the single band and ratios; align it with the front whorl. |
| Flyer | The U faces the operator, but its guide eyes are closed rings with four stations per arm. | Keep the U orientation; use six open hook stations per arm and an open front routing hook. |
| Orifice | The hollow opening faces the operator. Its projection is exaggerated by the rear whorl layout. | Keep the direction and 10 mm passage; shorten the projection relative to the band plane after relocating the whorl. |
| Bobbin | The bobbin turns independently inside the U. | Keep independent take-up and its rear brake groove. |
| Brake band | The band reaches the rear groove, but its single spring and screw sit behind the head. | Route both tails forward below the flyer, with two springs and an accessible front control. |
| Tension controls | The drive control is a horizontal leadscrew and the brake control is at the rear. | Use a vertical drive jack on a hinged mother-of-all and put the brake knob at the front. |
| Maidens | Two separate posts support the shaft. The front post is immediately before the flyer crossbar. | Make room for the whorl between the front maiden and crossbar; carry both posts on the tilting board. |
| Mother-of-all | Three broad stacked plates slide horizontally; there is no hinge. | Replace them with a narrow hinged upper board, base, alignment slots and vertical jack. |
| Footman | A flat four-bar has parallel pin axes and runs on the operator side. | Move it behind the wheel; use an upper two-axis gimbal and a compliant lower connector. |
| Treadle | A sideways board rocks about an axis parallel to the wheel axle. | Put a fore-and-aft footboard on a cross-machine rocker shaft, with support at both ends. |
| Treadle connector | A long lateral toe pin directly joins the sideways board to the rod. | Add the diagonal actuating rail and short flexible connector shown in the reference arrangement. |
| Supporting frame | Vertical posts stand on separate feet, with a high horizontal bench behind the wheel. | Use one head leg, a splayed wheel-support pair, two sloping rails and a shared treadle shaft. |

## Eight correction groups

1. Rotate and reposition the treadle, hinge and footboard.
2. Replace the planar linkage, relocate the crank/footman, and add the rocker rail and articulating connections.
3. Correct wheel/head proportions and replace the frame layout.
4. Relocate the one-piece whorl, front support and shaft retainers.
5. Replace the drive-tension slide with a hinged mother-of-all and vertical jack.
6. Bring the Scotch control forward and fit two springs.
7. Match the six-spoke wheel layout.
8. Replace closed guide eyes with six open hooks on each arm.

## Deliberate construction choices

Keep purchased 608-2RS and 605ZZ mounting envelopes, D-shaped positive couplings and removable bearing supports.
Use straight timber sections and a rectangular flyer rather than copying decorative turnings.
Use a front M4 screw for brake take-up rather than an unmodeled friction peg.
Use a 605-based upper gimbal and an ideal compliant lower connector; do not force skew pivots through a plain bearing.
Keep prescribed yarn feed in the animation. Do not claim simulated forces, flexure strain, traction or bearing life.
Nominal proportions inferred from photographs are gallery dimensions, not measurements of an Ashford product.

## Acceptance

First pass the real default assembly through all sketch and final-solid checks.
Then test scale, ratio and adjustment extremes; prove linkage closure and joint-axis alignment throughout a full turn.
Check actual moving geometry against the frame, both cords against timber, and the flyer against stationary head parts.
Render front/back, side and top views so depth cannot conceal a reversed component.
Regenerate the close-ups and animation from the corrected Decad bodies.

## Integration findings

The first native assembly exposed repeated-placement bookkeeping in Decad's exact-bore feature.
[Decad PR2932](https://github.com/lestrrat-3d/decad/pull/2932) repaired it generally and passed required CI.
The current pinned dependency includes that repair.

The straight diagonal actuator's mesh intersected the rear leg near the bottom of the stroke.
The corrected path stops the diagonal left of that leg and reaches the lower connector through a rear crossarm.
This preserves the solved joint positions and gives the moving timber room around the support.
The clearance checks charge verified mesh displacement and also reject containment, rather than accepting surface gaps alone.
The Scale0.8 case also found insufficient clearance between the crank's front collar and the splayed rear leg.
Moving the crank10nominalmm rearward preserves the solved crank-pin centre while clearing the stock collar.

## Verified correction results

The corrected default has181finalbodies and198verifiedsketches.
The default and all four specified scale/ratio/adjustment cases pass final-solid, joint-fit and clearance checks.
Each case checks72solved poses against the fixed frame, including the offset rear rail and crank hardware.
The renderer checks ten actual moving-body joint locations at four phases, including both gimbal axes and both clamps.
These checks establish the modeled interfaces; the prototype still needs force, traction and flexible-connector checks.
