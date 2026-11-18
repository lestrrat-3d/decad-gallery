# Dovetail joint

## Purpose

Cut a straight sliding dovetail socket into one host body and add the matching tail to another. The two halves
slide together along one direction and cannot be pulled apart across it. The feature makes no part of its own: the
caller passes two host bodies and a frame, and `Apply` returns both hosts finished. All dimensions are in
millimeters and all angles in degrees.

The tail and the socket are built from one `Params`, from one profile, so they cannot drift apart. The four convex
corners of the joint, the tail's two wide-end corners and the socket's two lips, carry a 0.5 mm chamfer. The four
concave corners stay sharp.

## Fit: nominal size, zero gap

This feature builds both halves to exact nominal size. The socket has the same outline as the tail before the
chamfers, so the halves touch along the two flanks, the wide end and the neck's mating plane, with no gap. Each
chamfer sits against a sharp concave corner of the other half and leaves a small triangular void there (see
"Chamfers"). Fit clearance is out of scope.
The user defines it separately. Until then a printed pair will not slide together, and the feature makes no claim
that it does.

## Inputs

| Input | Meaning | Default | Valid range |
| --- | --- | --- | --- |
| `TailWidth` | tail width at its narrow end, the neck where it leaves its host | 10 | 3 or more |
| `Angle` | dovetail angle: how far each flank leans from the depth direction | 10 | 5 to 20 |
| `Length` | joint length along the slide direction | 16 | 5 or more |
| `Depth` | how far the tail stands from its host, and how deep the socket is | 6 | 3 or more |
| `TailOn` | host that carries the tail: `TailOnFirst` or `TailOnSecond` | `TailOnFirst` | the two values |

Below 5 degrees the flanks lean too little to stop the halves pulling apart. Above 20 degrees the tail's wide end is
much wider than its neck and the socket's lips become thin edges. The chamfer sets no new limit: at the smallest
joint (`TailWidth` 3, `Depth` 3) the flank is 3.1 mm long and the mating face beside a lip 1.6 mm wide, both more than
the 0.5 mm setback.

Derived value: the tail's width at its wide end, `TipWidth = TailWidth + 2 * Depth * tan(Angle)`. At the defaults
it is 12.116.

## Frame and hosts

`Apply` takes a right-handed frame with origin O and axes U, V and N = U x V.

- O is the centre of the joint's start on the mating plane.
- U runs across the joint, V runs along the slide direction, and N points from the first host into the second.
- The mating plane is N = 0. The first host lies on the -N side of it and the second on the +N side.
- The joint runs from V = 0 to V = `Length` and is centred on U = 0.

The hosts must fit the joint, and `Apply` refuses hosts that do not:

- Both hosts span V from 0 to `Length` (a through joint, flush with both ends of the joint).
- Both hosts cover U within +-(`TipWidth`/2 + 0.8) so 0.8 mm of host material stays beside the wide end.
- Each host ends at the mating plane.
- The socket's host reaches `Depth` + 0.8 from the mating plane, so 0.8 mm of material stays under the socket floor.
- The tail's host reaches 1.0 mm (`Stock`) from the mating plane, so the tail's root sinks into it.

## Geometry

Profile in the plane of U and the tail direction s*N, where s is +1 for `TailOnFirst` (the tail rises into the second
host) and -1 for `TailOnSecond` (the tail hangs into the first host). Coordinates are (u, v) with v along s*N. With
w = `TailWidth`, W = `TipWidth` and D = `Depth`, the profile is a hexagon with these corners in order:

1. (-w/2, -1), the root of the neck, 1 mm inside the tail's host.
2. (w/2, -1).
3. (w/2, 0), the neck on the mating plane.
4. (W/2, D), the wide end.
5. (-W/2, D).
6. (-w/2, 0).

The sides from corner 2 to 3 and from 6 to 1 are parallel to the depth direction. The flanks from 3 to 4 and from 5 to 6
lean outward by `Angle` from them. The profile is symmetric about U = 0.

The tail is this profile extruded from V = 0 to V = `Length` and joined to the tail's host. The socket cutter is the
same profile extruded from V = -1 to V = `Length` + 1 and cut from the socket's host. The 1 mm extensions make the cut
open through both ends of the host. Only the part of the profile between v = 0 and v = D lies where the hosts meet, so
the cut leaves a socket of exactly the tail's outline, and the join leaves a tail of exactly that outline.

Feature order: tail sketch and body, socket sketch and body, the tail's wide-end chamfer, then the join, then the
cut, then the socket's lip chamfer.

## Chamfers

The edge-finishing rule asks for a 0.5 mm equal-setback chamfer on a convex edge, so the joint uses 0.5 mm. The
chamfer takes 0.5 mm from each of the two faces at the corner, and each runs the whole joint length, from V = 0 to
V = `Length`:

