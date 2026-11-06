# Planetary gearbox

![The default two-stage planetary gearbox, fully assembled, with the ring gear and both end covers cut open to show the planets, carriers, shafts and bearings inside](images/hero.png)

A compact planetary speed reducer. A fixed ring gear with teeth on its inside is the housing, closed at both ends by
an open-frame cover. A small sun gear turns three planet gears that roll around it inside the ring. Each planet turns
on a 608ZZ bearing pressed into a pocket in the gear and rides on a pin of the carrier, a solid triangular plate that
turns slowly. The sun is the input and the carrier is the output.

The default gearbox has a 13-tooth sun, three 29-tooth planets and a 71-tooth ring. One stage slows the output to
1 ÷ 6.46 of the input speed (1 + 71 ÷ 13). The image shows two stages in the same ring: a link shaft joins the first
carrier to the second sun, so two stages slow the output to 1 ÷ 41.8. The ring and both covers are cut open along a
plane through the axis; every other part is shown whole.

## Assembled

The complete two-stage gearbox with nothing cut away, from the output end, from the input end and from the side.

| From the output end | From the input end | From the side |
| --- | --- | --- |
| <img src="images/assembled-output.png" alt="Assembled two-stage gearbox seen from the output end" width="320"> | <img src="images/assembled-input.png" alt="Assembled two-stage gearbox seen from the input end" width="320"> | <img src="images/assembled-side.png" alt="Assembled two-stage gearbox seen from the side" width="320"> |

## Exploded

<img src="images/exploded.png" alt="The two-stage gearbox pulled apart along its axis: input shaft, front bearing and cover, two stages of sun, planets, bearings and carrier with a link shaft between them, the ring, the rear cover and bearing, and the output shaft" width="450">

From the bottom: the input shaft, the front bearing and cover, the first stage (sun and planets, their bearings, the
carrier), the link shaft, the second stage, the ring, the rear cover and its bearing, and the output shaft.

## How it moves

![The two-stage gearbox turning, seen cut open, from the input end and from the output end, with the input and output shaft angles read out underneath](images/motion.gif)

The input shaft and the first sun turn together. The planets roll inside the fixed ring, so each planet turns on its
own bearing while the carrier carries it around the sun. The first carrier turns at 13 ÷ 84 of the sun's speed and
drives the second sun, which drives the second carrier and the output shaft at the same ratio again. The keyway
grooves on each shaft show its turning: the input shaft's grooves sweep round, and the output shaft's creep. The readout
below the panels gives both angles. The animation plays forward and then back, so it has no jump where it repeats. A one-way loop would need the
input to turn 7,056 times, because the output shaft only comes back to its starting angle after that many turns of
the input. It moves the real tessellated bodies rigidly; it is not a force simulation.

## How the input and the output connect

Every shaft is a separate part, 8 mm across, the bore of a 608ZZ. It is a full round cylinder where a bearing sits and
has three keyway grooves, 1.1 mm deep and 2 mm wide and 120° apart, where a sun or a carrier sits, like the hand crank fan's shafts. The sun's bore and the carrier's
hole each have a tooth in each groove, so a shaft turns them without slipping. The input shaft and the output shaft each
have a round collar, 10.4 mm across, next to their bearing. It stands in for the washer a metal build would use, so
the sun or the carrier does not rub on the bearing and the cover. It stops 0.5 mm short of the part it faces. Each link shaft has a collar too, in the 1 mm gap between its carrier and
the next sun, so the sun cannot slide down onto the carrier; it has 0.25 mm of play either side.

- **Input.** The input shaft enters through the front cover's 608ZZ and fills the first sun's bore. It sticks out
  12 mm beyond the front cover's bearing boss, where a motor or a crank attaches.
- **Between stages.** A link shaft runs through the first carrier's hole and fills the second sun's bore, so the
  second sun turns with the first carrier.
- **Output.** The output shaft runs through the last carrier's hole and the rear cover's 608ZZ and sticks out 12 mm
  beyond its boss.

Each pin of the carrier has a 1 mm collar under the plate too, so the planets' bearings stay clear of the plate.

The model does not show a set screw, the screws in the four bolt holes that join the covers and the ring, or the
coupling to a motor. The input and output shafts' round ends, and the groove, are where a coupling or a set screw would grip.

## Parts

Each part is shown once, from the real model. A gearbox has one ring, two covers, two cover bearings and an input
shaft and, in every stage, one sun, three planets, three planet bearings, one carrier and one shaft. The bearing is a
purchased 608ZZ, drawn as its 8 × 22 × 7 mm envelope; the covers and the planets use the same part. Each bearing pocket, in the planets and in the covers, has a 0.5 mm chamfer on its entrance so the bearing can be
pressed into the tight printed seat. The front cover is
the rear cover turned over. Each carrier is the same part. A stage's shaft is the link shaft when another stage
follows it, and the output shaft for the last stage. The default one-stage gearbox has only the output shaft. Every
part is shaped to print on its own, without supports: the carrier plate down with its pins up, the shafts standing on
end or lying down.

| Image | Name |
| --- | --- |
| <img src="images/parts/ring.png" alt="Ring gear" width="240"> | Ring gear |
| <img src="images/parts/cover.png" alt="Cover" width="240"> | Cover (front and rear) |
| <img src="images/parts/sun.png" alt="Sun gear" width="240"> | Sun gear |
| <img src="images/parts/planet.png" alt="Planet gear" width="240"> | Planet gear |
| <img src="images/parts/bearing.png" alt="608ZZ bearing" width="240"> | Bearing (608ZZ) |
| <img src="images/parts/input-shaft.png" alt="Input shaft" width="240"> | Input shaft |
| <img src="images/parts/carrier.png" alt="Carrier" width="240"> | Carrier |
| <img src="images/parts/link-shaft.png" alt="Link shaft" width="240"> | Link shaft |
| <img src="images/parts/output-shaft.png" alt="Output shaft" width="240"> | Output shaft |

You can vary:

- The number of stacked stages in the one ring.
- The tooth counts of the sun, planets and ring, and the number of planets.
- Module and pressure angle.
- Gear, carrier and cover thickness, and the gaps between them.
- How far the input and output shafts reach out of the covers.
- The carrier wall, the bearing pocket clearance and entrance chamfer, the ring wall and the bolt hole size.
- Backlash.
- The position and orientation of the gearbox in space.

The ring must have as many teeth as the sun plus twice the planets, and the sun and ring teeth together must divide
by the planet count so the planets sit at equal angles. Dimensions are in millimeters, and the pressure angle is in
radians. Each pair of meshing gears has 0.1 mm of backlash by default, set by how much each tooth is thinned.

For how to build this gearbox from Go, see the [developer notes](../../docs/models/planetarygearbox.md).
