# Bevel gear Decad construction steps

These steps implement [instructions.md](instructions.md). All lengths are
millimeters. `Build(ctx, doc, Params)` returns one gear body.

1. Validate both tooth counts, the shared shaft angle, and this member's
   section dimensions. Compute both pitch cone angles and the shared pitch
   cone distance. Reject every invalid input before adding a body to the
   document.
2. Resolve the face width. Set the heel section at z=0 and the toe section
   at `z = FaceWidth × cos(γ)`. Their pitch radii are `r` and
   `r × (1 − FaceWidth / R)`. Check that the toe radius and its root circle
   remain positive.
3. Ask the spur gear package to build one complete involute outline in
   each section plane. Scale the toe module by the pitch-radius ratio.
   Require exactly one valid Sketch region in each section. Reusing that
   profile builder keeps the flank sampling and embedded-root trim owned
   by the spur gear model.
4. Loft matching profile edges with `Document.Loft`. The ruled walls taper
   every tooth toward the shared pitch apex. Check one solid, the expected
   section bounds and tooth count, and a `VerifyAll` tessellation before
   adding optional features.
5. Extend the build to the bore, root and cap finishes, toe controls,
   spacing, and spiral trace only after step 4 passes on the first real
   eight-tooth body. Check each added branch through the real Decad body.
6. For the Fusion-derived body, draw one back-cone virtual tooth using the
   real-valued virtual tooth count from the instructions. Place its Sketch
   on the tilted back-cone plane, including any spacing offset, and solve
   the fitted flank and arc profile before asking Decad for the exact
   apex-to-profile loft. This Sketch profile is an independent gate for the
   upstream loft handoff; it does not replace that loft with a shrunken
   section.
7. Draw the six-sided reference blank in an axial Sketch and revolve it
   around the shaft. Check its two flat end faces, three conical faces, and
   `VerifyAll` mesh. This blank is the actual body the patterned teeth must
   later join and the bore must later cut.
8. For a positive bore, extrude one centered circular tool past both axial
   ends and cut the revolved blank. Require the bounded diameter, one solid,
   lower volume, `VerifyAll`, and mesh Euler characteristic zero, which
   distinguishes the through-hole from the uncut blank's characteristic two.
