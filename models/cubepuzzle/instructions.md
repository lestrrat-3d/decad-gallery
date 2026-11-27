# Cube Puzzle construction contract

This draft records the checked prototype. Dimensions are in millimeters. The cube has an outer half-width of 30,
a nominal 0.5 gap between layer boundaries, and sixfold symmetry. The hub, 6 centers, 12 edges and 8 corners
make the mechanism. Each center mount also has a printed disk, an M2 × 25 socket-head bolt, an M2 nut, and two
machined metal sleeves. The sleeves hold the disk against the hub while the center turns freely between them.
This model has one fixed size; reject any parameter that attempts to change it. Build it in a document from
`NewDocument`, which sets a 0.01 mm chain tolerance. This is the tolerance used in the verified prototype.

## Layer surfaces

Let `r0 = sqrt(209)`, `r1 = 19`, and `m = 6 / (r1 - r0)`. The axial and radial coordinates of a layer boundary
before rounding are (4, r0) and (10, r1). For offset `o`, use `shift = o * sqrt(1 + m*m)`, `low = 4 + o`,
`high = 10 + o`, `rLow = r0 + (o-shift)/m`, and `rHigh = r1 + (o-shift)/m`. Revolve the polygon
`(low,0), (cap,0), (cap,43), (high,43), (high,rHigh), (low,rLow)` around its axial coordinate. Round the upper
boundary corner by `0.75+o` and the lower boundary corner by `0.75-o`. Use `o=0.25, cap=30` for material and
`o=-0.25, cap=60` for cutting tools. Each rounded layer is a real constrained sketch and a full revolution.

Make the +X/+Y/+Z corner as the intersection of the three positive material layers. Make the +X/+Y edge as the
intersection of those two layers, less the two ±Z cutting layers. Make the +Z center from its material layer less
the four ±X and ±Y cutting layers. Remove a radius-12 sphere at the origin from each block. Their intended fit is
Regular, with at least 0.35 mm proven surface clearance.

Trim the center below z=16. Cut a radius-2.9 bore on +Z through it, then a radius-6.4 counterbore from z=25.6
through the outside face. The center disk is a radius-6, height-4 cylinder at z=26..30, with the M2 clearance hole
and head pocket centered on +Z. The bolt head's top is flush with z=30. The disk clears the moving center by the
Regular fit. Print a corner with its -Z outside face on the bed, an edge with its -X outside face on the bed, the +Z center
with its -Z outside face on the bed, and the hub and disk along +Z. No planar face of a printed part reaches
400 mm²; the geometry check bounds each coplanar set by its mesh rectangle, including the mesh bound.

## Hub and mounts

Make the hub from a radius-11.5 sphere centered at the origin, with flat faces at axial distances ±10.5 on X, Y,
and Z. For each face, use an M2 site whose head is at distance 30 from the origin, direction points inward, length
is 25, and nut cavity starts 19.5 from the head. Use the M2 preset. The nut occupies distances 8.7..10.3 from the
origin and the 2 mm deep hex pocket opens on the hub face at distance 10.5. Its corners lie on the M2 nut cavity
circle; the pocket floor is rounded by 0.5 mm. The bolt hole is 2.6 mm across and has its inner end rounded by the
shared M2 hole tool. Check the bolt is fully inside the hole, the nut is inside the pocket, and each clears the hub.
The six pockets and the holes leave one solid hub. The six 60-degree corners in each of the +Z and -Z pocket sections at z = 9.5 and z = -9.5 match the sharp
nut corners. Their locations are `(r cos(iπ/3), r sin(iπ/3), z)` for `i=0..5`, with `r` equal to the M2 nut cavity
radius. These twelve matching pairs are retained sharp corners.

At each center, model two turned metal sleeves along the same outward axis. The flanged sleeve's axial/radial
outline is `(10.5,1.5), (15.6,1.5), (15.6,3.8), (15.05,3.8), (15.05,3), (10.5,3)`; the tube's outline is
`(15.6,1.5), (26,1.5), (26,2.5), (15.6,2.5)`. Revolve each as a rounded profile. These are custom machined parts,
not printed shafts. Their nominal contact with the hub, disk and one another is part of the mount. Check the moving
center clears both sleeves and the disk by the Regular fit.

## Assembly and checks

Use rotations and copies of each original body. Place one hub, 6 centers, 12 edges and 8 corners, and 6 of each
mount component. Put bodies into named containers by kind. Give each printed part its print orientation and finish
its exposed edges under the model pipeline. Check all sketches for full constraint before use. Verify each final body
mesh. Prove all 26 blocks clear each other, the hub, and mounted components at the assembled pose. Check the +Z layer
against the stationary parts and mounts at 15-degree increments through 90 degrees. Check the center and edge insertion
positions at offsets 1, 2, 3, 4, 5, 8, 12, 20 and 35 mm. Translate each edge along the sum of its two visible
unit axes; translate each center along its face normal. Check edge insertion against the other edges, corners and hub
before the centers are installed. Check center insertion against the other blocks, hub, and installed mounts, leaving
its own disk, tube and bolt out until after insertion.

For the corner extraction witness, translate the +X/+Y/+Z corner by (3,3,3); the point (13,13,7.5) belongs to
that translated corner and the stationary +X/+Y edge. For the edge extraction witness, translate a same-layer edge by
(4.15,0,4.15); the point (8.5,0,18.3) belongs to that translated edge and the +Z center. Establish both using the real
source solids and boundary-clear witness paths, not just a tessellation. These samples and witnesses do not prove every
possible motion path.

List the metal sleeves as custom parts with their profiles and quantities. State how the hub pockets accept their nuts,
how each center and mount is assembled, and how the cube is assembled. The hub has not been physically built or
print-tested. Mark the model not built yet.
