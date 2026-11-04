# Whorl construction decision

## Approved construction

The reference-layout audit later moved this one-piece whorl in front of the flyer and changed its bore to D16.2.
The current construction dimensions are in `models/spinningwheel/instructions.md`.
The rejected carrier dimensions below record the earlier D8.2 alternative; they are historical.

On2026-10-31 the user chose the one-piece D-bored whorl after reviewing the4–8engineering-day estimate.
Resume the generic Decad extension and require the actual whorl and full default assembly to pass.
Do not substitute a bonded carrier or faceted groove walls.

## Completed native repair

The four-groove whorl's V walls are conical. The previous Decad version rejected its D-shaped through-bore.
The generic repair merged in [Decad PR2928](https://github.com/lestrrat-3d/decad/pull/2928) with passing CI.
Both Gallery modules now pin `v0.0.0-20261030221427-96bc82a990de`.
That version passes the actual Gallery consumer in five parameter cases, including every-part VerifyAll
and a native-cone/analytical-volume check.
A repair restricted to cylindrical steps does not accept these V grooves.

## Rejected bonded-carrier alternative

The rejected alternative would use a permanent two-part whorl with the same outside dimensions and ratios:

- Revolve the existing four-groove outer profile with a stock18 mm circular bore.
- Extrude a carrier with stock18 mm outside diameter, D8.2 bore and flat at stock3.1 mm.
- Carrier width is32*Scale, flush with both whorl ends.
- Permanently bond the full cylindrical interface with a structural adhesive appropriate to the chosen materials.
- The8 mm shaft's D flat still transfers torque positively to the carrier. The bonded interface transfers
  torque from the carrier to the grooved rim. It is a rigid joint, not a bearing or a loose press fit.
- At minimum Scale0.8, the smallest groove root radius10.79 mm leaves1.79 mm radial wall outside the carrier.

The bonded interface has area2*pi*9*32*Scale mm², about1810 mm² at default scale.
This construction adds one part and an adhesive joint. It uses native primary-profile extrusion and
annular Revolve operations; it needs no new kernel through-cut route.
Do not claim a bond strength without a selected material, adhesive, preparation and prototype test.

## Selected one-piece construction

Keep the one-piece D-bored whorl and extend Decad's generic native topology builder for full
revolutions of line-segment meridians, including conical walls, and contained line/arc through-bores.
This removes the carrier and bonded joint, but adds the native session's estimated4–8 engineering days.
Require the actual Gallery producer and full assembly to pass, not only simplified native fixtures.
