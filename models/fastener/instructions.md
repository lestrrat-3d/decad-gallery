# Bolts and nuts

## Purpose and sources

Construct a bolt or a hex nut from a named size preset held in one catalog. A preset is one size and carries both its
bolt and its nut. The M2 bolt is a socket head cap screw following the dimension drawing of ISO 4762 / JIS B 1176. The
M5 bolt is a hex head bolt following ISO 4017 / JIS B 1180. The nuts follow ISO 4032 / JIS B 1181 (style 1 for M5).

All dimensions are in millimeters. The thread is not modeled. The model gives each part's envelope for fitting parts
and holes; it is not manufacturing geometry, and it does not describe the thread length `b`, the thread tolerance or the
strength class. The standards give some nut values as minimums or ranges; the model uses the exact hexagon corner and
the maximum thickness, a clean envelope for a wrench, a recess or a pocket.

## Preset catalog

The catalog is the only place a bolt or nut dimension lives. Each preset has exactly one bolt head style.

| Symbol | Field | M2 | M5 |
| --- | --- | --- | --- |
| | `Bolt.Head` | socket head cap screw | hex head bolt |
| `ds` | `Bolt.ShankDiameter` | 2 | 5 |
| `k` | `Bolt.HeadHeight` | 2 | 3.5 |
| `u` | `Bolt.TipChamfer` | 0.25 | 0.5 |
| `l` | `Bolt.StandardLengths` | 6, 10, 16, 18, 20, 25 | 16, 20, 25, 30, 35, 40, 45, 50, 60, 70, 80 |
| `dk` | `Bolt.SocketCap.HeadDiameter` | 3.8 | |
| `s` | `Bolt.SocketCap.SocketAcrossFlats` | 1.5 | |
| `t` | `Bolt.SocketCap.SocketDepth` | 1 | |
| `v` | `Bolt.SocketCap.HeadChamferWidth` | 0.2 | |
| | `Bolt.SocketCap.HeadChamferAngle` | 30° | |
| `r` | `Bolt.SocketCap.UnderheadRadius` | 0.1 | |
| | `Bolt.SocketCap.DrillPointAngle` | 118° | |
| | `Bolt.SocketCap.DrillPointBaseRadius` | 0.7 | |
| | `Bolt.SocketCap.PilotDepth` | 0.1 | |
| `s` | `Bolt.Hex.AcrossFlats` | | 8 |
| `dw` | `Bolt.Hex.BearingDiameter` | | 6.88 |
| `d` | `Nut.BoreDiameter` | 2 | 5 |
| `s` | `Nut.AcrossFlats` | 4 | 8 |
| `m` | `Nut.Thickness` | 1.6 | 4.7 |
| `dw` | `Nut.BearingDiameter` | 3.48 | 6.9 |

`e`, the width across corners of a hex head or nut, is `s / cos 30°`: 4.619 for the M2 nut, 9.2376 for the M5 head and
nut. Every hex head and nut chamfers its corners 30° from the face (`HexChamferAngle`).

## API

Package: `models/fastener`.

Catalog. `Names() []string` returns the catalog names in order, a fresh slice each call. `Lookup(name) (Preset, bool)`
returns a deep copy of the preset whose `Name` equals `name` exactly. `Preset{Name, Bolt BoltSize, Nut NutSize}` has
`Validate() error`. `BoltSize` holds `Head HeadStyle` (`HeadSocketCap` or `HeadHex`), the common bolt dimensions,
`SocketCap SocketCapHead` and `Hex HexHead`; the block of the other head style is zero. `HexHead.AcrossCorners()` and
`NutSize.AcrossCorners()` return `AcrossFlats / cos 30°`.

`Preset.Validate` returns an error prefixed `fastener: preset "<name>": ` for each of these: an empty `Name`; a bolt
dimension that is not finite and positive; `TipChamfer` not below `ShankDiameter/2`; empty, unsorted, repeated or
too-short `StandardLengths`; for a socket cap, non-zero `Hex`, a head not wider than the shank plus the fillet, a
chamfer angle of 90° or more or a chamfer taller than the head, socket corners outside the chamfered top face, a pilot
well wider than the socket, a drill point angle of 180° or more or a cone apex below the underside; for a hex head,
non-zero `SocketCap`, a bearing diameter outside `ShankDiameter < dw < s`, or a head no taller than the shank overlap
(1 mm); an unknown head style; a nut that `internal/hexnut` rejects; a nut bore that differs from the shank diameter.

Bolt. `BoltParams{Preset string; Length float64; Placement *r3.Frame}`. A nil Placement selects the local frame: the
origin is the center of the head's underside, the head lies along +Z and the shank along -Z. A hex head's corners lie
on the X axis. A supplied Placement maps that local frame onto it. `DefaultBoltParams()` returns M2, length 10.
`BoltParams.Validate()` rejects an unknown preset, an invalid preset, a nonfinite Length or one not in the preset's
`StandardLengths`, and an invalid Placement; the zero value is invalid. `BuildBolt(ctx, doc, p) (*Bolt, error)` adds
exactly one body to doc. `Bolt` exposes `Body`, `Profile`, `Socket` (socket cap only), `Head` (hex head only) and
`Sketches()`, which returns the non-nil sketches in build order.

