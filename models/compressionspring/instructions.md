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
nonfinite or nonpositive dimensions, outside diameter at or below twice the
wire diameter, nonpositive turns or rate, installed length greater than free
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
