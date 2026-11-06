# Helical gear construction instructions

`fusion360-gear-generator/spec/helicalgear/instructions.md` defines the gear.
Helical reuses the spur involute dimensions and tooth profile.
It draws the whole toothed outline of the gear at zero rotation on the lower plane and the same
outline rotated by the Helix Angle on the plane one Thickness above it. Decad
lofts the lower outline to the upper one in that order, and then rounds the root seams. The angle is the
whole twist across Thickness, not an angle calculated from a lead distance.

## Placement

`Params` also carries the spur gear's optional `Placement` (`*r3.Frame`). Its
origin is the reference point on the gear axis, its `U` and `V` axes span the
reference plane that holds the lower cap, and the gear grows along `U × V`.
`Build` constructs the whole gear about the local origin and +Z, then moves the
finished body with one `decad.Transform` from the local frame onto `Placement`.
A nil `Placement` leaves the gear on the world XY plane at the origin. An
invalid frame is rejected before any body is added. The bore and the chamfer are
cut in the local frame, so the offset is applied after the last boolean.

## Inputs and invalid cases

`Params` embeds the [spur dimensions](../spurgear/instructions.md): Module,
Teeth, PressureAngle, Thickness, Samples, BoreDiameter, and CapChamfer. It
adds `HelixAngle` in radians. The intended default is Module 1, 17 teeth,
20° pressure angle, 10 mm thickness, 15 samples, and a +14.5° helix. This
gallery accepts a finite, positive `HelixAngle` from 1° through 45° inclusive.
`Build` rejects angles outside that range before adding bodies. Use
`spurgear.Build` for zero twist. The Fusion source imposes no angle range;
the gallery applies this narrower input range. The default +14.5° gear has
full-body verification; the input range alone does not guarantee that every
angle builds a valid solid.

The spur input checks reject nonpositive Module or Thickness, fewer than
three teeth or samples, PressureAngle outside `(0, π/2)`, negative BoreDiameter
or CapChamfer, and a bore reaching the root circle. A helical tooth also
rejects the embedded profile case where the base circle lies inside the
root circle. The Fusion helical loft selects the ordinary six-curve tooth
profile only; it has no embedded branch.

`Build` lofts the whole outline, rounds the 2N root seams, cuts an optional bore and calls the
shared spur outer-cap chamfer. The ordinary 8-tooth, Module 1, 2 mm thick, +14.5° case and the
17-tooth source-default case build one solid with 16 and 34 root fillets, and the loft passes
`Tessellate(0.1 mm, VerifyAll)`. The loft has no root disc, no tooth pattern and no tooth union.

## Parameters

`Build` writes the spur parameter table with `spurgear.DefineParameters` into `doc.Params()` and the sketch
world's table, and each value the build reads is a derived parameter. Helical adds three, defined after
the spur ones:

| Name | Definition |
| --- | --- |
| `helix_angle` | the `HelixAngle` input, in radians |
| `helix_plane_offset` | `thickness`, the distance from the lower cap to the twisted section |
| `helical_fillet_radius` | `fillet_radius * cos(helix_angle)` |

The twisted section's plane, the fillet radius, the bore tool and the cap chamfer are all read back with
`doc.Param`, so `doc.Bindings()` lists them.

## Sketch constraints

The lower and rotated upper outlines come from `spurgear.BuildTwistedProfile`, which follows the
[spur sketch constraints](../spurgear/instructions.md#sketch-constraints): anchor each outline to its plane
origin, dimension the fitted flank points, and share and constrain the adjoining arc endpoints. The upper
outline has every flank turned by the Helix Angle in sketch coordinates; its plane is not turned. When a bore
is requested, center its tool circle on its plane origin and dimension its diameter. Do not fix every
computed point.

Solve and verify each outline and the optional bore sketch before using its profile. Require
`Sketch.Verify(ctx)` to report `sketch.FullyConstrained`, zero degrees of freedom, no redundant constraints,
and no conflicts.

## Required build order

1. Draw the whole ordinary spur outline on the bottom plane with `spurgear.BuildTwistedProfile` at twist 0. It
   returns the profile and the `2N` root contacts, the right and left root of each tooth.
2. Draw the same outline on the plane `helix_plane_offset` above, with every flank turned by the Helix Angle
   (twist `HelixAngle`) and its own `2N` root contacts. Loft bottom to top as one body with `decad.Loft`.
3. Select the `2N` root seams: for each root contact, the edge with one end at the lower contact at z=0 and
   the other at the matching turned contact at `helix_plane_offset`. Combine the clauses with `Or` and
   require exactly `2N` edges. Fillet them with `helical_fillet_radius` (the spur fillet radius times
   `cos(HelixAngle)`), or skip the fillet when that radius is not positive.
4. Cut an optional centered through-bore, then chamfer the completed gear's outer cap edges without
   selecting bore edges.
5. Verify one closed Decad solid and its mesh, bounds, tooth count, twist direction, the `2N` fillet roles,
   and the optional bore and chamfer before rendering an image.

`BuildTooth` still lofts one tooth section alone, as the earlier pipeline did, for callers that pattern
teeth themselves.
