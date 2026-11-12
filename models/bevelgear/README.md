# Bevel gear pair

Two meshing bevel gears whose shafts meet at an angle, shown as two pairs side by side. The pair on the
left has spiral teeth: 31 teeth on each gear at Module 1 with 35° spiral teeth on a 90° shaft angle. The
pair on the right has straight teeth: 32 and 16 teeth at Module 4. Every gear has a through-bore for its
shaft. The two gears of a pair share one pitch apex and are rotated so their teeth mesh.

![Bored 31/31 spiral bevel gear pair beside a bored 32/16 straight-tooth pair](images/hero.png)

The tooth root has no fillet.

## More views

The spiral pair:

| From above | High oblique angle | Low side angle |
| --- | --- | --- |
| ![Both spiral gear members and their meshing tooth bands from above](images/top.png) | ![Both spiral gear members from a high oblique angle](images/high.png) | ![Both bored spiral gear members from a low side angle](images/low.png) |

The 32/16 straight-tooth pair:

| From above | High oblique angle | Low side angle |
| --- | --- | --- |
| ![Both straight-tooth gear members from above](images/straight-top.png) | ![Both bored straight-tooth gear members from a high oblique angle](images/straight-high.png) | ![Both bored straight-tooth gear members from a low side angle](images/straight-low.png) |

## Exploded

Each pair pulled apart along its two shafts. The driving gear moves out along its shaft and the pinion along its
own, both away from the shared pitch apex.

| Spiral pair, 31/31 | Straight pair, 32/16 |
| --- | --- |
| <img src="images/exploded-spiral.png" alt="The two spiral gears pulled apart along their shafts" width="420"> | <img src="images/exploded-straight.png" alt="The 32-tooth driving gear and the 16-tooth pinion pulled apart along their shafts" width="420"> |

## Parts

A pair has two parts, the driving gear and the pinion. The two spiral gears are the same part, one mirrored
from the other, so they have one row. The straight pair has two different parts.

| Image | Name |
| --- | --- |
| <img src="images/parts/spiral-gear.png" alt="Spiral bevel gear, 31 teeth" width="240"> | Spiral gear (31 teeth; driving gear and pinion, mirror images) |
| <img src="images/parts/straight-driving.png" alt="Straight bevel driving gear, 32 teeth" width="240"> | Straight driving gear (32 teeth) |
| <img src="images/parts/straight-pinion.png" alt="Straight bevel pinion, 16 teeth" width="240"> | Straight pinion (16 teeth) |

You can vary:

- Straight teeth: any tooth count and the supported options.
- Spiral teeth: equal tooth counts at Module 1, the cutter radius, hand, face width, base height, tooth
  spacing, toe extension, toe radius, pressure angle and spiral angle.
- The module and the shaft angle.
- The bore of each gear.
- The position and orientation of the pair in space.

For how to build these pairs from Go, see the [developer notes](../../docs/models/bevelgear.md).
