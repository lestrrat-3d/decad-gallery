# Crank handle

![Crank handle for a 5 mm D-flat shaft: a red arm and a dark handle that turns on its pin](images/hero.png)

A printed hand crank for a D-flat shaft. It is the crank of the [hand-crank siren](../handcranksiren/), which builds
its crank from this model. The arm pinches the shaft: a slit runs through its hub into the D bore, and an M2 bolt across
the hub closes the slit on the shaft. A pin is clamped into the arm's far end by another M2 bolt and nut, and the handle
turns freely on the pin, so your hand does not twist on it as you crank. The pin's head holds the handle on.

The pin prints as two halves cut lengthwise, each lying flat on the bed, so its layers run along the pin and the hand's
force bends it along the printed lines instead of pulling its layers apart. Two small M2 bolts across the pin clamp the
halves together.

The crank fits a shaft that runs in a purchased bearing, because the shaft's diameter equals the bearing's bore. The
default fits the 5 mm shaft of a 605ZZ bearing. A second preset fits the 8 mm shaft of a 608-2RS (or 608ZZ) bearing; it
is the siren's crank. The shaft is part of the mechanism the crank drives and is not in this model.

**Clearance adjustment:** Mixed. The bore's clearance is a parameter; the bolts and nuts keep their nominal size.

## Views

From the side, the handle stands on the arm's far end.

![The crank from the side](images/side.png)

From below, the arm's back face shows the D bore with the slit running from it to the arm's end, and the pin bolt's
head, flush in its pocket. This face goes against the mechanism.

