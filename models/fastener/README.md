# Bolts and nuts

**Build status:** Built and tested.

![M2x10 socket head cap screw](images/hero.png)

This model builds a bolt or a nut from a named size preset. The presets are M2 and M5. Each preset carries one bolt
and the hex nut that pairs with it, and the bolt comes in the preset's standard lengths.

| Preset | Bolt | Nut | Lengths (mm) | Build status |
| --- | --- | --- | --- | --- |
| M2 | Socket head cap screw, ISO 4762 / JIS B 1176 | Hex nut, ISO 4032 / JIS B 1181 | 6, 10, 16, 18, 20, 25 | Built and tested |
| M5 | Hex head bolt, ISO 4017 / JIS B 1180 | Hex nut, ISO 4032 style 1 / JIS B 1181 | 16, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80 | Built and tested |

The thread is not modeled. A shank is a plain cylinder of the nominal diameter and a nut bore is the same nominal
diameter, so the model gives each part's envelope for fitting holes and parts. It does not set thread length,
tolerance or strength class.

**Clearance adjustment:** Nominal. Adjust custom mating geometry to provide fit clearance.

For how to build these from Go, see the [developer notes](../../docs/models/fastener.md).