Nut. `NutParams{Preset string; Placement *r3.Frame}`. A nil Placement selects the local frame: the origin is the center
of the lower face, +Z runs to the upper face, and the corners lie on the X axis (flats face Y). `DefaultNutParams()`
returns M2. `NutParams.Validate()` rejects an unknown preset, an invalid preset and an invalid Placement.
`BuildNut(ctx, doc, p) (*Nut, error)` adds exactly one body to doc. `Nut` exposes `Body`, `Hexagon`, `LowerChamfer`,
`UpperChamfer` and `Sketches()`.

Both builders reject a nil ctx and a nil doc, then validate, then check `ctx.Err` before adding geometry. Every error
is prefixed `fastener: `. Nothing is added to doc when a check fails.

## Socket head cap screw construction

1. Sketch the axial section on the XZ plane, U as radius and V as height, from the tip center up the axis, around the
   shank, the head and the pilot well, and back down to the axis. The section holds the tip chamfer `u`, the shank, the
   underside, the head side, the top chamfer (width `v`, 30° from the top face), the top face out to the drill point
   base radius, a cylindrical pilot well down to `t + PilotDepth` below the top, and the drill-point cone down to the
   axis.
2. Verify the section is fully constrained, then revolve its one valid profile a full turn about the Z axis.
3. Fillet the one concave circle where the shank meets the underside, with radius `r`.
4. Sketch a regular hexagon on the XY plane centered on the origin, with a flat parallel to X and `s` across the flats.
   Verify it is fully constrained.
5. Extrude the hexagon from the socket floor (`k - t`) to above the top face and cut it from the body. The hexagon's
   flats lie outside the pilot well, so the well is removed above the floor.
6. Transform the finished body to Placement if one is supplied. Require the body to be solid.

The sketches are anchored to their origin. Only the axis and the underside are dimensioned from it; the rest is held by
horizontal, vertical and angle constraints and by dimensions between neighboring points.

## Hex head bolt construction

1. Build the head with `internal/hexnut` and no bore: a hexagonal prism of `s` across flats and height `k` along +Z
   from the underside, with a 30° chamfer from the circle of diameter `dw` on both faces.
2. Sketch the shank's axial section on the XZ plane, U as radius and V as height, from the tip center up the axis: tip
   face, 45° tip chamfer, cylinder wall, and a top that reaches 1 mm into the head so the two bodies overlap. Verify it
   is fully constrained, then revolve its one valid profile a full turn about the Z axis.
3. Join the head and the shank into one solid. Require the bolt to leave exactly one body in the document.
4. Transform the body to Placement if one is supplied. Require the body to be solid.

The head sketch is anchored to its origin by `internal/hexnut`. The shank sketch is anchored to the origin; only the
axis and the underside are dimensioned from it, and the rest is held by horizontal, vertical and dimension constraints
between neighboring points. There is no underhead fillet.

## Hex nut construction

`internal/hexnut` builds the whole nut; this model adds no geometry of its own.

1. Sketch a regular hexagon on XY centered on the origin, with a flat parallel to X and `s` across the flats, and a
   circle of diameter `d` at the origin. Verify the sketch is fully constrained.
2. Extrude the profile with one hole along +Z by `m`.
3. For each face, sketch on XZ a triangle revolved about Z. Its slanted side lies on the chamfer cone, which meets the
   face at the circle of diameter `dw` and rises 30° from the face. The triangle reaches past the face and its outer
   side clears the hexagon's corners, so nothing touches tangentially. Verify it is fully constrained.
4. Revolve each triangle a full turn and cut it from the body.
5. Transform the body to Placement if one is supplied. Require the body to be solid.

Every face stays exact: six flats, two bearing faces, two cones and the bore.

## Representative cases

- Every preset's default bolt and nut build one solid body, with bounds, volume against an independent integral, face
  kinds and `VerifyAll` tessellation, and every sketch fully constrained.
- Every standard length of every preset builds with Z from `-l` to `k`.
- A placed bolt and a placed nut follow their frame, and the first body in the same document keeps its bounds.
- Invalid inputs leave the document empty: an unknown or empty preset, a length not in the preset's list, NaN or
  infinite length, an invalid Placement, a cancelled context, a nil document and a nil context.
- Every catalog preset passes `Preset.Validate`, and each rule of `Validate` has a test that breaks it.
- The M2 and M5 nut bores equal their bolt shank diameters.

## Adding a preset

Append one `Preset` literal to `catalog` in `catalog.go`, with a comment naming its standards. Never add a size without
a cited standard, and never invent dimensions. Then update the catalog table above, the presets table in `README.md`,
the expected values in `TestCatalogValues`, and the build status line in `README.md`. The generic tests cover the new preset without an edit. A new preset gets no
`internal/m<N>fastener` placement layer; consumers that need one are a separate change.
