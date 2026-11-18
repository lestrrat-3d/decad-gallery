# Dovetail joint

**Build status:** Not built yet.

![Two blocks joined by a dovetail](images/hero.png)

A feature is a reusable shape that cuts or adds material on parts that other items own. This one is a straight
sliding dovetail. It adds a tapered tail to one host body and cuts a socket of the same outline into another, so the
two slide together along one direction and cannot be pulled apart across it. Both halves are built from one set of
parameters, so they match exactly.

The halves are built at exact nominal size. The tail and the socket touch with no gap, so a printed pair needs a fit
clearance before it will slide. The clearance is defined separately and is not part of this feature yet.

**Clearance adjustment:** Nominal. There is no clearance parameter.

You can change the tail width at its narrow end (10 mm by default), the dovetail angle (10 degrees, 5 to 20), the
joint length (16 mm), the depth (6 mm) and which block carries the tail.

## Demo blocks

The demo joins two plain blocks, which are test fixtures and not printed parts. Each is 20.1 mm wide, 16 mm long and
10 mm tall at the defaults. The tail rises 6 mm from the first block into the second, which is the tail's socket.

| Block | Image | Holds |
| --- | --- | --- |
| Tail block | ![Tail block](images/parts/tail-block.png) | The tail on its face. Orange in the images. |
| Socket block | ![Socket block](images/parts/socket-block.png) | The socket cut through its length. Blue in the images. |

## Views

The end view looks along the slide direction at the dovetail profile.

![End of the joint](images/end.png)

The rear view shows the other end and side of the joint.

![Rear of the joint](images/rear.png)

The underside view shows the tail block's flat bottom and the socket block's far end.

![Underside of the joint](images/low.png)

## Exploded view

The exploded view pulls the two blocks apart along the slide direction: the tail block 15 mm one way and the socket
block 15 mm the other.

![Exploded dovetail joint: the tail block with its tail, and the socket block with its socket](images/exploded.png)

For the Go API, the build command and the geometry checks, see the [developer notes](../../docs/features/dovetail.md).
