# Purchased compression spring

Build a reusable nominal representation of a purchased round-wire compression
spring. Keep catalog identity and spring data separate from the installed
length. The caller places the body in an assembly.

The first case is Lee Spring LC 024C 14 M: outside diameter 6.10 mm, wire
diameter 0.61 mm, free length 50.80 mm, 19.64 total turns, 17.64 active
turns, nominal rate 0.49 N/mm, and solid height 12.32 mm. The first installed
length is 43.80 mm. The manufacturer's ends are squared and ground; the
temporary nominal model has constant pitch and unground, planar wire-section
ends. It does not model the manufacturer's end transition or ground faces and
must not be used to prove spring seating. The winding hand is a modeling
choice because the manufacturer chooses it.

The input is the catalog data and installed length in millimeters. Reject
nonfinite or nonpositive dimensions (active turns may be zero, meaning the supplier does not publish them),
outside diameter at or below twice the wire diameter, nonpositive total turns or rate, negative or nonfinite active
turns, active turns greater than total turns, installed length greater than free
length, and installed length at or below solid height. The coil's pitch is
`(installed length - wire diameter) / total turns`. Reject pitch at or below
wire diameter. Return one solid and its fully constrained section sketch.

The coil axis is the sketch V axis, through the origin. The circular wire
section center is `( (outside diameter - wire diameter)/2, 0 )`. Anchor its
center to the origin by horizontal alignment and horizontal distance, and
dimension its diameter. Verify the sketch before consuming its profile.
Use Decad Coil and preserve its full station density. The body's axial extent
is the installed length; its bounding width is the nominal outside diameter.

The catalog's maximum solid load is 19.13 N. Compute nominal force as rate
times compression; this does not establish a working fatigue rating. The
spring is purchased, not printed. Its modeled end contact must not be used as
proof of a flat bearing seat.

The button return uses stainless Lee LC 026D 16 S, with 7.62 mm outside
diameter, 0.66 mm wire, 63.50 mm free length, 18.33 total turns, and 16.33
active turns. Its rate is 0.30 N/mm, solid height is 12.17 mm, and load at
solid is 15.93 N. The same constant-pitch end approximation applies.
The button compresses it from 55.50 to 18.94 mm. Its nominal force rises
from 2.40 to 13.368 N, leaving 6.77 mm above solid at full press.
[Lee Spring product data](https://www.leespring.com/product/compression-spring-lc026d16s-stainless-steel)

The seven-segment click indexer uses stainless Lee LC 022D 03 S: 7.62 mm outside diameter, 0.56 mm wire, 15.88 mm free
length, 5.25 total turns, 3.25 active turns, 0.736 N/mm, 3.10 mm solid height and 9.27 N at solid. The same
constant-pitch end approximation applies. [Lee Spring product data](https://www.leespring.com/product/compression-spring-lc022d03s-stainless-steel)

The metric click-indexer spring is the Lesjofors stock spring Cat.no 2353, stainless steel ISO 6931-1, from the
Lesjofors Stock Spring Catalogue 2025, page 30. Published: 0.60 mm wire, 6.85 mm mean diameter, 6.00 mm minimum inside
diameter, 15 mm free length, 6.3 total turns, 0.86 N/mm, 3/4 of the end coils ground, and 8.9 N at the permitted loaded
length of 4.5 mm. The catalogue does not publish the active turns, so `ActiveTurns` is zero. Three values are computed
here, not published: outside diameter 7.45 mm (mean diameter plus wire), solid height 3.78 mm (wire times total turns,
the catalogue's own definition) and load at solid 9.65 N (0.86 x (15 - 3.78)). The same constant-pitch end
approximation applies. Availability in a given country is not confirmed.
