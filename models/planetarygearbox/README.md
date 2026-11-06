# Planetary gearbox

![The default two-stage planetary gearbox, pulled apart along its axis: ring gear, two suns, six planets and two carriers](images/hero.png)

A compact planetary speed reducer. A fixed ring gear with teeth on its inside is the housing. A small sun gear
turns three planet gears that roll around it inside the ring, and the planets' pins ride on a carrier plate that
turns slowly. The sun is the input and the carrier is the output.

The default gearbox has a 13-tooth sun, three 29-tooth planets and a 71-tooth ring. One stage slows the output to
1 ÷ 5.46 of the input speed (1 + 71 ÷ 13). The image shows two stages in the same ring, pulled apart: the first
carrier's stub shaft drives the second sun, so two stages slow the output to 1 ÷ 29.8.

You can vary:

- The number of stacked stages in the one ring.
- The tooth counts of the sun, planets and ring, and the number of planets.
- Module and pressure angle.
- Gear and carrier thickness, and the gaps between them.
- The shaft, pin and output shaft sizes, and the ring wall.
- The position and orientation of the gearbox in space.

The ring must have as many teeth as the sun plus twice the planets, and the sun and ring teeth together must divide
by the planet count so the planets sit at equal angles. Dimensions are in millimeters, and the pressure angle is in
radians. The gears mesh with no backlash.

For how to build this gearbox from Go, see the [developer notes](../../docs/models/planetarygearbox.md).
