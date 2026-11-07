# M2 hex nut

![M2 hex nut](images/hero.png)

This model is the M2 hex nut that pairs with the [M2 bolt](../m2bolt/). It follows ISO 4032 and JIS B 1181:
4 mm across flats, 1.6 mm thick, a 2 mm bore, and a 30 degree chamfer on every corner of both faces.

The thread is not modeled. The bore is the nominal 2 mm, the same as the bolt's shank.
The hexagon uses its exact corners, 4.619 mm across, where the standards give a smaller minimum.

`DefaultParams` selects the nut in its local frame, with the lower face on the XY plane. `Placement`
positions the lower-face center and the nut's axis in an assembly.

## Views

The top view shows the hexagon, the flat bearing circle and the bore.

![Top of the M2 nut](images/top.png)

The side view shows how the chamfers curve across each flat.

![Side of the M2 nut](images/side.png)

The underside has the same chamfer and bearing circle as the top.

![Underside of the M2 nut](images/low.png)

The nut fits the M2x10 bolt. This image seats it 1 mm from the tip.

![M2 nut on an M2x10 bolt](images/withbolt.png)

The [developer guide](../../docs/models/m2nut.md) documents construction and checks.
The [instructions](instructions.md) own the dimensions; the [steps](steps.md) map them to Decad.
