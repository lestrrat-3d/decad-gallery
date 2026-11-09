# Planetary gearbox

![The default two-stage planetary gearbox, fully assembled, with the ring gear and both end covers cut open to show the planets, carriers, shafts and bearings inside](images/hero.png)

A compact planetary speed reducer. A fixed ring gear with teeth on its inside is the housing, closed at both ends by
an open-frame cover. A small sun gear turns three planet gears that roll around it inside the ring. Each planet turns
on a 605ZZ bearing with 0.1 mm radial clearance in its pocket and rides on a pin of the carrier, a solid triangular plate that
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

Every shaft is a separate part, 4.5 mm across, with 0.25 mm radial clearance in the 5 mm bore of a 605ZZ. It is a full round cylinder where a bearing sits and
has three keyway grooves, 1.1 mm deep and 2 mm wide and 120° apart, where a sun or a carrier sits, like the hand crank fan's shafts. The sun's bore and the carrier's
hole each have a tooth in each groove, so a shaft turns them without slipping. The input shaft and the output shaft each
have a round collar, 9.9 mm across, next to their bearing. It stands in for the washer a metal build would use, so
the sun or the carrier does not rub on the bearing and the cover. It stops 0.5 mm short of the part it faces. Each link shaft has a collar too, in the 1 mm gap between its carrier and
the next sun, so the sun cannot slide down onto the carrier; it has 0.25 mm of play either side.

- **Input.** The input shaft enters through the front cover's 605ZZ and fills the first sun's bore. It sticks out
  12 mm beyond the front cover's bearing boss, where a motor or a crank attaches.
- **Between stages.** A link shaft runs through the first carrier's hole and fills the second sun's bore, so the
  second sun turns with the first carrier.
- **Output.** The output shaft runs through the last carrier's hole and the rear cover's 605ZZ and sticks out 12 mm
  beyond its boss.

Each pin of the carrier has a collar that starts 0.25 mm above the bearing's top face, so the planet bearing stays clear of the plate.

The model does not show a set screw or the coupling to a motor. The input and output shafts' round ends, and the groove, are where a coupling or a set screw would grip.

## Edge finishing

Every large flat face of the ring, the covers and the carriers has its outer edge chamfered by 0.5 mm at 45°, so the
printed parts have no sharp rims to chip or cut. That covers both end faces of the ring, both faces of each cover
including the edges of its three windows, and both faces of each carrier plate. The ring's teeth, the shaft holes, the
bolt holes and the bearing seats keep their shapes, so the gears mesh and the parts fit as before. The chamfer on the ring
follows the mounting ears too.

## Mounting the gearbox

The ring has three mounting ears, spaced 120° apart, to fix the gearbox in place. Each ear is a tab on the ring's outside,
as long as the ring, with a 4 mm bore parallel to the gearbox axis. The bore is a plain hole: the model adds no bolt,
head pocket or nut cavity, so you choose what passes through it. The bore's edge is 3 mm from the covers' outer edge, so a
fastener head up to 10 mm across lies beside the covers and not on them. Set the bore diameter and the number of ears
with `EarBoreDiameter` and `EarCount`. `EarCount = 0` leaves the ring a plain cylinder.

The ring is the fixed part of a planetary gearbox, so the ears carry the whole reaction torque of the output.

## How the housing is bolted

Eight M2 bolts and eight M2 nuts ([`models/m2bolt`](../m2bolt), [`models/m2nut`](../m2nut)) hold the covers to the
ring, four bolts and four nuts per cover. Each bolt head lies flush with its cover's outer face. Each nut sits in a
closed cavity inside the ring wall, so the ring holds the nut and it cannot fall out. The section is cut along the plane
through the axis and one bolt of each cover, in the one-stage gearbox.

<img src="images/fasteners-section.png" alt="Section through one bolt at each end of the housing" width="800">

Each bolt is 6 mm long. It runs from a cover into the end of the ring and never along the whole housing, so the bolt
length stays the same for any number of stages. The front and rear bolts share each hole from opposite ends and their
tips do not meet. The exploded view lifts the nut out of its cavity to show where it sits.

<img src="images/fasteners-exploded.png" alt="Rear cover, nut and bolt pulled apart along one bolt" width="800">

### Printing the ring: insert the nuts by hand

The nuts are not pressed in afterwards. Each nut sits in a cavity the ring walls close on every side except the bolt
hole, so the nuts have to go in while the ring prints. Print the ring standing on its front face, and stop the printer
twice:

1. At 2.8 mm above the bed, the front cavities are open at the top. Drop a nut into each of the four, with a flat of the
   nut against the cavity's flats, then resume.
2. At `L - 1` mm above the bed, where `L` is the ring length (with the default dimensions 14 mm for one stage and `13 K + 1` mm for `K` stages),
   the rear cavities are open at the top. Drop a nut into each of the four, then resume.

The next layers print the 1 mm roofs over the nuts. After that the ring holds all eight nuts by itself.

To assemble, set each cover on its ring end and drive each bolt in from the cover's outer face, through the ring's 1 mm
roof into the nut.

| Part | Pocket | Size |
| --- | --- | --- |
| Bolt head | Round, in the cover's outer face | 4.0 mm wide, 2 mm deep, 1 mm of cover under it |
| Bolt shank | Hole through the cover and the ring | 2.4 mm wide |
| Nut | Closed hex cavity in the ring wall, 1 mm behind the end face | 4.2 mm across flats, 1.8 mm high |

## Parts

Each part is shown once, from the real model. A gearbox has one ring, two covers, two cover bearings and an input
shaft and, in every stage, one sun, three planets, three planet bearings, one carrier and one shaft. The bearing is a
purchased 605ZZ, drawn as its 5 × 14 × 5 mm envelope; the covers and the planets use the same part. Planet pins and shafts are 4.5 mm, with 0.25 mm radial clearance in the bearing bore. Planet and cover bearing pockets have 0.1 mm radial clearance. A planet bearing sits 0.25 mm above its pocket floor and extends 0.25 mm above the planet. Each pocket has a 0.5 mm chamfer on its entrance. The front cover is
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
| <img src="images/parts/bearing.png" alt="605ZZ bearing" width="240"> | Bearing (605ZZ) |
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
- The carrier wall, bearing pocket clearance and entrance chamfer, ring wall and bolt hole size (2.4 mm, the M2 clearance hole).
- The number of mounting ears on the ring (three by default, zero for none) and the diameter of their bore (4 mm).
- Backlash.
- The position and orientation of the gearbox in space.

The ring must have as many teeth as the sun plus twice the planets, and the sun and ring teeth together must divide
by the planet count so the planets sit at equal angles. Dimensions are in millimeters, and the pressure angle is in
radians. Each pair of meshing gears has 0.3 mm of backlash by default, set by how much each tooth is thinned. The tip-to-root clearance is at least `max(0.25 × module, 0.5 mm)`.

For how to build this gearbox from Go, see the [developer notes](../../docs/models/planetarygearbox.md).