- The tail's two wide-end edges, at U = +-`TipWidth`/2, N = s `Depth`, where the wide-end face meets a flank. The
  tail body is chamfered before it joins its host.
- The socket's two lip edges, at U = +-`TailWidth`/2, N = 0, where the host's mating face meets a flank. The
  socket's host is chamfered after the cut.

The four concave edges stay sharp: the tail's two root edges, where a flank meets the tail's host, and the socket's two
floor edges, at the socket's widest part. A chamfer that meets a sharp concave corner of the other half leaves a small
triangular void of 0.5 mm setback, and the flanks still touch over the rest of their length, so the fit and the grip are
kept. Each chamfer removes `0.5 * 0.5^2 * cos(Angle)` mm squared of profile, 0.123 mm squared at the defaults, along
`Length`, from its host. The tail's host gains the tail's volume less its two chamfers. The socket's host loses the
tail's volume and its two lip chamfers.

The chamfer uses decad's `Chamfer` on the edges, found by their end points.

## Sketch constraints

Each of the two profile sketches is anchored at the centre of the neck on the mating plane (the sketch origin is the
midpoint of the neck's edge). It is constrained with:

- horizontal lines on corners 1 to 2 and 4 to 5, and vertical lines on 2 to 3 and 6 to 1;
- the origin as the midpoint of corners 6 and 3, with those two corners level;
- the neck width, the 1 mm depth of the neck's root, the tail's depth, and the two flank angles.

Each sketch reports `sketch.FullyConstrained` with DOF 0, no redundant constraints and no conflicts. The demo's two
block sketches are constrained the same way: edges horizontal or vertical, the origin at the midpoint of the V = 0
edge, and the width and the length.

## Demo

`BuildDemo` builds two blocks, each `TipWidth` + 8 wide (U), `Length` long (V) and `Depth` + 4 tall (N). The first
spans N from -(`Depth` + 4) to 0 and the second from 0 to `Depth` + 4, at the world frame (U = X, V = Y, N = Z). The
joint is applied between them. Bodies sit in two containers: `Tail Blocks` (the block carrying the tail) and
`Socket Blocks` (the block holding the socket).

The blocks are test fixtures for the joint, not printed parts. Their plain outer edges are not chamfered, and the
rule on edge finishing applies to the caller's own parts. The joint's own chamfers are part of the feature, so
`Apply` cuts them on any host. The wide-end face of the tail is a large planar surface for a long joint (at the
large case, 24.2 mm by 30 mm), and its two long edges are the chamfered ones. Its short edges lie on the fixture
block's end face, which the demo leaves plain.

## Print orientation and corners

Print each block with the slide direction (V) upward. Each layer then shows the dovetail profile as its outline, and no
face of the joint overhangs.

Retained sharp corners: the joint keeps four sharp concave corners in each layer, listed below. The four convex
corners carry the chamfer, so no mating pair has two sharp corners now. In each pair one mate is chamfered and the
other is sharp:

| Pair | Tail block corner | Socket block corner |
| --- | --- | --- |
| Neck | the inside corner where each flank meets the host face at the mating plane, at U = +-`TailWidth`/2 (100 degrees of turn), sharp | the lip, the outward corner at the same place, chamfered 0.5 mm |
| Wide end | the outward corner at the wide end, at U = +-`TipWidth`/2, chamfered 0.5 mm | the inside corner at the socket floor (100 degrees of turn), sharp |

The tests list the tail block's root corners and the socket block's floor corners for `cornercheck`. A concave
corner is kept sharp so a chamfered convex corner meeting it leaves only a small void and the flanks keep their fit.

Each inside corner turns 90 + `Angle` degrees toward the material. The rule rounds an inside corner of more than 45
degrees, unless it is a mating corner kept sharp and listed. Each chamfered outward corner turns 90 + `Angle` degrees
away from the material in two turns of about half that, so it needs no further blunting.

The blocks' outer corners are 90 degrees outward corners.

## Look-alike parts, fasteners and collars

The feature has none. The two blocks differ by shape, and no bought part is involved.

## Validation

Reject a nil context, document, host or invalid frame. Reject `Angle`, `TailWidth`, `Depth` and `Length` outside the
ranges above, a non-finite size and a `TailOn` that is neither value. Reject hosts that do not fit the joint, with the
reason in the error.

## Representative cases

| Case | `TailWidth` | `Angle` | `Length` | `Depth` | `TailOn` |
| --- | --- | --- | --- | --- | --- |
| default | 10 | 10 | 16 | 6 | first |
| tail on second | 10 | 10 | 16 | 6 | second |
| min angle | 10 | 5 | 16 | 6 | first |
| max angle | 10 | 20 | 16 | 6 | first |
| smallest | 3 | 15 | 5 | 3 | first |
| large | 20 | 12 | 30 | 10 | second |
