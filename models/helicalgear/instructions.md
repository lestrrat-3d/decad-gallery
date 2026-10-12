# Helical gear construction instructions

`fusion360-gear-generator/spec/helicalgear/instructions.md` defines the gear.
Helical reuses the spur involute dimensions and tooth profile.
It draws one tooth at zero rotation on the lower plane and the same tooth
rotated by the signed Helix Angle on the plane one Thickness above it. Decad
lofts the lower section to the upper section in that order. The angle is the
whole twist across Thickness, not an angle calculated from a lead distance.

## Inputs and invalid cases

`Params` embeds the [spur dimensions](../spurgear/instructions.md): Module,
Teeth, PressureAngle, Thickness, Samples, BoreDiameter, and CapChamfer. It
adds `HelixAngle` in radians. The intended default is Module 1, 17 teeth,
20° pressure angle, 10 mm thickness, 15 samples, and a +14.5° helix. A
negative angle reverses the twist. The Fusion source does not impose a
helix-angle range. This model supports finite, nonzero angles and lets the
geometry operation report angles it cannot build. A zero angle has spur
geometry; callers use `spurgear.Build` for that case. The source's helical
zero-angle fillet sequence awaits Decad support for joined root seams.

The spur input checks reject nonpositive Module or Thickness, fewer than
three teeth or samples, PressureAngle outside `(0, π/2)`, negative BoreDiameter
or CapChamfer, and a bore reaching the root circle. A helical tooth also
rejects the embedded profile case where the base circle lies inside the
root circle. The Fusion helical loft selects the ordinary six-curve tooth
profile only; it has no embedded branch.

`Build` joins the tooth pattern, cuts an optional bore, and calls the shared
spur outer-cap chamfer. The ordinary 8-tooth, Module 1, 2 mm thick, +14.5°
case builds the real lower and upper tooth sketches, their Decad loft, and
the extruded root disc. Each body passes `Tessellate(0.1 mm, VerifyAll)`
separately. Against merged Decad `dc2d041f1523`, all eight tooth Unions,
the source-order 1 mm bore Cut, and both outer cap Chamfers pass the real
gallery checks at 0.1 mm. The completed bored gear has two planar cap faces.
Each `OuterLoopOf(CapStart/End(gear))` selector returns 1,072 edges on its
cap and no bore edges. The source-order fillet selector checks the root
cylinder's straight axial edges before the bore. It finds none on the real
small 8-tooth gear and skips the fillet, as the source requires.

## Required build order

1. Draw the ordinary spur tooth with two fitted involute flanks, a tip arc,
   a root arc, and two radial flank-to-root lines on the bottom plane.
2. Draw the same section on the top plane with every point rotated by the
   signed Helix Angle. Loft bottom to top as one tooth body.
3. Extrude the spur root-circle region through Thickness, pattern the tooth
   around the shaft, and join every copy to the root disc.
4. Search root-cylinder faces for straight axial edges. If any are found,
   apply the spur root fillet radius multiplied by `cos(HelixAngle)`; the
   source skips this step when no edge matches. Cut an optional centered bore,
   then chamfer the completed gear's cap edges without selecting bore edges.
5. Verify one closed Decad solid and its mesh, bounds, tooth count, twist
   direction, and optional bore and chamfer before rendering an image.

The real small gear completes steps 1 through 5. The source-default 17-tooth
gear completes the tooth pattern and passes `VerifyAll` at 0.1 mm against
merged Decad `228ae52cc75c`. Its optional 1 mm bore also passes the
source-order Cut and `VerifyAll` against merged Decad `c3059829064f`. A
completed-gear cap Chamfer on that bored 17-tooth gear passes both outer
loops and `VerifyAll` against merged Decad `5f6f09763897` in 291.86
seconds. The verified mesh keeps one through-hole. A gear with nonempty
root-edge selection still needs a geometry check. The helical build uses
the source's single-tooth loft and circular pattern.