![The crank from below, showing the D bore and the bolt head in the arm's back face](images/back.png)

## The clamp

The arm keeps its full 20 mm width past the shaft. Its sides run straight on, turn into 4.5 mm corners, and close with
one shallow 22 mm end arc, every join smooth, so the hub has no inside corner and no step. The end reaches 13.9 mm
behind the shaft's axis on the 605ZZ crank and 15.4 mm on the 608-2RS crank.

![The arm's hub cut through the clamp bolt: the M2 x 18 bolt's head flush in its pocket in one side face, the bolt crossing the 1 mm slit, and its nut in a hexagonal pocket in the other side face, with the bolt's tip flush with that face](images/clamp.png)

The cut runs along the clamp bolt. A 1 mm slit runs from the arm's end into the D bore and ends in a full round in the
bore's floor. The M2 x 18 bolt crosses the whole arm: its head sits flush in a pocket in one side face, and it runs
across the slit into a nut in a hexagonal pocket in the other side face. Its tip ends flush with that face.
Tightening the bolt closes the slit and pinches the bore on the shaft.

![The arm's hub from the nut's side: the M2 nut in its hexagonal pocket, below the side face, with the bolt's tip flush](images/clamp-nut.png)

The nut's pocket is a hexagon 4.35 mm across its flats and 2 mm deep, open to the side face. Its flats stop the nut
turning, so you tighten the bolt from the head's side alone, and the tightened bolt holds the nut in. The nut ends
below the face. One of the hexagon's diagonals runs along Z: a corner points up and one down, and the flats are its
two sides. The pocket's roof is two short slopes that meet in the upper corner, each 30° above horizontal, so they
print as a bridge and the arm needs no supports and no pause.

![A section across the clamp's nut pocket, half way down its depth: the hexagon with a corner up and one down and its flats on the sides, the M2 nut in it and the bolt through the nut](images/clamp-section.png)

The clamp keeps five listed exceptions to the gallery's rules ([instructions.md](instructions.md), "Retained sharp
corners and edges"):

- The slit's edge in the front face stays sharp. The slit is the working 1 mm clamp gap, and a 0.5 mm chamfer would
  open its mouth to 2 mm.
- The arm between the bore and the clamp bolt's hole is 1.05 mm, under the 1.6 mm wall a press-fit bore asks for. The
  slit makes the hub a clamp, so no pressed ring is loaded there.
- The clamp bolt's head pocket keeps a sharp floor rim. The 0.3 mm head clearance leaves no room to round it.
- The nut pocket keeps its six hexagon corners sharp, to match the nut's sharp corners.
- The pocket's floor round (0.6 mm) is cut at a slant by the layers, which read it as a turn over 45°.

## How the parts go together

![The crank cut through the shaft axis and the pin axis: the blind D bore, the pin's stub in the arm's pocket with the M2 x 10 bolt and nut, the two M2 x 6 cross bolts and nuts across the pin, and the handle round the pin's shank under its head](images/cutaway.png)

The picture cuts the crank through the shaft axis and the pin axis.

- The D bore is blind: 1 mm of arm closes it at the front, and the shaft's end stops there. The bore is 0.05 mm larger
  than the shaft all round, a press fit, and the clamp bolt pinches it on the shaft, so the arm turns with the shaft.
  You can raise that clearance up to 0.5 mm.
- The pin's hexagonal stub sits in a hexagonal pocket in the arm with 0.25 mm of clearance on each flat, so the pin
  cannot turn in the arm. An M2 x 10 bolt from the arm's back face runs into a nut held inside the stub and holds the
  pin in the arm.
- Two M2 x 6 bolts cross the pin, one near each end of the handle. Each head sits flush on a small flat on one half,
  and each bolt runs into a nut in a hex pocket open to the other half's outer flat, so the bolts clamp the halves
  together.
- The handle's bore is 0.4 mm larger than the pin's shank all round, and the handle stands 0.4 mm clear of the arm and of
  the pin's head, so it turns freely. The head is wider than the handle's bore, so the handle cannot come off.

## Exploded

![The crank pulled apart along the pin axis, from the bottom: the M2 x 10 bolt, the arm with the M2 x 18 clamp bolt pulled out of one side and the clamp nut out of the other, the pin bolt's M2 nut, the handle, and the pin at the top with its two M2 x 6 cross bolts pulled out to one side and their two M2 nuts to the other](images/exploded.png)

From the bottom: the M2 x 10 bolt, the arm with the M2 x 18 clamp bolt pulled out of one side and the clamp nut out of
its pocket in the other, the pin bolt's nut, the handle, and the pin with its two cross bolts on one side and their
nuts on the other.

## The split pin

![The pin's two halves pulled apart: the half with the head pockets in front with its two M2 x 6 bolts, the other half behind with its two hex nut pockets, showing its cut face with the half groove for the M2 x 10 bolt and the half pocket for its nut, and the M2 x 10 bolt and nut between them](images/split.png)

The pin's two halves meet on the plane through the pin's axis that lies across the arm, the direction your hand pushes
the handle. That push then loads the two halves side by side, and they do not slide on each other.

- One half carries the cross bolts' head pockets. The other holds their nuts in hex pockets open to its outer flat,
  and each bolt holds its nut there.
- Each nut pocket's corners point across the pin and its flats lie along the pin's axis. That leaves 0.73 mm of pin
  between a corner and the shank's round at the flat, under the 1 mm the gallery asks for. The corners along the axis
  would keep 1.07 mm. [instructions.md](instructions.md) lists this as an exception to confirm.
- The M2 x 10 bolt that holds the pin in the arm runs along the seam. Its hole and its nut's pocket are split between
  the halves, and the clamped halves hold the nut in.

![A section across a cross bolt's nut pocket, half way down its depth: the hexagon with its corners pointing across the pin and its flats along the pin's axis, the M2 nut in it and the bolt through the nut](images/cross-section.png)

![The pin's two halves lying on their cut faces, as they print](images/print.png)

Each half prints lying on its cut face, with no supports.

## Parts

| Image | Name |
| --- | --- |
| <img src="images/parts/arm.png" alt="Arm" width="240"> | Arm (printed) |
| <img src="images/parts/pin.png" alt="Pin" width="240"> | Pin, assembled from its two halves |
| <img src="images/parts/pin-half-heads.png" alt="Pin half with the head pockets" width="240"> | Pin half with the cross bolts' head pockets (printed) |
| <img src="images/parts/pin-half-nuts.png" alt="Pin half with the nut pockets" width="240"> | Pin half with the cross bolts' nut pockets (printed) |
| <img src="images/parts/handle.png" alt="Handle" width="240"> | Handle (printed) |
| <img src="images/parts/bolt.png" alt="M2 x 10 bolt" width="240"> | M2 × 10 socket head cap screw ([`models/m2bolt`](../m2bolt)), 1 |
| <img src="images/parts/cross-bolt.png" alt="M2 x 6 bolt" width="240"> | M2 × 6 socket head cap screw ([`models/m2bolt`](../m2bolt)), 2 |
| <img src="images/parts/clamp-bolt.png" alt="M2 x 18 bolt" width="240"> | M2 × 18 socket head cap screw ([`models/m2bolt`](../m2bolt)), 1, the clamp bolt |
| <img src="images/parts/nut.png" alt="M2 nut" width="240"> | M2 hex nut ([`models/m2nut`](../m2nut)), 4 |

## Sizes

The 605ZZ crank (left) and the 608-2RS crank (right). They differ in the bore and in how far the clamp's end reaches
behind the shaft.

![The 605ZZ crank and the 608-2RS crank side by side](images/sizes.png)

## Parameters

All dimensions are in millimeters. `Params.Validate` rejects a value outside its limit with an error that names it.

| Parameter | 605ZZ (default) | 608-2RS | Limit |
| --- | --- | --- | --- |
| `ShaftDiameter` | 5 | 8 | Above 0; leaves a 1.6 mm wall in the 20 mm arm (at most 16.7) |
| `FlatDepth` | 0.5 | 0.75 | Above 0 and at most a quarter of `ShaftDiameter` |
| `BoreClearance` | 0.05 | 0.05 | 0.05 to 0.5 |
| `Throw` | 55 | 55 | At least 20 |
| `Placement` | none | none | A valid frame |

`Throw` is the distance from the shaft's axis to the pin's axis. The arm is 20 mm wide and 8 mm thick. The pin's shank
is 10.3 mm across and its head 12.9 mm. The handle is 50 mm long and up to 18.6 mm across, with an 11.1 mm bore.

## Printing

Print the arm on its back face and the handle upright on its inner end, with no supports and no pause. Print the pin
as its two halves, each lying on its cut face, with no supports and no pause.

To assemble:

1. Drop the M2 x 10 bolt's nut into its half pocket in one pin half, and lay the other half on it.
2. Set an M2 nut in each hex pocket of the other half, push each M2 x 6 bolt through its head pocket into its nut,
   and tighten it.
3. Slide the handle onto the pin from the stub end.
4. Push the stub into the arm's pocket and drive the M2 x 10 bolt in from the arm's back face.
5. Press the arm onto the shaft, set an M2 nut in the hexagonal pocket in one side of the arm's hub, push the M2 x 18
   clamp bolt in from the other side into it, and tighten it.

For how to build this crank from Go, see the [developer notes](../../docs/models/crankhandle.md). The construction is
in [instructions.md](instructions.md) and [steps.md](steps.md).
