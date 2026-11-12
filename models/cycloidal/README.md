# Cycloidal drive

![The assembled two-disc cycloidal drive, with its housing, eccentric cam, output plate and pins](images/hero.png)

A cycloidal speed reducer. One or two lobed discs ride on an eccentric cam inside a pinless housing. An output
plate carries pins through holes in the discs. The default configuration has one disc; the images show the
two-disc option so both rotors are visible.

The drive has 16 pin positions on a 90 mm circle, 1.5 mm of eccentricity, an 8 mm disc and six output pins on a
50 mm circle.

## Assembled

The two-disc drive from the output end, input end and side.

| From the output end | From the input end | From the side |
| --- | --- | --- |
| <img src="images/assembled-output.png" alt="Two-disc cycloidal drive seen from the output end" width="320"> | <img src="images/assembled-input.png" alt="Two-disc cycloidal drive seen from the input end" width="320"> | <img src="images/assembled-side.png" alt="Two-disc cycloidal drive seen from the side" width="320"> |

## Exploded

![The two-disc cycloidal drive pulled apart along its axis in Decad's derived order: housing, rotor disc, output pins, cam, rotor disc and output plate](images/exploded.png)

## Parts

Each image shows one named body of the two-disc build. The six output pins have the same shape and appear
separately because each pin is its own body.

| Image | Name |
| --- | --- |
| <img src="images/parts/disc-1.png" alt="Cycloidal Disk 1" width="240"> | Cycloidal Disk 1 |
| <img src="images/parts/disc-2.png" alt="Cycloidal Disk 2" width="240"> | Cycloidal Disk 2 |
| <img src="images/parts/housing.png" alt="Housing" width="240"> | Housing |
| <img src="images/parts/cam.png" alt="Eccentric Cam" width="240"> | Eccentric Cam |
| <img src="images/parts/output-plate.png" alt="Output Plate" width="240"> | Output Plate |
| <img src="images/parts/output-pin-1.png" alt="Output Pin 1" width="240"> | Output Pin 1 |
| <img src="images/parts/output-pin-2.png" alt="Output Pin 2" width="240"> | Output Pin 2 |
| <img src="images/parts/output-pin-3.png" alt="Output Pin 3" width="240"> | Output Pin 3 |
| <img src="images/parts/output-pin-4.png" alt="Output Pin 4" width="240"> | Output Pin 4 |
| <img src="images/parts/output-pin-5.png" alt="Output Pin 5" width="240"> | Output Pin 5 |
| <img src="images/parts/output-pin-6.png" alt="Output Pin 6" width="240"> | Output Pin 6 |

You can vary:

- The number of pin positions and the pin circle diameter.
- The eccentricity, the pin diameters and the clearances. Moving custom parts have the Regular fit and pins in
  the plate have the press fit.
- One or two discs, their thickness and the gap between them.
- The cam diameter and the input shaft bore, or a solid cam.
- The number and circle of the output pins.
- The housing wall, base and output plate thickness.
- The chamfer size; it defaults to the required 0.5 mm at 45° on long convex outer rims of the
  rotor discs, housing, cam and output plate, and on both ends of every output pin.
- The position and orientation of the drive in space.

Dimensions are in millimeters.

For how to build this drive from Go, see the
[developer notes](../../docs/models/cycloidal.md).
