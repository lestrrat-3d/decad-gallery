# Planetary gearbox

![The default two-stage planetary gearbox, fully assembled, with the ring gear and both end covers cut open to show the planets, carriers, shafts and bearings inside](images/hero.png)

A compact planetary speed reducer. A fixed ring gear with teeth on its inside is the housing, closed at both ends by
an open-frame cover. A small sun gear turns three planet gears that roll around it inside the ring. Each planet turns
on a 608ZZ bearing pressed into a pocket in the gear and rides on a pin of the carrier, a three-armed spider that turns
slowly. The sun is the input and the carrier is the output.

The default gearbox has a 13-tooth sun, three 29-tooth planets and a 71-tooth ring. One stage slows the output to
1 ÷ 6.46 of the input speed (1 + 71 ÷ 13). The image shows two stages in the same ring: the first carrier's stub shaft
drives the second sun, so two stages slow the output to 1 ÷ 41.8. The ring and both covers are cut open along a
plane through the axis; every other part is shown whole.

## Assembled

The complete two-stage gearbox with nothing cut away, from the output end, from the input end and from the side.

| From the output end | From the input end | From the side |
| --- | --- | --- |
| <img src="images/assembled-output.png" alt="Assembled two-stage gearbox seen from the output end" width="320"> | <img src="images/assembled-input.png" alt="Assembled two-stage gearbox seen from the input end" width="320"> | <img src="images/assembled-side.png" alt="Assembled two-stage gearbox seen from the side" width="320"> |

## How it moves

![The two-stage gearbox turning, seen cut open, from the input end and from the output end, with the input and output shaft angles read out underneath](images/motion.gif)

The input shaft and the first sun turn together. The planets roll inside the fixed ring, so each planet turns on its
own bearing while the carrier carries it around the sun. The first carrier turns at 13 ÷ 84 of the sun's speed and
drives the second sun, which drives the second carrier and the output shaft at the same ratio again. The flat on
each shaft shows its turning: the input shaft's flat sweeps round, and the output shaft's flat creeps. The readout
below the panels gives both angles. The animation runs 56 frames and repeats from the start, so the second stage and
the output shaft jump back when it does. It moves the real tessellated bodies rigidly; it is not a force simulation.

## How the input and the output connect

Every shaft is 8 mm across, the bore of a 608ZZ, with a 1 mm flat along one side (a D). The sun's bore is the same D, so
the shaft turns the sun without slipping.

- **Input.** The input shaft enters through the front cover's 608ZZ and fills the first sun's bore. It sticks out
  12 mm beyond the front cover's bearing boss, where a motor or a crank attaches.
- **Between stages.** The first carrier's stub shaft fills the second sun's bore, so the second sun turns with the
  first carrier.
- **Output.** The last carrier's stub shaft passes through the rear cover's 608ZZ and sticks out 12 mm beyond its
  boss. It is the output shaft.

The model does not show a set screw, the screws in the four bolt holes that join the covers and the ring, or the
coupling to a motor. The flats are where a coupling or a set screw would grip.

## Parts

Each part is shown once, from the real model. A gearbox has one ring, two covers, two cover bearings and an input
shaft and, in every stage, one sun, three planets, three planet bearings and one carrier. The bearing is a purchased
608ZZ, drawn as its 8 × 22 × 7 mm envelope; the covers and the planets use the same part. The front cover is the rear
cover turned over. A stage's carrier is the linking carrier when another stage follows it, because its stub shaft
drives that stage's sun. The last stage's carrier is the output carrier, and its shaft is the output. The default
one-stage gearbox has only the output carrier.

| Image | Name |
| --- | --- |
| <img src="images/parts/ring.png" alt="Ring gear" width="240"> | Ring gear |
| <img src="images/parts/cover.png" alt="Cover" width="240"> | Cover (front and rear) |
| <img src="images/parts/sun.png" alt="Sun gear" width="240"> | Sun gear |
| <img src="images/parts/planet.png" alt="Planet gear" width="240"> | Planet gear |
| <img src="images/parts/bearing.png" alt="608ZZ bearing" width="240"> | Bearing (608ZZ) |
| <img src="images/parts/input-shaft.png" alt="Input shaft" width="240"> | Input shaft |
| <img src="images/parts/linking-carrier.png" alt="Linking carrier" width="240"> | Linking carrier |
| <img src="images/parts/output-carrier.png" alt="Output carrier" width="240"> | Output carrier |

You can vary:

- The number of stacked stages in the one ring.
- The tooth counts of the sun, planets and ring, and the number of planets.
- Module and pressure angle.
- Gear, carrier and cover thickness, and the gaps between them.
- How far the input and output shafts reach out of the covers.
- The carrier wall, the bearing pocket clearance, the ring wall and the bolt hole size.
- Backlash.
- The position and orientation of the gearbox in space.

The ring must have as many teeth as the sun plus twice the planets, and the sun and ring teeth together must divide
by the planet count so the planets sit at equal angles. Dimensions are in millimeters, and the pressure angle is in
radians. Each pair of meshing gears has 0.1 mm of backlash by default, set by how much each tooth is thinned.

For how to build this gearbox from Go, see the [developer notes](../../docs/models/planetarygearbox.md).
